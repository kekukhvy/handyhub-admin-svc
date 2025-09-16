package user

import (
	"context"
	"handyhub-admin-svc/src/clients"
	"handyhub-admin-svc/src/internal/models"
	"math"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	regexKey   = "$regex"
	optionsKey = "$options"
)

type Repository interface {
	GetAllUsers(ctx context.Context, req *GetAllUsersRequest) ([]*User, int64, error)
	GetUserStats(ctx context.Context) (*models.Stats, error)
}

type userRepository struct {
	Collection mongo.Collection
}

func NewUserRepository(mongoClient *clients.MongoDB, collectionName string) Repository {
	collection := *mongoClient.Database.Collection(collectionName)
	return &userRepository{
		Collection: collection,
	}
}

func (r *userRepository) GetAllUsers(ctx context.Context, req *GetAllUsersRequest) ([]*User, int64, error) {
	collection := r.Collection

	// Build filter
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}

	if req.Role != "" {
		filter["role"] = req.Role
	}

	if req.Status != "" {
		filter["status"] = req.Status
	}

	if req.Search != "" {
		filter["$or"] = []bson.M{
			{"first_name": bson.M{regexKey: req.Search, optionsKey: "i"}},
			{"last_name": bson.M{regexKey: req.Search, optionsKey: "i"}},
			{"email": bson.M{regexKey: req.Search, optionsKey: "i"}},
		}
	}

	// Count total documents
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		logrus.WithError(err).Error("Failed to count users")
		return nil, 0, err
	}

	skip := (req.Page - 1) * req.Limit

	// Find options
	opts := options.Find().
		SetLimit(int64(req.Limit)).
		SetSkip(int64(skip)).
		SetSort(bson.M{"created_at": -1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		logrus.WithError(err).Error("Failed to find users")
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []*User
	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			logrus.WithError(err).Error("Failed to decode user")
			continue
		}
		users = append(users, &user)
	}

	if err := cursor.Err(); err != nil {
		logrus.WithError(err).Error("Cursor error")
		return nil, 0, err
	}

	logrus.WithFields(logrus.Fields{
		"count": len(users),
		"total": totalCount,
		"page":  req.Page,
		"limit": req.Limit,
	}).Debug("Retrieved users successfully")

	return users, totalCount, nil
}

func (r *userRepository) GetUserStats(ctx context.Context) (*models.Stats, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	pipeline := mongo.Pipeline{
		// Match only non-deleted users
		{{"$match", bson.D{{"deleted_at", bson.D{{"$exists", false}}}}}},

		// Add computed fields for easier grouping
		{{"$addFields", bson.D{
			{"isNewThisMonth", bson.D{{"$gte", bson.A{"$created_at", startOfMonth}}}},
			{"isFromLastMonth", bson.D{{"$lt", bson.A{"$created_at", startOfMonth}}}},
		}}},

		// Use facet to calculate all stats in one aggregation
		{{"$facet", bson.D{
			// Current stats
			{"currentStats", mongo.Pipeline{
				{{"$group", bson.D{
					{"_id", nil},
					{"total", bson.D{{"$sum", 1}}},
					{"active", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$status", StatusActive}}}, 1, 0}}}}}},
					{"inactive", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$status", StatusInactive}}}, 1, 0}}}}}},
					{"suspended", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$status", StatusSuspended}}}, 1, 0}}}}}},
					{"specialists", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$role", RoleExecutor}}}, 1, 0}}}}}},
					{"clients", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$role", RoleClient}}}, 1, 0}}}}}},
					{"newThisMonth", bson.D{{"$sum", bson.D{{"$cond", bson.A{"$isNewThisMonth", 1, 0}}}}}},
				}}},
			}},

			// Previous month stats for growth calculation
			{"previousStats", mongo.Pipeline{
				{{"$match", bson.D{{"isFromLastMonth", true}}}},
				{{"$group", bson.D{
					{"_id", nil},
					{"total", bson.D{{"$sum", 1}}},
					{"active", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$status", StatusActive}}}, 1, 0}}}}}},
					{"specialists", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$role", RoleExecutor}}}, 1, 0}}}}}},
					{"clients", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$role", RoleClient}}}, 1, 0}}}}}},
				}}},
			}},
		}}},
	}

	cursor, err := r.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		logrus.WithError(err).Error("Failed to execute aggregation for user stats")
		return nil, err
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		logrus.Error("No results from user stats aggregation")
		return &models.Stats{}, nil
	}

	var result struct {
		CurrentStats  []models.StatsResult `bson:"currentStats"`
		PreviousStats []models.StatsResult `bson:"previousStats"`
	}

	if err := cursor.Decode(&result); err != nil {
		logrus.WithError(err).Error("Failed to decode aggregation results")
		return nil, err
	}

	// Extract current stats
	var currentStats models.StatsResult
	if len(result.CurrentStats) > 0 {
		currentStats = result.CurrentStats[0]
	}

	// Extract previous stats for growth calculation
	var previousStats models.StatsResult
	if len(result.PreviousStats) > 0 {
		previousStats = result.PreviousStats[0]
	}

	// Calculate growth rates
	growth := &models.GrowthStats{
		Total:       r.calculatePercentageGrowth(previousStats.Total, currentStats.Total),
		Active:      r.calculatePercentageGrowth(previousStats.Active, currentStats.Active),
		Specialists: r.calculatePercentageGrowth(previousStats.Specialists, currentStats.Specialists),
		Clients:     r.calculatePercentageGrowth(previousStats.Clients, currentStats.Clients),
	}

	stats := &models.Stats{
		Total:        currentStats.Total,
		Active:       currentStats.Active,
		Inactive:     currentStats.Inactive,
		Specialists:  currentStats.Specialists,
		Clients:      currentStats.Clients,
		Suspended:    currentStats.Suspended,
		NewThisMonth: currentStats.NewThisMonth,
		Growth:       growth,
	}

	logrus.WithFields(logrus.Fields{
		"total":        stats.Total,
		"active":       stats.Active,
		"inactive":     stats.Inactive,
		"specialists":  stats.Specialists,
		"clients":      stats.Clients,
		"suspended":    stats.Suspended,
		"newThisMonth": stats.NewThisMonth,
		"growth":       growth,
	}).Debug("Retrieved user stats successfully using aggregation")

	return stats, nil
}

func (r *userRepository) calculatePercentageGrowth(previous, current int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0 // 100% growth from zero
		}
		return 0.0
	}

	growth := float64(current-previous) / float64(previous) * 100
	// Round to 1 decimal place
	return math.Round(growth*10) / 10
}
