package user

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID                  primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	FirstName           string             `json:"firstName" bson:"first_name"`
	LastName            string             `json:"lastName" bson:"last_name"`
	Email               string             `json:"email" bson:"email"`
	Phone               *string            `json:"phone,omitempty" bson:"phone,omitempty"`
	Role                string             `json:"role" bson:"role"`
	Status              string             `json:"status" bson:"status"`
	IsEmailVerified     bool               `json:"isEmailVerified" bson:"is_email_verified"`
	RegistrationDate    time.Time          `json:"registrationDate" bson:"registration_date"`
	LastLoginAt         *time.Time         `json:"lastLoginAt,omitempty" bson:"last_login_at,omitempty"`
	LastActiveAt        *time.Time         `json:"lastActiveAt,omitempty" bson:"last_active_at,omitempty"`
	TotalRequests       int64              `json:"totalRequests" bson:"total_requests"`
	FailedLoginAttempts int                `json:"failedLoginAttempts" bson:"failed_login_attempts"`
	LastFailedLoginAt   *time.Time         `json:"lastFailedLoginAt,omitempty" bson:"last_failed_login_at,omitempty"`
	Avatar              *string            `json:"avatar,omitempty" bson:"avatar,omitempty"`
	TimeZone            string             `json:"timeZone" bson:"time_zone"`
	Language            string             `json:"language" bson:"language"`
	CreatedAt           time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt           time.Time          `json:"updatedAt" bson:"updated_at"`
	DeletedAt           *time.Time         `json:"deletedAt,omitempty" bson:"deleted_at,omitempty"`
}

type Profile struct {
	ID               primitive.ObjectID `json:"id"`
	FirstName        string             `json:"firstName"`
	LastName         string             `json:"lastName"`
	Email            string             `json:"email"`
	Phone            *string            `json:"phone,omitempty"`
	Role             string             `json:"role"`
	Status           string             `json:"status"`
	IsEmailVerified  bool               `json:"isEmailVerified"`
	RegistrationDate time.Time          `json:"registrationDate"`
	LastLoginAt      *time.Time         `json:"lastLoginAt,omitempty"`
	LastActiveAt     *time.Time         `json:"lastActiveAt,omitempty"`
	TotalRequests    int64              `json:"totalRequests"`
	Avatar           *string            `json:"avatar,omitempty"`
	TimeZone         string             `json:"timeZone"`
	Language         string             `json:"language"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
}

type Stats struct {
	Total        int64 `json:"total"`
	Active       int64 `json:"active"`
	Inactive     int64 `json:"inactive"`
	Specialists  int64 `json:"specialists"`
	Clients      int64 `json:"clients"`
	Suspended    int64 `json:"suspended"`
	NewThisMonth int64 `json:"newThisMonth"`
}

// Role constants
const (
	RoleAdmin    = "admin"
	RoleClient   = "client"
	RoleExecutor = "executor"
)

// Status constants
const (
	StatusActive    = "active"
	StatusInactive  = "inactive"
	StatusSuspended = "suspended"
)

// GetAllUsersRequest represents request for getting all users
type GetAllUsersRequest struct {
	Page   int    `json:"page" form:"page"`
	Limit  int    `json:"limit" form:"limit"`
	Role   string `json:"role" form:"role"`
	Status string `json:"status" form:"status"`
	Search string `json:"search" form:"search"`
}

// GetAllUsersResponse represents response for getting all users
type GetAllUsersResponse struct {
	Users      []*Profile `json:"users"`
	TotalCount int64      `json:"totalCount"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"totalPages"`
}

// ToProfile converts User to UserProfile
func (u *User) ToProfile() *Profile {
	return &Profile{
		ID:               u.ID,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		Email:            u.Email,
		Phone:            u.Phone,
		Role:             u.Role,
		Status:           u.Status,
		IsEmailVerified:  u.IsEmailVerified,
		RegistrationDate: u.RegistrationDate,
		LastLoginAt:      u.LastLoginAt,
		LastActiveAt:     u.LastActiveAt,
		TotalRequests:    u.TotalRequests,
		Avatar:           u.Avatar,
		TimeZone:         u.TimeZone,
		Language:         u.Language,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

// IsAdmin checks if user is admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsActive checks if user is active
func (u *User) IsActive() bool {
	return u.Status == StatusActive && u.DeletedAt == nil
}
