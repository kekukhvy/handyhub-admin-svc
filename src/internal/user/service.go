package user

import (
	"context"
	"handyhub-admin-svc/src/internal/config"
	"handyhub-admin-svc/src/internal/models"
	"math"

	"github.com/sirupsen/logrus"
)

type Service interface {
	GetAllUsers(ctx context.Context, req *GetAllUsersRequest) (*GetAllUsersResponse, error)
	GetUserStats(ctx context.Context) (*models.Stats, error)
}

type userService struct {
	userRepository Repository
	cfg            *config.Configuration
}

func NewUserService(userRepository Repository, cfg *config.Configuration) Service {
	return &userService{
		userRepository: userRepository,
		cfg:            cfg,
	}
}

func (s *userService) GetAllUsers(ctx context.Context, req *GetAllUsersRequest) (*GetAllUsersResponse, error) {
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}

	logrus.WithFields(logrus.Fields{
		"page": req.Page, "limit": req.Limit, "role": req.Role, "status": req.Status, "search": req.Search,
	}).Debug("Getting all users")

	users, totalCount, err := s.userRepository.GetAllUsers(ctx, req)
	if err != nil {
		logrus.WithError(err).Error("Failed to get users from repository")
		return nil, err
	}

	profiles := make([]*Profile, len(users))
	for i, user := range users {
		profiles[i] = user.ToProfile()
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(req.Limit)))
	response := &GetAllUsersResponse{
		Users: profiles, TotalCount: totalCount, Page: req.Page, Limit: req.Limit, TotalPages: totalPages,
	}

	logrus.WithFields(logrus.Fields{
		"users_count": len(profiles), "total_count": totalCount, "total_pages": totalPages,
	}).Info("Successfully retrieved users")

	return response, nil
}

// validateRequest validates and normalizes the GetAllUsersRequest
func (s *userService) validateRequest(req *GetAllUsersRequest) error {
	if req.Limit <= 0 {
		req.Limit = s.cfg.Search.MinQueryLimit
	}
	if req.Limit > 100 {
		req.Limit = s.cfg.Search.MaxQueryLimit
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	// Reset invalid optional fields to empty/default values instead of erroring
	if req.Role != "" && !isValidRole(req.Role) {
		req.Role = ""
	}
	if req.Status != "" && !isValidStatus(req.Status) {
		req.Status = ""
	}
	if req.SortBy != "" && !isValidSortBy(req.SortBy) {
		req.SortBy = ""
	}
	if req.SortOrder != "" && !isValidSortOrder(req.SortOrder) {
		req.SortOrder = ""
	}

	// Set default sort by registration date if not specified or invalid
	if req.SortBy == "" {
		req.SortBy = SortByRegistrationDate
	}
	if req.SortOrder == "" {
		req.SortOrder = SortOrderDesc
	}

	req.SortDirection = getSortDirection(req.SortOrder)
	return nil
}

// isValidRole validates if role is valid
func isValidRole(role string) bool {
	validRoles := []string{RoleAdmin, RoleClient, RoleExecutor}
	for _, validRole := range validRoles {
		if validRole == role {
			return true
		}
	}
	return false
}

// isValidStatus validates if status is valid
func isValidStatus(status string) bool {
	validStatuses := []string{StatusActive, StatusInactive, StatusSuspended}
	for _, validStatus := range validStatuses {
		if validStatus == status {
			return true
		}
	}
	return false
}

func (s *userService) GetUserStats(ctx context.Context) (*models.Stats, error) {
	logrus.Debug("Getting user statistics")

	stats, err := s.userRepository.GetUserStats(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get user stats from repository")
		return nil, err
	}

	logrus.WithFields(logrus.Fields{
		"total":        stats.Total,
		"active":       stats.Active,
		"specialists":  stats.Specialists,
		"clients":      stats.Clients,
		"suspended":    stats.Suspended,
		"newThisMonth": stats.NewThisMonth,
	}).Info("Successfully retrieved user statistics")

	return stats, nil
}

func isValidSortBy(sortBy string) bool {
	validSortFields := []string{
		SortByRegistrationDate,
		SortByFirstName,
		SortByLastName,
		SortByEmail,
		SortByLastActiveAt,
		SortByRole,
	}
	for _, validField := range validSortFields {
		if validField == sortBy {
			return true
		}
	}
	return false
}

func isValidSortOrder(sortOrder string) bool {
	return sortOrder == SortOrderAsc || sortOrder == SortOrderDesc
}

func getSortDirection(sortOrder string) int {
	if sortOrder == SortOrderAsc {
		return 1
	}
	return -1 // default to descending
}
