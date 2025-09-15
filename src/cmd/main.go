package main

import (
	"handyhub-admin-svc/src/internal/config"
	"handyhub-admin-svc/src/internal/logger"
	"handyhub-admin-svc/src/internal/server"

	"github.com/sirupsen/logrus"
)

var log = *logrus.StandardLogger()

func main() {
	cfg := config.Load()
	logger.Init(cfg)

	log.Infof("Application %s is starting....", cfg.App.Name)

	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		log.WithError(err).Fatalf("Error starting server: %v", err)
	}
}
