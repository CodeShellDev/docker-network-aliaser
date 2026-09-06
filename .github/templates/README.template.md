# Docker Network Aliaser

Alias containers on global networks to prevent dns collisions.

## Installation

Add the following to your `docker-compose.yaml`:

```yaml
+{{{ read "docker-compose.yaml" }}}
```

## Usage

Take a look at this compose file below:

```yaml
+{{{ read "examples/postgres.docker-compose.yaml" }}}
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
