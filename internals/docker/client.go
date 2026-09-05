package docker

import (
	"github.com/codeshelldev/gotl/pkg/logger"
	"github.com/moby/moby/client"
)

var apiClient *client.Client

func InitClient(options ...client.Opt) {
	var err error

	if len(options) <= 0 {
		options = append(options, client.WithHost("unix:///var/run/docker.sock"))
	}

	apiClient, err = client.New(options...)

	if err != nil {
		logger.Fatal("Could not connect to " + apiClient.DaemonHost() + ": ", err.Error())
	}
	defer apiClient.Close()
}

func Client() *client.Client {
	return apiClient
}