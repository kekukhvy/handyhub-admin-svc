package user

import (
	"context"
	"handyhub-admin-svc/src/clients"
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
	GetUserStats(ctx context.Context) (*Stats, error)
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

func (r *userRepository) GetUserStats(ctx context.Context) (*Stats, error) {

	baseFilter := bson.M{"deleted_at": bson.M{"$exists": false}}

	total, err := r.countUsers(ctx, baseFilter)
	if err != nil {
		return nil, err
	}

	active, err := r.countActiveUsers(ctx)
	if err != nil {
		return nil, err
	}

	inActive, err := r.countInActiveUsers(ctx)
	if err != nil {
		return nil, err
	}

	specialists, err := r.countUsersByRole(ctx, RoleExecutor)
	if err != nil {
		return nil, err
	}

	clients, err := r.countUsersByRole(ctx, RoleClient)
	if err != nil {
		return nil, err
	}

	suspended, err := r.countUsersByStatus(ctx, StatusSuspended)
	if err != nil {
		return nil, err
	}

	newThisMonth, err := r.countNewUsersThisMonth(ctx)
	if err != nil {
		return nil, err
	}

	return &Stats{
		Total:        total,
		Active:       active,
		Inactive:     inActive,
		Specialists:  specialists,
		Clients:      clients,
		Suspended:    suspended,
		NewThisMonth: newThisMonth,
	}, nil
}

func (r *userRepository) countUsers(ctx context.Context, filter bson.M) (int64, error) {
	count, err := r.Collection.CountDocuments(ctx, filter)
	if err != nil {
		logrus.WithError(err).Error("Failed to count users")
		return 0, err
	}
	return count, nil
}

func (r *userRepository) countActiveUsers(ctx context.Context) (int64, error) {
	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
		"status":     StatusActive,
	}
	return r.countUsers(ctx, filter)
}

func (r *userRepository) countInActiveUsers(ctx context.Context) (int64, error) {
	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
		"status":     StatusInactive,
	}
	return r.countUsers(ctx, filter)
}

func (r *userRepository) countUsersByRole(ctx context.Context, role string) (int64, error) {
	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
		"role":       role,
	}
	return r.countUsers(ctx, filter)
}

func (r *userRepository) countUsersByStatus(ctx context.Context, status string) (int64, error) {
	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
		"status":     status,
	}
	return r.countUsers(ctx, filter)
}

func (r *userRepository) countNewUsersThisMonth(ctx context.Context) (int64, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
		"created_at": bson.M{"$gte": startOfMonth},
	}
	return r.countUsers(ctx, filter)
}
