package config

import (
	"os"

	"github.com/codeshelldev/docker-network-aliaser/internals/config/structure"
	"github.com/codeshelldev/gotl/pkg/logger"
)

var ENV = &structure.ENV{
	LOG_LEVEL: "info",
}

func Load() {
	ENV.LOG_LEVEL = os.Getenv("LOG_LEVEL")
}

func Log() {
	logger.Dev("Loaded Environment:", ENV)
}