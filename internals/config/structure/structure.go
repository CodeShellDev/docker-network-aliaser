package structure

type ENV struct {
	LOG_LEVEL 				string
	DNS_NAME_TEMPLATE		string
	CREATE_NETWORKS			bool
	NETWORKS				map[string]NetworkConfig
}

type NetworkConfig struct {
	Name					string
}