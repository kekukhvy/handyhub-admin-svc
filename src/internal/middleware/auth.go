package middleware

import (
	"context"
	"errors"
	"fmt"
	"handyhub-admin-svc/src/internal/cache"
	"handyhub-admin-svc/src/internal/session"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
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

// Session represents user session from database

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	jwtSecret    string
	cacheService cache.Service
	collection   *mongo.Collection
	sessionRepo  session.Repository
}

const (
	redisKeyPattern = "session:%s:%s" // session:userID:sessionID
)

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtSecret string, cacheService cache.Service, sessionRepo session.Repository) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:    jwtSecret,
		cacheService: cacheService,
		sessionRepo:  sessionRepo,
	}
}

// RequireAuth validates JWT token and session
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization token is required",
			})
			c.Abort()
			return
		}

		claims, err := m.validateJWTToken(token)
		if err != nil {
			logrus.WithError(err).Error("JWT token validation failed")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		//Validate session from Redis/MongoDB
		isValidSession, err := m.validateSession(c.Request.Context(), claims.SessionID, claims.UserID)
		if err != nil {
			logrus.WithError(err).Error("Session validation failed")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Session validation error",
			})
			c.Abort()
			return
		}

		if !isValidSession {
			logrus.WithField("session_id", claims.SessionID).Warn("Session is invalid or expired")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Session expired - please login again",
			})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		logrus.WithFields(logrus.Fields{
			"user_id":    claims.UserID,
			"session_id": claims.SessionID,
			"user_role":  claims.Role,
		}).Debug("User authenticated successfully")

		c.Next()
	}
}

// RequireAdminRights checks if user has admin privileges
func (m *AuthMiddleware) RequireAdminRights() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user role from context (set by RequireAuth middleware)
		userRoleInterface, exists := c.Get("user_role")
		if !exists {
			logrus.Error("User role not found in context - ensure RequireAuth middleware runs first")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		userRole, ok := userRoleInterface.(string)
		if !ok {
			logrus.Error("Invalid user role format")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user role format",
			})
			c.Abort()
			return
		}

		// Check if user has admin role
		if userRole != "admin" {
			userID, _ := c.Get("user_id")
			logrus.WithFields(logrus.Fields{
				"user_id":   userID,
				"user_role": userRole,
			}).Warn("User attempted to access admin endpoint without admin privileges")

			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access forbidden - admin privileges required",
			})
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
	if authHeader == "" {
		logrus.Error("Authorization header missing")
		return ""
	}

	// Extract token from "Bearer <token>" format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		logrus.Error("Invalid authorization header format")
		return ""
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		logrus.Error("Empty token")
		return ""
	}

	return token
}

// validateJWTToken парses and validates JWT token (checks signature and expiration)
func (m *AuthMiddleware) validateJWTToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		//verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		//JWT library automatically checks expiration
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token expired")
		}
		return nil, errors.New("invalid token")
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Check token type (should be access token)
	if claims.TokenType != "access" {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}

// validateSession checks session validity in Redis first, then MongoDB fallback
func (m *AuthMiddleware) validateSession(ctx context.Context, sessionID, userID string) (bool, error) {
	key := fmt.Sprintf(redisKeyPattern, userID, sessionID)
	session, err := m.cacheService.GetActiveSession(ctx, key)
	if err == nil && session != nil {
		logrus.Info("Session from cache", session)
		m.cacheService.UpdateSessionActivity(ctx, key)
		m.sessionRepo.UpdateActivity(ctx, sessionID)
		return true, nil
	}

	session, err = m.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return false, err
	}

	// Check if session is active and not expired
	if !session.IsActive {
		logrus.WithField("session_id", sessionID).Warn("Session is not active")
		return false, nil
	}

	if session.LogoutAt != nil {
		logrus.WithField("session_id", sessionID).Warn("Session has logout timestamp")
		return false, nil
	}

	if time.Now().After(session.ExpiresAt) {
		logrus.WithField("session_id", sessionID).Warn("Session has expired")
		return false, nil
	}

	session.LastActiveAt = time.Now()
	m.sessionRepo.UpdateActivity(ctx, sessionID)
	m.cacheService.CacheActiveSession(ctx, session)

	logrus.WithField("session_id", sessionID).Debug("Session validated from MongoDB")
	return true, nil
}
