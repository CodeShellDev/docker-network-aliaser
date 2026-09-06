# Docker Network Aliaser

Alias containers on global networks to prevent dns collisions.

## Installation

Add the following to your `docker-compose.yaml`:

```yaml
services:
  aliaser:
    image: ghcr.io/codeshelldev/docker-network-aliaser:latest
    container_name: docker-network-aliaser
    environment:
      PREFIXES: my_prefix_1, my_prefix_2
      NETWORKS: my_network_1, my_network_2
      DOCKER_HOST: tcp://socket-proxy:2375
    depends_on:
      - socket-proxy
    restart: always
    networks:
      - backend

  socket-proxy:
    image: lscr.io/linuxserver/socket-proxy:latest
    container_name: docker-network-aliaser-socket-proxy
    environment:
      CONTAINERS: 1
      NETWORKS: 1
      EVENTS: 1
      POST: 1
      PING: 1
      VERSION: 1
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    read_only: true
    tmpfs:
      - /run
    restart: unless-stopped
    networks:
      - backend

networks:
  backend:

```

## Usage

Take a look at this compose file below:

```yaml
# inside of example-project/ directory
services:
  postgres:
    image: postgres:16-alpine
    container_name: example-project-postgres
    labels:
      - my_prefix_1.enable=true
      - my_prefix_1.alias=db
    restart: unless-stopped
    networks:
      - backend

```

Here we are enabling `my_prefix_1` and overriding the default alias that is derived from the service name with `my_prefix_1.alias`.

D.N.A will then connect the container to the `my_network_1` network with the alias `example-project_db`.
Now other containers can reach `postgres` via `example-project_db` without having to manually configure this alias or attaching the network to the compose project.

## Configuration

### `ALIAS_NAME_TEMPLATE`

Alias template for containers (default `{{.PROJECT}}_{{.SERVICE}}`).

### `RECONCILE_INTERVAL`

Interval of reconciliation (default `5m`).

### `CREATE_NETWORKS`

Whether to automatically create networks from [`NETWORKS`](#networks) at startup (default: `true`).

### `PREFIXES`

Comma-separated list of prefixes, that are used for `prefix.enable` and `prefix.alias`.

### `NETWORKS`

Comma-separated list of networks to connect containers to according to enabled [prefixes](#prefixes).

> [!IMPORTANT]
> The amount of [prefixes](#prefixes) must match exactly the amount of [networks](#networks)!

## Contributing

Have suggestions or improvements? Feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](./LICENSE).
