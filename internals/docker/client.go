package docker

import (
	"github.com/codeshelldev/gotl/pkg/logger"
	"github.com/moby/moby/client"
)

var apiClient *client.Client

func InitClient() {
	var err error

	apiClient, err = client.New(client.FromEnv)

	if err != nil {
		logger.Fatal("Could not connect to ", apiClient.DaemonHost(), ": ", err.Error())
	}
	defer apiClient.Close()
}

func Client() *client.Client {
	return apiClient
}