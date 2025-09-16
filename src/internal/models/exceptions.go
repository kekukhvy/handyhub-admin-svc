package models

import "errors"

var (
	ErrRedisConnection = errors.New("redis connection error")
	ErrRedisGet        = errors.New("redis get error")
	ErrRedisSet        = errors.New("redis set error")
	ErrRedisDelete     = errors.New("redis delete error")
	ErrRedisExpire     = errors.New("redis expire error")
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionInactive = errors.New("session inactive")
	ErrSessionInvalid  = errors.New("session invalid")
	ErrSessionCreating = errors.New("error creating session")
	ErrSessionUpdating = errors.New("error updating session")
	ErrSessionDeleting = errors.New("error deleting session")
	ErrTooManySessions = errors.New("too many active sessions")
)

var (
	ErrDatabaseConnection = errors.New("database connection error")
	ErrDatabaseQuery      = errors.New("database query error")
	ErrDatabaseInsert     = errors.New("database insert error")
	ErrDatabaseUpdate     = errors.New("database update error")
	ErrDatabaseDelete     = errors.New("database delete error")
	ErrRecordNotFound     = errors.New("record not found")
	ErrDuplicateRecord    = errors.New("duplicate record")
)
