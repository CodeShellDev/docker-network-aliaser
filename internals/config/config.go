package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/codeshelldev/docker-network-aliaser/internals/config/structure"
	"github.com/codeshelldev/gotl/pkg/logger"
)

var ENV = &structure.ENV{
	LOG_LEVEL: "info",
	ALIAS_NAME_TEMPLATE: "{{.PROJECT}}_{{.ALIAS}}",
	CREATE_NETWORKS: true,
	NETWORKS: map[string]structure.NetworkConfig{},
}

func Load() {
	level := os.Getenv("LOG_LEVEL")

	if strings.TrimSpace(level) != "" {
		ENV.LOG_LEVEL = level
	}

	tmpl := os.Getenv("ALIAS_NAME_TEMPLATE")

	if strings.TrimSpace(tmpl) != "" {
		ENV.ALIAS_NAME_TEMPLATE = tmpl
	}

	createNetworksStr := os.Getenv("CREATE_NETWORKS")

	if strings.TrimSpace(createNetworksStr) != "" {
		createNetworks, err := strconv.ParseBool(createNetworksStr)

		if err != nil {
			logger.Error("Invalid CREATE_NETWORKS: " + err.Error())
		} else {
			ENV.CREATE_NETWORKS = createNetworks
		}
	}

	prefixesStr := os.Getenv("PREFIXES")
	networksStr := os.Getenv("NETWORKS")

	prefixes := strings.Split(prefixesStr, ",")
	networks := strings.Split(networksStr, ",")

	if len(prefixes) != len(networks) {
		logger.Error("Invalid length of prefixes to networks (", len(prefixes), " and ", len(networks), ")")
	}

	for i, prefix := range prefixes {
		logger.Debug("Registered prefix " + prefix + " with network name " + networks[i])

		ENV.NETWORKS[prefix] = structure.NetworkConfig{Name: networks[i]}
	}

	logger.Info("Registered ", len(prefixes), " prefixes")
}

func Log() {
	logger.Dev("Loaded Environment:", ENV)
}