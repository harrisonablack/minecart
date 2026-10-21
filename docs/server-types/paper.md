# Paper

The `paper` server type installs a [Paper](https://papermc.io/software/paper) server jar from the PaperMC Downloads API. Paper is a high-performance fork of the Minecraft server that includes optimizations, configuration options, and support for Bukkit/Spigot/Paper plugins.

## Example configuration

```yaml [minecart.yml]
server:
  type: paper
  version: latest
  eula: true
  memory: 4G

resources:
  plugins:
    - name: luckperms
      provider: modrinth

config:
  files:
    server.properties:
      motd: "Paper Server"
      simulation-distance: 10
    config/paper-global.yml:
      proxies:
        velocity:
          enabled: false
```

## Version resolution

Minecart resolves Paper builds via the PaperMC v3 API and verifies the downloaded jar against its SHA-256 checksum.

- **`version: latest`** resolves to the newest Minecraft version that has a **`STABLE`** Paper build. When a new Minecraft release only has experimental Paper builds available, Minecart steps back through earlier versions until it finds one with a stable build.
- **Pinned versions** (for example, `version: "1.21.4"`) resolve to the latest `STABLE` Paper build for that specific Minecraft version.

## Plugins & configuration

- **Plugins:** Paper servers support automatic plugin installation under `resources.plugins`. See [Plugins & Resources](../plugins.md) for details.
- **Configuration files:** Paper's YAML configuration files (such as `config/paper-global.yml` and `config/paper-world-defaults.yml`) can be managed under `config.files` alongside `server.properties`.
