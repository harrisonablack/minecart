# minecart

A declarative Minecraft server installer and launcher. Describe a server
(type, version, mods) in a config file, and minecart resolves, downloads,
verifies, and runs it.

## Quick start

1. Build the binary:

   ```sh
   go build -o minecart ./cmd/minecart
   ```

2. Create a config from the template:

   ```sh
   cp minecart.example.yml minecart.yml
   ```

3. Edit `minecart.yml` (at minimum, set the Minecraft `version` and accept the
   EULA with `eula: true`).

4. Run it:

   ```sh
   ./minecart
   ```

Minecart downloads the server jar into `server/`, verifies it, writes
`eula.txt`, and starts the server. Press `Ctrl-C` to stop it.

## Configuration

Configuration is read from `minecart.yml` (YAML or JSON), with environment
variables (`MINECART_*`) overriding file values. See `minecart.example.yml`
for the full schema.

| Field | Description |
| --- | --- |
| `directory` | Where to install/run the server (default `server`) |
| `server.type` | `vanilla` (more types coming) |
| `server.version` | Minecraft version |
| `server.eula` | Must be `true` to accept the Minecraft EULA |
| `server.memory` | Java heap size (e.g. `2G`) |
| `server.args` | Extra arguments passed to the server |

## Development

```sh
go build ./...
go test ./...
golangci-lint run
```
