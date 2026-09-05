package main

import (
	"github.com/codeshelldev/docker-network-aliaser/internals/config"
	"github.com/codeshelldev/docker-network-aliaser/internals/docker"
	"github.com/codeshelldev/docker-network-aliaser/internals/manager"
	"github.com/codeshelldev/gotl/pkg/logger"
)

func main() {
	config.Load()

	logger.Init(config.ENV.LOG_LEVEL)

	docker.Init()

	logger.Info("Initialized Logger with Level of ", logger.Level())

	if logger.Level() == "dev" {
		logger.Dev("Welcome back Developer!")
	}

	config.Log()

	docker.InitClient()

	stop := docker.Run(func() {
		manager.Start()
	})

	<-stop
	docker.Shutdown()
}
