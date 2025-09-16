package dependency

import (
	"handyhub-admin-svc/src/clients"
	"handyhub-admin-svc/src/internal/cache"
	"handyhub-admin-svc/src/internal/config"
	"handyhub-admin-svc/src/internal/session"
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
	SessionRepo  session.Repository
}

func NewDependencyManager(router *gin.Engine,
	mongodb *clients.MongoDB,
	redisClient *clients.RedisClient,
	rabbitMQ *clients.RabbitMQ,
	cfg *config.Configuration) *Manager {

	userRepo := user.NewUserRepository(mongodb, cfg.Database.UserCollection)
	userService := user.NewUserService(userRepo, cfg)
	userHandler := user.NewHandler(cfg, userService)
	cacheService := cache.NewCacheService(redisClient.Client, cfg)
	sessionRepo := session.NewSessionRepository(mongodb, cfg.Database.SessionCollection)
	return &Manager{
		Router:       router,
		Config:       cfg,
		Mongodb:      mongodb,
		Redis:        redisClient,
		RabbitMQ:     rabbitMQ,
		UserService:  userService,
		UserHandler:  userHandler,
		CacheService: cacheService,
		SessionRepo:  sessionRepo,
	}
}
