# Minecart - Dockerised Minecraft Server

![](https://skillicons.dev/icons?i=golang,docker,git,markdown)

![GitHub commit activity](https://img.shields.io/github/commit-activity/w/harrisonablack/minecart) ![GitHub Closed Pull Requests](https://img.shields.io/github/issues-pr-closed/harrisonablack/minecart) ![GitHub Issues](https://img.shields.io/github/issues/harrisonablack/minecart)

## Quick Start

Minecart is configured by a single `minecart.yml` file, this describes the current server, resources and configuration. 

To create a simple [PaperMC](https://papermc.io/software/paper/) server, create a file named `minecart.yml` with the following contents;

::: code-group
```yaml [minecart.yml]
server:
  type: paper
  version: latest
  eula: true
```
:::

Then, create a `docker-compose.yml` file in the same directory with the following;

::: code-group
```yaml [docker-compose.yml]
services:
  minecart:
    image: ghcr.io/harrisonablack/minecart
    tty: true
    stdin_open: true
    ports:
      - "25565:25565"
    volumes:
      - ./minecart.yml:/app/minecart.yml:ro
      - ./data:/data
    restart: unless-stopped
```
:::

The server can then be started using;

```bash
docker compose up -d
```

To then attach to the console, use;

```bash
docker compose attach minecart
```

The server will now be availably local at `localhost:25565`

For a more verbose initial setup, please see [getting started](./getting-started.md), for further configuration, please see [configuration](./configuration.md)
