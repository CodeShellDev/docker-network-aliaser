package docker

import (
	"os"

	"github.com/codeshelldev/gotl/pkg/docker"
	log "github.com/codeshelldev/gotl/pkg/logger"
)

func Init() {
	log.Info("Running ", os.Getenv("IMAGE_TAG"), " Image")
}

func Run(main func()) chan os.Signal {
	return docker.Run(main)
}

func Exit(code int) {
	log.Info("Exiting...")

	docker.Exit(code)
}

func Shutdown() {
	log.Info("Shutdown signal received")

	log.Sync()

	log.Info("Server exited gracefully")
}
