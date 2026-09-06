package manager

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/codeshelldev/docker-network-aliaser/internals/config"
	"github.com/codeshelldev/docker-network-aliaser/internals/config/structure"
	"github.com/codeshelldev/docker-network-aliaser/internals/docker"
	"github.com/codeshelldev/gotl/pkg/logger"
	"github.com/codeshelldev/gotl/pkg/templating"
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
			logger.Error("Issue with ", network.Name, " network: ", err.Error())
		}
	}

	logger.Debug("Starting initial reconciliation...")

	err := reconcile(ctx)
	if err != nil {
		logger.Error("Reconciliation failed: ", err.Error())
	}

	go runWatcher(ctx)
	go runReconciler(ctx)

	<-ctx.Done()

	logger.Info("Shutting down...")
}

func runWatcher(ctx context.Context) {
	const retryDelay = 5*time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		logger.Debug("Starting Docker event watcher...")

		err := watch(ctx)

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			logger.Error("Docker event watcher stopped: ", err.Error())
		}

		logger.Debug("Docker event watcher reconnecting in ", retryDelay)

		timer := time.NewTimer(retryDelay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return

		case <-timer.C:
		}
	}
}

func runReconciler(ctx context.Context) {
	ticker := time.NewTicker(config.ENV.RECONCILE_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			logger.Debug("Starting periodic reconciliation...")

			err := reconcile(ctx)
			if err != nil {
				logger.Error("Periodic reconciliation failed: ", err.Error())
			}
		}
	}
}

func reconcile(ctx context.Context) error {
	client := docker.Client()

	filters := cli.Filters{}.
		Add("status", "running")

	containers, err := client.ContainerList(ctx, cli.ContainerListOptions{
		Filters: filters,
	})
	if err != nil {
		return err
	}

	var failed int

	for _, c := range containers.Items {
		err := processContainer(ctx, c.ID)
		if err != nil {
			failed++

			logger.Error("Could not process ", shortID(c.ID), ": ", err.Error())
		}
	}

	if failed > 0 {
		return errors.New(strconv.Itoa(failed) + " containers failed reconciliation")
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

		case err, ok := <-result.Err:
			if !ok {
				return errors.New("Docker event error stream closed")
			}

			if err != nil {
				return err
			}

		case event, ok := <-result.Messages:
			if !ok {
				return errors.New("Docker event stream closed")
			}

			err := processContainer(ctx, event.Actor.ID)
			if err != nil {
				logger.Error("Could not process ", shortID(event.Actor.ID), ":", err.Error())
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

	enabled, prefixes := getEnabled(labels)

	if !enabled {
		return nil
	}

	project := labels[PROJECT_LABEL]

	if project == "" {
		logger.Warn("Container ", shortID(containerID), " is enabled but has no Compose project")
		return nil
	}

	service := labels[SERVICE_LABEL]

	var failed int

	for _, prefix := range prefixes {
		network := config.ENV.NETWORKS[prefix]

		alias := labels[prefix + "." + ALIAS_LABEL_PART]
		if alias == "" {
			alias = service
		}

		if alias == "" {
			logger.Warn("Container ", shortID(containerID), " has no service or alias")
			failed++
			continue
		}

		dnsName, err := constructDnsAlias(project, alias)
		if err != nil {
			logger.Error("Could not construct alias for ", prefix, ": ", err.Error())
			failed++
			continue
		}

		err = connectNetwork(ctx, containerID, dnsName, network)
		if err != nil {
			logger.Error("Could not process prefix ", prefix, ": ", err.Error())
			failed++
			continue
		}
	}

	if failed > 0 {
		return errors.New(strconv.Itoa(failed) + " prefixes failed")
	}

	return nil
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

		logger.Info("Adding alias for ", shortID(containerID), ": ", alias)

		aliases := append([]string{}, endpoint.Aliases...)
		aliases = append(aliases, alias)

		_, err := client.NetworkDisconnect(ctx, endpoint.NetworkID, cli.NetworkDisconnectOptions{
			Container: containerID,
			Force: true,
		})
		if err != nil {
			return err
		}

		_, err = client.NetworkConnect(ctx, endpoint.NetworkID, cli.NetworkConnectOptions{
			Container: containerID,
			EndpointConfig: &net.EndpointSettings{
				Aliases: aliases,
			},
		})

		return err
	}

	logger.Debug("Connecting ", shortID(containerID), " to ", network.Name, " as ", alias)

	result, err := getNetworkByName(ctx, network.Name)
	if err != nil {
		return err
	}

	if result.ID == "" {
		return errors.New("network not found: " + network.Name)
	}

	_, err = client.NetworkConnect(ctx, result.ID, cli.NetworkConnectOptions{
		Container: containerID,
		EndpointConfig: &net.EndpointSettings{
			Aliases: []string{alias},
		},
	})

	return err
}

func checkNetwork(ctx context.Context, network structure.NetworkConfig) error {
	client := docker.Client()

	result, err := getNetworkByName(ctx, network.Name)
	if err != nil {
		return err
	}

	if result.ID != "" {
		return nil
	}

	if !config.ENV.CREATE_NETWORKS {
		logger.Error("Network ", network.Name, " does not exist")

		return errors.New("network not found")
	}

	logger.Info("Network ", network.Name, " does not exist. Creating it.")

	_, err = client.NetworkCreate(ctx, network.Name, cli.NetworkCreateOptions{Driver: "bridge", Labels: map[string]string{"managed-by": "docker-network-aliaser"}})
	return err
}

func getNetworkByName(ctx context.Context, name string) (net.Summary, error) {
	client := docker.Client()

	result, err := client.NetworkList(ctx, cli.NetworkListOptions{
		Filters: cli.Filters{}.
			Add("name", name),
	})
	if err != nil {
		return net.Summary{}, err
	}

	for _, n := range result.Items {
		if n.Name == name {
			return n, nil
		}
	}

	return net.Summary{}, nil
}

func constructDnsAlias(project, alias string) (string, error) {
	tmplt, err := templating.CreateTemplateFromString(project + ":" + alias, config.ENV.ALIAS_NAME_TEMPLATE)

	if err != nil {
		return "", err
	}

	return templating.ExecuteTemplate(tmplt, map[string]string{"PROJECT": sanitize(project), "ALIAS": sanitize(alias)})
}

func getEnabled(labels map[string]string) (bool, []string) {
	prefixes := []string{}

	for prefix := range config.ENV.NETWORKS {
		value, exists := labels[prefix + "." + ENABLED_LABEL_PART]

		if !exists {
			continue
		}

		value = strings.ToLower(value)

		if value == "true" {
			prefixes = append(prefixes, prefix)
		}
	}

	return len(prefixes) != 0, prefixes
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