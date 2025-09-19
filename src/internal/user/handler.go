package user

import (
	"context"
	"errors"
	"handyhub-admin-svc/src/internal/cache"
	"handyhub-admin-svc/src/internal/config"
	"handyhub-admin-svc/src/internal/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	GetAllUsers(c *gin.Context)
	GetUserStats(c *gin.Context)
	ActivateUser(c *gin.Context)
	DeactivateUser(c *gin.Context)
	SuspendUser(c *gin.Context)
}

type handler struct {
	config       *config.Configuration
	service      Service
	cacheService cache.Service
}

func NewHandler(cfg *config.Configuration, service Service, cacheService cache.Service) Handler {
	return &handler{
		config:       cfg,
		service:      service,
		cacheService: cacheService,
	}
}

func (h *handler) GetAllUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.config.App.Timeout)*time.Second)
	defer cancel()

	// Parse query parameters
	req := &GetAllUsersRequest{
		Page:      parseIntParam(c, "page", 1),
		Limit:     parseIntParam(c, "limit", 20),
		Role:      c.Query("role"),
		Status:    c.Query("status"),
		Search:    c.Query("search"),
		SortBy:    c.Query("sortBy"),
		SortOrder: c.Query("sortOrder"),
	}

	logrus.WithFields(logrus.Fields{
		"page":   req.Page,
		"limit":  req.Limit,
		"role":   req.Role,
		"status": req.Status,
		"search": req.Search,
		"sortBy": req.SortBy,
		"order":  req.SortOrder,
	}).Info("GetAllUsers request received")

	// Get admin user info from context
	userID, _ := c.Get("user_id")
	logrus.WithField("admin_user_id", userID).Debug("Admin user accessing GetAllUsers")

	response, err := h.service.GetAllUsers(ctx, req)
	if err != nil {
		logrus.WithError(err).Error("Failed to get all users")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve users",
			"message": err.Error(),
		})
		return
	}

	logrus.WithFields(logrus.Fields{
		"users_returned": len(response.Users),
		"total_count":    response.TotalCount,
		"page":           response.Page,
		"total_pages":    response.TotalPages,
	}).Info("GetAllUsers completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Users retrieved successfully",
	})
}

func parseIntParam(c *gin.Context, param string, defaultValue int) int {
	value := c.Query(param)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"param": param,
			"value": value,
			"error": err,
		}).Warn("Invalid integer parameter, using default")

		return defaultValue
	}
	return parsed
}

func (h *handler) GetUserStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.config.App.Timeout)*time.Second)
	defer cancel()

	logrus.Info("GetUserStats request received")

	// Get admin user info from context
	userID, _ := c.Get("user_id")
	userEmail, _ := c.Get("user_email")

	logrus.WithFields(logrus.Fields{
		"admin_user_id": userID,
		"admin_email":   userEmail,
	}).Debug("Admin user accessing GetUserStats")

	userStats, err := h.cacheService.GetUserStats(ctx)
	if err == nil && userStats != nil {
		logrus.Debug("User statistics retrieved from cache")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    userStats,
			"message": "User statistics retrieved successfully (from cache)",
		})
		return
	}

	stats, err := h.service.GetUserStats(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get user statistics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve user statistics",
			"message": err.Error(),
		})
		return
	}

	h.cacheService.SaveUserStats(ctx, stats)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
		"message": "User statistics retrieved successfully",
	})
}

func (h *handler) ActivateUser(c *gin.Context) {
	h.updateUserStatusHandler(c, StatusActive, "User activated successfully")
}

func (h *handler) DeactivateUser(c *gin.Context) {
	h.updateUserStatusHandler(c, StatusInactive, "User deactivated successfully")
}

func (h *handler) SuspendUser(c *gin.Context) {
	h.updateUserStatusHandler(c, StatusSuspended, "User suspended successfully")
}

func (h *handler) updateUserStatusHandler(c *gin.Context, status, successMessage string) {
	ctx, cancel := context.WithTimeout(c.Request.Context(),
		time.Duration(h.config.App.Timeout)*time.Second)
	defer cancel()

	userID := c.Param("id")
	if userID == "" {
		logrus.Error("User ID is required")
		h.sendErrorResponse(c, http.StatusBadRequest, "User ID is required", "Please provide a valid user ID")
		return
	}

	logrus.WithFields(logrus.Fields{
		"user_id": userID,
		"status":  status,
	}).Info("Updating user status")

	err := h.executeStatusUpdate(ctx, userID, status)

	if err != nil {
		h.handleStatusUpdateError(c, userID, status, err)
		return
	}

	logrus.WithFields(logrus.Fields{
		"user_id": userID,
		"status":  status,
	}).Info("User status updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": successMessage,
	})
}

func (h *handler) executeStatusUpdate(ctx context.Context, userID, status string) error {
	switch status {
	case StatusActive:
		return h.service.ActivateUser(ctx, userID)
	case StatusInactive:
		return h.service.DeactivateUser(ctx, userID)
	case StatusSuspended:
		return h.service.SuspendUser(ctx, userID)
	default:
		return models.ErrInvalidUserStatus
	}
}

func (h *handler) handleStatusUpdateError(c *gin.Context, userID, status string, err error) {
	logrus.WithError(err).WithFields(logrus.Fields{
		"user_id": userID,
		"status":  status,
	}).Error("Failed to update user status")

	switch {
	case errors.Is(err, models.ErrUserNotFound):
		h.sendErrorResponse(c, http.StatusNotFound, "User not found", "No user found with the provided ID")
	case errors.Is(err, models.ErrInvalidParams):
		h.sendErrorResponse(c, http.StatusBadRequest, "Invalid user ID", "Please provide a valid user ID")
		h.sendErrorResponse(c, http.StatusBadRequest, "Invalid user ID", "Please provide a valid user ID")
	default:
		h.sendErrorResponse(c, http.StatusInternalServerError, "Failed to update user status", err.Error())
	}
}

func (h *handler) sendErrorResponse(c *gin.Context, statusCode int, error, message string) {
	c.JSON(statusCode, gin.H{
		"error":   error,
		" ":       false,
		"message": message,
	})
}
