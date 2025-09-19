package server

import (
	"handyhub-admin-svc/src/clients"
	"handyhub-admin-svc/src/internal/dependency"
	"handyhub-admin-svc/src/internal/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRoutes(deps *dependency.Manager) {
	router := deps.Router
	router.Use(enableCORS)

	setupHealthEndpoint(deps)
	setupPublicRoutes(router, deps)
	setupAdminRoutes(router, deps)
}

func setupHealthEndpoint(deps *dependency.Manager) {
	router := deps.Router
	mongodb := deps.Mongodb
	redisClient := deps.Redis
	cfg := deps.Config

	router.GET("/health", func(c *gin.Context) {
		log.Info("Health check endpoint requested")

		mongoStatus := "ok"
		if err := mongodb.Client.Ping(c.Request.Context(), nil); err != nil {
			mongoStatus = "error: " + err.Error()
		}

		redisStatus := "ok"
		if err := redisClient.Client.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = "error: " + err.Error()
		}

		c.JSON(200, gin.H{
			"status":    "ok",
			"service":   cfg.App.Name,
			"version":   cfg.App.Version,
			"mongodb":   mongoStatus,
			"redis":     redisStatus,
			"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	})

	router.GET("health/detailed", func(c *gin.Context) {
		log.Info("Detailed health check endpoint requested")

		c.JSON(200, gin.H{
			"status":  "operational",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
			"components": gin.H{
				"database": gin.H{
					"mongodb": getStatus(isMongoConnected(mongodb, c)),
					"redis":   getStatus(isRedisConnected(redisClient.Client, c)),
				},
				"services": gin.H{
					"auth":    "operational",
					"session": "operational",
					"cache":   "operational",
				},
			},
		})
	})
}

func setupPublicRoutes(router *gin.Engine, deps *dependency.Manager) {
	// API status endpoint
	router.GET("/api/v1/status", func(c *gin.Context) {
		log.Info("API status requested")
		c.JSON(200, gin.H{
			"api_version": "v1",
			"status":      "operational",
			"service":     "handyhub-admin-svc",
		})
	})
}

func setupAdminRoutes(router *gin.Engine, deps *dependency.Manager) {
	// Create auth middleware with AuthClient instead of SessionRepo
	authMiddleware := middleware.NewAuthMiddleware(
		deps.Config.Security.JwtKey,
		deps.CacheService,
		deps.AuthClient,
	)

	handler := deps.UserHandler

	// Apply route name FIRST, then auth middlewares
	admin := router.Group("/api/v1/admin")
	{
		admin.GET("/users",
			setRouteName("getUsersList"),
			authMiddleware.RequireAuth(),
			authMiddleware.RequireAdminRights(),
			handler.GetAllUsers)

		admin.GET("/users/stats",
			setRouteName("getUsersStats"),
			authMiddleware.RequireAuth(),
			authMiddleware.RequireAdminRights(),
			handler.GetUserStats)

		admin.PATCH("/users/:id/activate",
			setRouteName("activateUser"),
			authMiddleware.RequireAuth(),
			authMiddleware.RequireAdminRights(),
			handler.ActivateUser)

		admin.PATCH("/users/:id/deactivate",
			setRouteName("deactivateUser"),
			authMiddleware.RequireAuth(),
			authMiddleware.RequireAdminRights(),
			handler.DeactivateUser)

		admin.PATCH("/users/:id/suspend",
			setRouteName("suspendUser"),
			authMiddleware.RequireAuth(),
			authMiddleware.RequireAdminRights(),
			handler.SuspendUser)
	}
}

func setRouteName(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("route_name", name)
		c.Next()
	}
}

func isMongoConnected(mongodb *clients.MongoDB, c *gin.Context) bool {
	if err := mongodb.Client.Ping(c.Request.Context(), nil); err != nil {
		return false
	}
	return true
}

func isRedisConnected(redisClient *redis.Client, c *gin.Context) bool {
	if err := redisClient.Ping(c.Request.Context()).Err(); err != nil {
		return false
	}
	return true
}

func enableCORS(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}

	c.Next()
}

func getStatus(b bool) string {
	if b {
		return "connected"
	}
	return "disconnected"
}
