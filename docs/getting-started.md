# Getting Started

> [!IMPORTANT]
> Minecart is currently in early development and is subject to breaking changes. Features such as automatic plugin installation are currently limited to `paper` and will be expanded to additional server types and resource providers in the future.

Minecart is a declarative wrapper for running Minecraft servers in Docker. Instead of managing startup scripts or long lists of environment variables, you define your server version, plugins, and configuration overrides in a single `minecart.yml` file. On startup, Minecart resolves and downloads the required files, applies your configuration, and launches the server process.

## Prerequisites

### Docker

Minecart is packaged and distributed as a Docker container. You can verify that Docker is installed and running on your host with:

```bash
docker run hello-world
```

If you need to install Docker:
- **macOS / Windows:** Install [Docker Desktop](https://www.docker.com/get-started/).
- **Linux:** Install the [Docker Engine](https://docs.docker.com/engine/install/) for your distribution.

### Architecture

Published Docker images are built for both **`linux/amd64`** (x86_64) and **`linux/arm64`** (Apple Silicon, Raspberry Pi, AWS Graviton). Docker automatically pulls the matching manifest for your host architecture.

### Java version

The published `ghcr.io/harrisonablack/minecart` image currently ships with **Java 27** (`eclipse-temurin:27-jre`) only.

If you want to run an older version of Minecraft that requires an earlier Java release (such as Java 21, 17, or 8), you will need to build the image locally using the `JAVA_VERSION` build argument:

```bash
git clone https://github.com/harrisonablack/minecart.git
cd minecart
docker build --build-arg JAVA_VERSION=21 -t minecart:java21 .
```

You can then reference `image: minecart:java21` in your `docker-compose.yml`.

> [!NOTE]
> Publishing images with prior Java LTS is currently planned, however this will most likely be implemented once we reach an early stable state. Please see [issue](https://github.com/harrisonablack/minecart/issues/32) 

## Pick an example config

Create a new directory for your server and add a `minecart.yml` file. Below are two starter configurations:

::: code-group
```yaml [Paper]
server:
  type: paper
  version: latest
  eula: true
  memory: 6G

config:
  files:
    server.properties:
      motd: "Paper Server"
      max-players: 99
      difficulty: hard
      view-distance: 32
```

```yaml [Vanilla]
server:
  type: vanilla
  version: latest
  eula: true
  memory: 2G

config:
  files:
    server.properties:
      motd: "Vanilla Minecraft"
      max-players: 10
```
:::

A quick breakdown of these fields:
- **`type`** selects the server platform: [`paper`](./server-types/paper.md) or [`vanilla`](./server-types/vanilla.md).
- **`version: latest`** resolves the newest stable Minecraft release. You can also pin a specific version string, such as `"1.21.4"`.
- **`eula: true`** accepts the [Minecraft EULA](https://www.minecraft.net/en-us/eula) and writes `eula=true` to `eula.txt`.
- **`memory`** sets the maximum JVM heap size (`-Xmx`).

## Run it

In the same directory as your `minecart.yml`, create a `docker-compose.yml` file:

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

The container uses two volume mounts:
1. `./minecart.yml:/app/minecart.yml:ro` mounts your configuration file read-only into Minecart's working directory (`/app`).
2. `./data:/data` persists the server installation, world data, plugins, and generated configuration files in `./data` on the host.

> [!NOTE]
> The container runs as an unprivileged user (`UID 10001`). On Linux hosts, if `./data` does not exist yet, Docker will create it as `root` and Minecart will fail with a permission error. Create the directory and set its ownership before starting (e.g. `mkdir -p data && sudo chown 10001:10001 data`), or use a named Docker volume instead.

Start the container in the background:

```bash
docker compose up -d
```

## First launch

Each time the container starts, Minecart runs through four stages:

1. **Load config:** Reads `/app/minecart.yml` and applies any `MINECART_*` environment variable overrides.
2. **Install server & plugins:** Queries the upstream API (Mojang or PaperMC, plus Modrinth for plugins), downloads any missing or outdated artifacts into `/data`, verifies their checksums, and records the installed state in `/data/.manifest`. If the installation already matches `.manifest`, downloads are skipped.
3. **Apply configuration:** Writes `eula.txt` and merges any settings declared under `config.files` into the target `.properties` or `.yml` files in `/data`, preserving any existing settings you haven't declared.
4. **Run server:** Starts the JVM inside a pseudo-terminal (PTY) and forwards signals and standard I/O.

### Logs and console access

To follow the startup logs:

```bash
docker compose logs -f
```

Once the server finishes generating the world and logs `Done (...)! For help, type "help"`, you can connect at `localhost:25565`.

Because `tty: true` and `stdin_open: true` are enabled in `docker-compose.yml`, you can attach directly to the interactive server console to run commands (`op`, `whitelist`, etc.):

```bash
docker compose attach minecart
```

::: tip Detaching from the console
Pressing `Ctrl+C` while attached sends an interrupt signal and shuts the server down. To detach from the console while leaving the server running, press **`Ctrl+P` followed by `Ctrl+Q`**.
:::

### Stopping and restarting

To apply changes made to `minecart.yml` or stop the server:

```bash
docker compose restart
# or to stop and remove the container:
docker compose down
```

When Docker stops the container, Minecart intercepts the signal and forwards `SIGTERM` to the JVM so the server saves the world cleanly before exiting.
