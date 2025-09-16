package session

import (
	"context"
	"errors"
	"handyhub-admin-svc/src/clients"
	"handyhub-admin-svc/src/internal/models"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type repository struct {
	collection *mongo.Collection
}

type Repository interface {
	GetByID(ctx context.Context, sessionID string) (*Session, error)
	UpdateActivity(ctx context.Context, sessionID string) error
}

func NewSessionRepository(db *clients.MongoDB, collectionName string) Repository {
	collection := db.Database.Collection(collectionName)
	return &repository{collection: collection}
}

func (r *repository) GetByID(ctx context.Context, sessionID string) (*Session, error) {
	var session Session
	filter := bson.M{"session_id": sessionID}

	err := r.collection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, models.ErrSessionNotFound
		}
		logrus.WithError(err).WithField("session_id", sessionID).Error("Failed to get session")
		return nil, models.ErrDatabaseQuery
	}

	return &session, nil
}

func (r *repository) UpdateActivity(ctx context.Context, sessionID string) error {
	filter := bson.M{
		"session_id": sessionID,
		"is_active":  true,
	}

	update := bson.M{
		"$set": bson.M{
			"last_active_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Error("Failed to update session activity")
		return models.ErrSessionUpdating
	}

	return nil
}
