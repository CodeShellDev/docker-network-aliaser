package main

import (
	"os"
	"runtime/debug"

	"github.com/codeshelldev/docker-network-aliaser/internals/config"
	"github.com/codeshelldev/docker-network-aliaser/internals/docker"
	"github.com/codeshelldev/docker-network-aliaser/internals/manager"
	"github.com/codeshelldev/gotl/pkg/logger"
)

func main() {
	logger.Init(os.Getenv("LOG_LEVEL"))

	config.Load()

	docker.Init()

	logger.Info("Initialized Logger with Level of ", logger.Level())

	if logger.Level() == "dev" {
		logger.Dev("Welcome back Developer!")
	}

	config.Log()

	docker.InitClient()

	stop := docker.Run(func() {
		defer func() {
			r := recover()
			if r != nil {
				logger.Fatal("Paniced: ", debug.Stack())
			}
		}()

		manager.Start()
	})

	<-stop
	docker.Shutdown()
}
