package dependency

import (
	"handyhub-admin-svc/src/clients"
	"handyhub-admin-svc/src/internal/cache"
	"handyhub-admin-svc/src/internal/config"
	"handyhub-admin-svc/src/internal/user"

	"github.com/gin-gonic/gin"
)

type Manager struct {
	Router       *gin.Engine
	Config       *config.Configuration
	Mongodb      *clients.MongoDB
	Redis        *clients.RedisClient
	RabbitMQ     *clients.RabbitMQ
	UserService  user.Service
	UserHandler  user.Handler
	CacheService cache.Service
	AuthClient   *clients.AuthClient
}

func NewDependencyManager(router *gin.Engine,
	mongodb *clients.MongoDB,
	redisClient *clients.RedisClient,
	rabbitMQ *clients.RabbitMQ,
	cfg *config.Configuration) *Manager {
	cacheService := cache.NewCacheService(redisClient.Client, cfg)
	userRepo := user.NewUserRepository(mongodb, cfg.Database.Collections.Users)
	userService := user.NewUserService(userRepo, cfg)
	userHandler := user.NewHandler(cfg, userService, cacheService)
	authClient := clients.NewAuthClient(cfg, rabbitMQ.Channel)

	return &Manager{
		Router:       router,
		Config:       cfg,
		Mongodb:      mongodb,
		Redis:        redisClient,
		RabbitMQ:     rabbitMQ,
		UserService:  userService,
		UserHandler:  userHandler,
		CacheService: cacheService,
		AuthClient:   authClient,
	}
}
