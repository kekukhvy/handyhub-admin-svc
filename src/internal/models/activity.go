package models

import "time"

type ActivityMessage struct {
	UserID      string            `json:"user_id"`
	SessionID   string            `json:"session_id"`
	ServiceName string            `json:"service_name"`
	Action      string            `json:"action"`
	IPAddress   string            `json:"ip_address,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Activity action constants
const (
	ActionAuthenticated    = "authenticated"
	ActionSessionCheck     = "session_check"
	ActionUserStatsRequest = "user_stats_request"
	ActionUserListRequest  = "user_list_request"
	ActionUserStatusUpdate = "user_status_update"
)

// Service name constants
const (
	ServiceAdminAuth       = "admin.middleware.auth"
	ServiceAdminUserStats  = "admin.handler.user_stats"
	ServiceAdminUserList   = "admin.handler.user_list"
	ServiceAdminUserUpdate = "admin.handler.user_update"
)
