![logo.svg](img/logo.svg)

![Test Status](https://github.com/acamb/continuity/actions/workflows/makefile.yml/badge.svg)

# Continuity - Load balancing made simple.
Continuity is a lightweight load balancer designed for simplicity and ease of use, with a focus on small environments and home labs.
## Features
- Simple configuration backed to a single YAML file
- Can be managed statically via yaml file or via CLI client / RESTful API
- Zero downtime deployments of applications behind the load balancer via transactional API
- Configurable health checks for backend services
- Sticky sessions via application cookies or managed by the load balancer
- Dynamic pool configuration via API
- Custom routing via request headers
- Human-readable and JSON output for CLI client

## Installation

Continuity is distributed as a Docker image, .deb and .rpm packages, statically linked binary, or can be built from source.
The CLI client is also available for Windows.

### Docker
The simplest way to run Continuity is via Docker. You can pull the latest image from Docker Hub:

```bash
docker pull acamb/continuity:latest
```

Then, run the container with your configuration file mounted:

```bash
docker run -d -p 80:80 -v /path/to/continuity.yaml:/opt/continuity/config.yaml acamb/continuity:latest
```

or using docker-compose:

```yaml
services:
  continuity:
    image: acamb/continuity:latest
    ports:
      - "80:80"
    volumes:
      - /path/to/continuity.yaml:/opt/continuity/config.yaml
```

### Debian and RPM Packages

You can download and install the latest .deb or .rpm package from the release page.
For example on Debian-based systems:

```bash
apt install ./continuity-x.y.z.deb
```
And to install the client:

```bash
apt install ./continuity-client-x.y.z.deb
```

Both `.deb` and `.rpm` packages will install the server binary to `/usr/bin/continuity-server` and will create a systemd service for the server running as the `continuity-server` user.
For the client, the binary will be installed to `/usr/bin/continuity`.

### Statically Linked Binary / Manual installation

A statically linked version of Continuity is available for Linux amd64 and can be downloaded from the release page.

You can generate a configuration file template using:

```bash
./continuity-server -sample-config
```
And for the client:

```bash
./continuity sample-config
```

In both cases a config.yml file will be created in the current directory.

### Building from Source

To build Continuity from source, ensure you have Go installed (version 1.18 or later), then clone the repository and build:

```bash
make server
```
This will create the server (`continuity-server`) binary in the `bin/` directory.

To build the CLI client:

```bash
make client
```
This will create the client binary (`continuity`) in the `bin/` directory.

## Client Usage

Run the client in the directory containing your configuration file (config.yaml by default) or specify the config file with the `-config` flag.
The configuration file is per project, so you can have multiple configuration files for different environments / services.
You can also share the same configuration for different targets by creating different pools on the same server.
A pool represents a hostname or path you want to load balance traffic for.

### Create a new pool
```
continuity pool add <hostname>              # hostname and optional path to serve requests for, must contain the schema (e.g. http://my-app.domain.com)
  --health-check-interval SECONDS           # Seconds between health checks (default: 10s)
  --health-check-timeout SECONDS            # Health check connection timeout (default: 5s)
  --health-check-initial-delay SECONDS      # Initial delay on new server registration before starting health checks (default: 20s)
  --health-fail NUM_KO_RESPONSES_THRESHOLD  # Number of failed health checks before marking server as down (default: 3)
  --health-ok NUM_OK_RESPONSES_THRESHOLD    # Number of successful health checks before marking server as healthy (default: 2)
 [--sticky-sessions true/false]             # Enable sticky sessions (default: false)
 [--sticky-method [IP|AppCookie|LBCookie] ] # Sticky session method (default, if sticky sessions enabled: IP)
 [--cookie-name NAME]                       # Name of the application cookie to use for sticky sessions (required if sticky-method is AppCookie)
```
See the help (-h) for the full list of options and shorts.
Example:
```bash
continuity pool add http://my-app.domain.com -i 30 -t 10 -d 35 --health-ok 1 --health-fail 3
```

### Add a server to the pool
```
continuity server add  --pool POOL_HOSTNAME   # Pool hostname the server should be added to
  --address SERVER_ADDRESS:PORT               # Address of the server (IP or hostname)
 [--health-check /healthcheck_endpoint]       # Optional header name for routing condition
 [--condition MY_HEADER=MY_VALUE]             # Optional header value for routing condition
```

Example:
```bash
continuity server add --pool http://my-app.domain.com --address docker-1:8080 --health-check /health
```

### Add a server with a routing condition to the pool
```bash
continuity server add --pool http://my-app.domain.com --address docker-2:8080 --condition X-HEADER=srv2
```

### Add a server and remove an old server transactionally (zero downtime deployments)
```
continuity server transaction --pool POOL_HOSTNAME    # Pool hostname the server should be added to
  -address NEW_SERVER_ADDRESS:PORT                    # Address of the new server (IP or hostname)
  --remove-server OLD_SERVER_UUID                     # Address of the old server to remove
 [--health-check /healthcheck_endpoint]               # Optional header name for routing condition
 [--condition MY_HEADER=MY_VALUE]                     # Optional header value for routing condition
```
To obtain the server UUID, use the `continuity pool config POOLNAME` command (use `--json` for JSON output), see [View current configuration](#view-current-configuration) below.

Example:
```bash
continuity server transaction --pool http://my-app.domain.com --address docker-3:8080 --remove-server 123e4567-e89b-12d3-a456-426614174000 --health-check /health
```

### View current configuration
```bash
continuity pool config POOL_HOSTNAME   # Pool hostname to view configuration for
 [--json]                               # Output in JSON format
```

### Server statistics
```bash
continuity pool stats POOL_HOSTNAME    # Pool hostname to view statistics for
 [--json]                               # Output in JSON format
```

### Remove a server from a pool
```
continuity server remove --pool POOL_HOSTNAME   # Pool hostname the server should be removed from
  --server SERVER_UUID                          # UUID of the server to remove
```

Example:
```bash
continuity server remove --pool http://my-app.domain.com --server 123e4567-e89b-12d3-a456-426614174000
```

### Update a pool
```
continuity pool update POOL_HOSTNAME              # Pool hostname to update
  --health-check-interval SECONDS           # Seconds between health checks, default 10s
  --health-check-timeout SECONDS            # Health check connection timeout, default 5s
  --health-check-initial-delay SECONDS      # Initial delay on new server registration before starting health checks, default 20s
  --health-fail NUM_KO_RESPONSES_THRESHOLD  # Number of failed health checks before marking server as down, default 3
  --health-ok NUM_OK_RESPONSES_THRESHOLD    # Number of successful health checks before marking server as healthy, default 2
 [--sticky-sessions true/false]             # Enable sticky sessions
 [--sticky-method [IP|AppCookie|LBCookie] ] # Sticky session method
 [--cookie-name NAME]                       # Name of the application cookie to use for sticky sessions
```
Example:
```bash
continuity pool update http://my-app.domain.com --health-check-interval 20 --health-fail 5
```
### Delete a pool
```bash
continuity pool delete POOL_HOSTNAME   # Pool hostname to delete
```

## Server Usage

### Start the server
If you are using the docker image, you can start the server as shown in the [Docker installation section](#docker).

If you have installed the .deb or .rpm package, the server will be started automatically as a systemd service and you can manage it via systemctl:

```bash
sudo systemctl [start/stop/restart] continuity-server
```

if you have installed the statically linked binary or built from source, you can start the server with:

```bash
./continuity-server -config /path/to/config.yaml
```
If -config is not specified, the server will look for a config.yaml file in the current directory.

### Configuration file auto update

Every configuration update made via the CLI client or RESTful API is automatically persisted to the configuration file specified when starting the server.
Please note that the file is overwritten on every change, so if you are manually editing the file don't use the CLI / API at the same time to avoid losing changes.

### View server logs

The server will print logs to stdout, so if you are running it via docker you can view the logs with:

```bash
docker logs -f <container_id>
```
If you are running the server as a systemd service, you can view the logs with:

```bash
sudo journalctl -u continuity-server -f
```
If you are running the server manually, the logs will be printed to the terminal and is up to you to redirect them to a file:

```bash
./continuity-server -config /path/to/config.yaml >> /path/to/log/continuity-server.log 2>&1 &
```