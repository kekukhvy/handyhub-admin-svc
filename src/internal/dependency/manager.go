package dependency

import (
	"handyhub-admin-svc/src/clients"
	"handyhub-admin-svc/src/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Manager struct {
	Router   *gin.Engine
	Config   *config.Configuration
	Mongodb  *clients.MongoDB
	Redis    *redis.Client
	RabbitMQ *clients.RabbitMQ
}

func NewDependencyManager(router *gin.Engine,
	mongodb *clients.MongoDB,
	redisClient *redis.Client,
	rabbitMQ *clients.RabbitMQ,
	cfg *config.Configuration) *Manager {

	return &Manager{
		Router:   router,
		Config:   cfg,
		Mongodb:  mongodb,
		Redis:    redisClient,
		RabbitMQ: rabbitMQ,
	}
}
