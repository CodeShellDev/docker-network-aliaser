package structure

import "time"

type ENV struct {
	LOG_LEVEL 				string
	ALIAS_NAME_TEMPLATE		string
	RECONCILE_INTERVAL		time.Duration
	CREATE_NETWORKS			bool
	NETWORKS				map[string]NetworkConfig
}

type NetworkConfig struct {
	Name					string
}