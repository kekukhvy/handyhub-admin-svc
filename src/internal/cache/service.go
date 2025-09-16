package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"handyhub-admin-svc/src/internal/config"
	"handyhub-admin-svc/src/internal/models"
	"handyhub-admin-svc/src/internal/session"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Service interface {
	GetActiveSession(ctx context.Context, key string) (*session.Session, error)
	UpdateSessionActivity(ctx context.Context, key string) error
	CacheActiveSession(ctx context.Context, session *session.Session) error
	SaveUserStats(ctx context.Context, stats *models.Stats) error
	GetUserStats(ctx context.Context) (*models.Stats, error)
}

type cacheService struct {
	client *redis.Client
	cfg    *config.CacheConfig
}

func NewCacheService(client *redis.Client, cfg *config.Configuration) Service {
	return &cacheService{
		client: client,
		cfg:    &cfg.Cache}
}

func (c *cacheService) GetActiveSession(ctx context.Context, key string) (*session.Session, error) {
	logrus.WithField("key", key).Debug("Getting active session from cache")

	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logrus.WithField("key", key).Debug("Session not found in cache")
			return nil, nil // Not an error, just not found
		}
		logrus.WithError(err).WithField("key", key).Error("Failed to get session from cache")
		return nil, models.ErrRedisGet
	}

	var session session.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		logrus.WithError(err).WithField("key", key).Error("Failed to unmarshal session from cache")
		return nil, models.ErrRedisGet
	}

	logrus.WithField("key", key).Debug("Session retrieved from cache successfully")
	return &session, nil
}

func (c *cacheService) UpdateSessionActivity(ctx context.Context, key string) error {
	logrus.WithField("key", key).Debug("Updating session activity in cache")

	// Get current session
	session, err := c.GetActiveSession(ctx, key)
	if err != nil || session == nil {
		return err
	}

	// Update last active time
	session.LastActiveAt = time.Now()

	// Re-cache with updated time and extended TTL
	data, err := json.Marshal(session)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal session for activity update")
		return models.ErrRedisSet
	}

	extendedTTL := time.Duration(c.cfg.SessionExpirationMinutes) * time.Minute
	err = c.client.Set(ctx, key, data, extendedTTL).Err()
	if err != nil {
		logrus.WithError(err).WithField("key", key).Error("Failed to update session activity")
		return models.ErrRedisSet
	}

	logrus.WithField("key", key).Debug("Session activity updated successfully")
	return nil
}

func (c *cacheService) CacheActiveSession(ctx context.Context, session *session.Session) error {
	key := fmt.Sprintf("session:%s:%s", session.SessionID, session.SessionID)

	data, err := json.Marshal(session)
	if err != nil {
		logrus.WithError(err).WithField("session_id", session.SessionID).Error("Failed to marshal session for cache")
		return models.ErrRedisSet
	}

	expiration := time.Until(session.LastActiveAt.Add(time.Minute * time.Duration(c.cfg.SessionExpirationMinutes)))
	if expiration <= 0 {
		logrus.WithField("session_id", session.SessionID).Warn("Session already expired, not caching")
		return nil
	}

	err = c.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		logrus.WithError(err).WithField("session_id", session.SessionID).Error("Failed to cache session")
		return models.ErrRedisSet
	}

	logrus.WithField("session_id", session.SessionID).Debug("Session cached successfully")
	return nil
}

func (c *cacheService) SaveUserStats(ctx context.Context, stats *models.Stats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal user stats for cache")
		return models.ErrRedisSet
	}
	expiration := time.Until(time.Now().Add(time.Minute * time.Duration(c.cfg.UsetStatExpirationMinutes)))
	err = c.client.Set(ctx, c.cfg.UserStatKey, data, expiration).Err()
	if err != nil {
		logrus.WithError(err).Error("Failed to cache stats")
		return models.ErrRedisSet
	}
	return nil
}

func (c *cacheService) GetUserStats(ctx context.Context) (*models.Stats, error) {

	data, err := c.client.Get(ctx, c.cfg.UserStatKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logrus.Debug("User stats not found in cache")
			return nil, nil // Not an error, just not found
		}
		logrus.WithError(err).Error("Failed to get user stats from cache")
		return nil, models.ErrRedisGet
	}

	var stats models.Stats
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal user stats from cache")
		return nil, models.ErrRedisGet
	}

	logrus.Debug("User stats retrieved from cache successfully")
	return &stats, nil
}
