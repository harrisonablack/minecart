# Vanilla

The `vanilla` server type installs the official Mojang Minecraft server jar. Use `vanilla` when you want unmodified gameplay mechanics without third-party patches or plugins.

## Example configuration

```yaml [minecart.yml]
server:
  type: vanilla
  version: latest
  eula: true
  memory: 2G

config:
  files:
    server.properties:
      motd: "Vanilla Minecraft Server"
      difficulty: normal
      max-players: 10
```

## Version resolution

Minecart queries Mojang's official version manifest (`version_manifest_v2.json`) and verifies the downloaded server jar against Mojang's SHA-1 hash.

- **`version: latest`** resolves to the version marked as `latest.release` in Mojang's manifest.
- **Pinned versions** (for example, `version: "1.21.4"`) resolve to that exact release entry in the manifest.

## Plugins

Vanilla servers do not support plugins. Declaring `resources.plugins` with `server.type: vanilla` logs a warning and skips plugin installation.
