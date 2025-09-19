package middleware

import (
	"context"
	"errors"
	"fmt"
	"handyhub-admin-svc/src/clients"
	"handyhub-admin-svc/src/internal/cache"
	"handyhub-admin-svc/src/internal/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// Claims represents JWT token claims
type Claims struct {
	UserID    string `json:"userId"`
	SessionID string `json:"sessionId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"tokenType"`
	jwt.RegisteredClaims
}

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	jwtSecret    string
	cacheService cache.Service
	authClient   *clients.AuthClient
}

const (
	redisKeyPattern = "session:%s:%s" // session:userID:sessionID
)

// NewAuthMiddleware creates new auth middleware with AuthClient
func NewAuthMiddleware(jwtSecret string, cacheService cache.Service, authClient *clients.AuthClient) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:    jwtSecret,
		cacheService: cacheService,
		authClient:   authClient,
	}
}

// RequireAuth validates JWT token and session
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token is required"})
			c.Abort()
			return
		}

		claims, err := m.validateJWTToken(token)
		if err != nil {
			logrus.WithError(err).Error("JWT token validation failed")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		isValid, err := m.validateSession(c.Request.Context(), claims.SessionID, claims.UserID)
		if err != nil {
			logrus.WithError(err).Error("Session validation failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Session validation error"})
			c.Abort()
			return
		}

		if !isValid {
			logrus.WithField("session_id", claims.SessionID).Warn("Session is invalid or expired")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired - please login again"})
			c.Abort()
			return
		}

		// Get route action for activity tracking
		action := m.getRouteAction(c)
		m.publishActivity(claims.UserID, claims.SessionID, c.ClientIP(), c.Request.UserAgent(), action)
		m.setUserContext(c, claims)

		logrus.WithFields(logrus.Fields{
			"user_id": claims.UserID, "session_id": claims.SessionID, "action": action,
		}).Debug("User authenticated successfully")

		c.Next()
	}
}

// RequireAdminRights checks if user has admin privileges
func (m *AuthMiddleware) RequireAdminRights() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoleInterface, exists := c.Get("user_role")
		if !exists {
			logrus.Error("User role not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		userRole, ok := userRoleInterface.(string)
		if !ok {
			logrus.Error("Invalid user role format")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role format"})
			c.Abort()
			return
		}

		if userRole != "admin" {
			userID, _ := c.Get("user_id")
			logrus.WithFields(logrus.Fields{
				"user_id": userID, "user_role": userRole,
			}).Warn("User attempted to access admin endpoint without admin privileges")

			c.JSON(http.StatusForbidden, gin.H{"error": "Access forbidden - admin privileges required"})
			c.Abort()
			return
		}

		userID, _ := c.Get("user_id")
		logrus.WithField("user_id", userID).Debug("Admin access granted")
		c.Next()
	}
}

// extractToken extracts JWT token from Authorization header
func (m *AuthMiddleware) extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(authHeader, "Bearer ")
}

// validateJWTToken parses and validates JWT token
func (m *AuthMiddleware) validateJWTToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.TokenType != "access" {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// validateSession checks session validity via AuthClient API
func (m *AuthMiddleware) validateSession(ctx context.Context, sessionID, userID string) (bool, error) {
	key := fmt.Sprintf(redisKeyPattern, userID, sessionID)

	// Try cache first
	session, err := m.cacheService.GetActiveSession(ctx, key)
	if err == nil && session != nil {
		m.cacheService.UpdateSessionActivity(ctx, key)
		return true, nil
	}

	// Call auth service API
	authSession, err := m.authClient.GetSessionById(ctx, sessionID)
	if err != nil {
		return false, err
	}

	// Check session validity
	if !authSession.IsActive || authSession.LogoutAt != nil || time.Now().After(authSession.ExpiresAt) {
		return false, nil
	}

	// Cache session and update activity
	m.cacheService.CacheActiveSession(ctx, authSession)
	return true, nil
}

// getRouteAction gets action name from route context
func (m *AuthMiddleware) getRouteAction(c *gin.Context) string {
	if routeName, exists := c.Get("route_name"); exists {
		if name, ok := routeName.(string); ok {
			return name
		}
	}
	return "unknown_action"
}

// publishActivity publishes session activity to RabbitMQ
func (m *AuthMiddleware) publishActivity(userID, sessionID, ipAddress, userAgent, action string) {
	err := m.authClient.PublishActivityWithDetails(
		userID, sessionID, models.ServiceAdminAuth, action, ipAddress, userAgent)
	if err != nil {
		logrus.WithError(err).Warn("Failed to publish activity message")
	}
}

// setUserContext stores user info in context
func (m *AuthMiddleware) setUserContext(c *gin.Context, claims *Claims) {
	c.Set("user_id", claims.UserID)
	c.Set("session_id", claims.SessionID)
	c.Set("user_email", claims.Email)
	c.Set("user_role", claims.Role)
}
