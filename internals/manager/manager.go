package manager

import (
	"context"
	"errors"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strings"

	"github.com/codeshelldev/docker-network-aliaser/internals/config"
	"github.com/codeshelldev/docker-network-aliaser/internals/config/structure"
	"github.com/codeshelldev/docker-network-aliaser/internals/docker"
	"github.com/codeshelldev/gotl/pkg/logger"
	"github.com/codeshelldev/gotl/pkg/templating"
	"github.com/moby/moby/api/types/network"
	net "github.com/moby/moby/api/types/network"
	cli "github.com/moby/moby/client"
)

const (
	ENABLED_LABEL_PART = "enable"
	ALIAS_LABEL_PART = "alias"

	PROJECT_LABEL = "com.docker.compose.project"
	SERVICE_LABEL = "com.docker.compose.service"
)

func Start() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
	)
	defer cancel()

	for _, network := range config.ENV.NETWORKS {
		err := checkNetwork(ctx, network)

		if err != nil {
			logger.Error("Issue with " + network.Name + " network: " + err.Error())
		}
	}

	err := reconcile(ctx)

	if err != nil {
		logger.Error("Reconsiliation failed: " + err.Error())
	}

	err = watch(ctx)
	if err != nil {
		logger.Error("Watcher errored: " + err.Error())
	}
}

func reconcile(ctx context.Context) error {
	client := docker.Client()

	filters := cli.Filters{}.
		Add("type", "container").
		Add("status", "running")
		
	containers, err := client.ContainerList(ctx, cli.ContainerListOptions{
		Filters: filters,
	})

	if err != nil {
		return err
	}

	for _, c := range containers.Items {
		err := processContainer(ctx, c.ID)
		if err != nil {
			logger.Error("Could not process " + shortID(c.ID) + ":" + err.Error())
		}
	}

	return nil
}

func watch(ctx context.Context) error {
	client := docker.Client()

	filters := cli.Filters{}.
		Add("type", "container").
		Add("event", "start", "restart")

	result := client.Events(ctx, cli.EventsListOptions{
		Filters: filters,
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-result.Err:
			if err != nil {
				return err
			}

		case event, ok := <-result.Messages:
			if !ok {
				return errors.New("Docker event stream closed")
			}

			err := processContainer(ctx, event.Actor.ID)
			if err != nil {
				logger.Error("Could not process " + shortID(event.Actor.ID) + ":" + err.Error())
			}
		}
	}
}

func processContainer(ctx context.Context, containerID string) error {
	client := docker.Client()

	result, err := client.ContainerInspect(ctx, containerID, cli.ContainerInspectOptions{})

	if err != nil {
		return err
	}

	labels := result.Container.Config.Labels

	enabled, prefix := isEnabled(labels)

	network := config.ENV.NETWORKS[prefix]

	if !enabled {
		return nil
	}

	project := labels[PROJECT_LABEL]
	if project == "" {
		logger.Warn("Container " + shortID(containerID) + " is enabled but has no Compose project")
		return nil
	}

	alias := labels[prefix + "." + ALIAS_LABEL_PART]

	if alias == "" {
		alias = labels[SERVICE_LABEL]
	}

	if alias == "" {
		logger.Warn("Container " + shortID(containerID) + " has no service or alias")
		return nil
	}

	dnsName, err := constructDNSName(project, alias)

	if err != nil {
		return err
	}

	return connectNetwork(ctx, containerID, dnsName, network)
}

func connectNetwork(ctx context.Context, containerID, alias string, network structure.NetworkConfig) error {
	client := docker.Client()

	containerInfo, err := client.ContainerInspect(ctx, containerID, cli.ContainerInspectOptions{})
	if err != nil {
		return err
	}

	endpoint, connected := containerInfo.Container.NetworkSettings.Networks[network.Name]

	if connected {
		if slices.Contains(endpoint.Aliases, alias) {
			return nil
		}

		logger.Info("Updating alias for " + shortID(containerID) + ": " + alias)

		_, err := client.NetworkDisconnect(ctx, endpoint.NetworkID, cli.NetworkDisconnectOptions{})
		if err != nil {
			return err
		}
	}

	logger.Debug("Connecting " + shortID(containerID) + " to " + endpoint.NetworkID + " as " + alias)

	result, err := getNetworkByName(ctx, network.Name)

	if err != nil {
		return err
	}

	_, err = client.NetworkConnect(ctx, result.ID, cli.NetworkConnectOptions{EndpointConfig: &net.EndpointSettings{Aliases: []string{ alias }}})
	return err
}

func checkNetwork(ctx context.Context, network structure.NetworkConfig) error {
	client := docker.Client()

	result, err := getNetworkByName(ctx, network.Name)

	if err != nil {
		return err
	}

	_, err = client.NetworkInspect(ctx, result.ID, cli.NetworkInspectOptions{})
	if err == nil {
		return nil
	}

	if !config.ENV.CREATE_NETWORKS {
		logger.Error("Network " + network.Name + " does not exist")

		return errors.New("network not found")
	}

	logger.Info("Network " + network.Name + " does not exist. Creating it.")

	_, err = client.NetworkCreate(ctx, network.Name, cli.NetworkCreateOptions{Driver: "bridge", Labels: map[string]string{"managed-by": "docker-network-aliaser"}})
	return err
}

func getNetworkByName(ctx context.Context, name string) (net.Summary, error){
	client := docker.Client()

	filters := cli.Filters{}.
		Add("name", name)
	
	result, err := client.NetworkList(ctx, cli.NetworkListOptions{
		Filters: filters,
	})

	if err != nil {
		return network.Summary{}, err
	}

	if len(result.Items) == 0 {
		return network.Summary{}, nil
	}

	return result.Items[0], nil
}

func constructDNSName(project, alias string) (string, error) {
	tmplt, err := templating.CreateTemplateFromString(project + ":" + alias, config.ENV.ALIAS_NAME_TEMPLATE)

	if err != nil {
		return "", err
	}

	return templating.ExecuteTemplate(tmplt, map[string]string{"project": sanitize(project), "alias": sanitize(alias)})
}

func isEnabled(labels map[string]string) (bool, string) {
	for prefix := range maps.Keys(config.ENV.NETWORKS) {
		value, exists := labels[prefix + "." + ENABLED_LABEL_PART]

		if !exists {
			continue
		}

		value = strings.ToLower(value)
		return value == "true", prefix
	}

	return false, ""
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}

	return id[:12]
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)

	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, " ", "_")

	return value
}