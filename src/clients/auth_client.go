package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"handyhub-admin-svc/src/internal/config"
	"handyhub-admin-svc/src/internal/models"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// AuthClient handles communication with auth service
type AuthClient struct {
	baseURL    string
	httpClient *http.Client
	channel    *amqp.Channel
	cfg        *config.MessagingConfig
}

// NewAuthClient creates new auth service client
func NewAuthClient(cfg *config.Configuration, channel *amqp.Channel) *AuthClient {
	return &AuthClient{
		baseURL: cfg.ExternalServices.AuthService.URL,
		channel: channel,
		cfg:     &cfg.Messaging,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ExternalServices.AuthService.Timeout) * time.Second,
		},
	}
}

// GetSessionById retrieves session info from auth service
func (c *AuthClient) GetSessionById(ctx context.Context, sessionID string) (*models.Session, error) {
	url := fmt.Sprintf("%s/session/%s", c.baseURL, sessionID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call auth service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, models.ErrSessionNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth service returned status: %d", resp.StatusCode)
	}

	var response struct {
		Session *models.Session `json:"session"`
		Status  string          `json:"status"`
		Message string          `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Session, nil
}

// PublishActivity publishes session activity message to RabbitMQ
func (c *AuthClient) PublishActivity(userID, sessionID, serviceName, action string) error {
	return c.PublishActivityWithDetails(userID, sessionID, serviceName, action, "", "")
}

// PublishActivityWithDetails publishes session activity with IP and UserAgent
func (c *AuthClient) PublishActivityWithDetails(userID, sessionID, serviceName, action, ipAddress, userAgent string) error {
	message := models.ActivityMessage{
		UserID:      userID,
		SessionID:   sessionID,
		ServiceName: serviceName,
		Action:      action,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Timestamp:   time.Now(),
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal activity message: %w", err)
	}

	err = c.channel.Publish(
		c.cfg.RabbitMQ.Exchange,
		c.cfg.Queues.UserActivity.RoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)

	if err != nil {
		logrus.WithError(err).Error("Failed to publish activity message")
		return fmt.Errorf("failed to publish activity message: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":     userID,
		"session_id":  sessionID,
		"service":     serviceName,
		"action":      action,
		"exchange":    c.cfg.RabbitMQ.Exchange,
		"routing_key": c.cfg.Queues.UserActivity.RoutingKey,
	}).Debug("Activity message published")

	return nil
}
