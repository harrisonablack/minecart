# Plugins & Resources

> [!IMPORTANT]
> Minecart is currently in early development. Resource management is currently limited to Modrinth plugins on `paper` servers and will be expanded to additional server types, providers, and resource types (such as mods and datapacks) in the future.

For [Paper](./server-types/paper.md) servers, Minecart can resolve, download, and update plugins alongside the server jar. Declare plugins under `resources.plugins` in `minecart.yml`:

```yaml
server:
  type: paper
  version: latest
  eula: true

resources:
  sync: false
  plugins:
    - name: luckperms
      provider: modrinth
    - name: viaversion
      provider: modrinth
      version: 5.2.1
      channel: beta
```

`resources` is the top-level block for external server add-ons, with `plugins` as the first supported resource type.

## Plugin options

Each entry in `resources.plugins` supports the following fields:

- **`name`** *(required)* — The project slug on Modrinth (for example, `https://modrinth.com/plugin/luckperms` uses `luckperms`).
- **`provider`** *(required)* — The upstream source. Currently, only `modrinth` is supported.
- **`version`** *(optional)* — A specific version number to pin (e.g. `"5.2.1"`). Omit this field or set it to `latest` to automatically resolve the newest version compatible with your server's Minecraft version and the `paper` loader.
- **`channel`** *(optional, default `stable`)* — The minimum stability threshold used when resolving `latest`:
  - `stable` — Accepts `release` versions only.
  - `beta` — Accepts `release` and `beta` versions.
  - `alpha` — Accepts `release`, `beta`, and `alpha` versions.

## Updates and `sync` behavior

Plugins are installed into `<directory>/plugins/` and tracked in the `.manifest` file inside the server directory:

- On startup, if a declared plugin is already recorded in `.manifest` at the resolved version and its jar file passes checksum verification, Minecart skips downloading it.
- When a plugin updates to a new version, Minecart downloads the new jar, updates `.manifest`, and removes the previous jar file.

### Removing undeclared plugins (`sync`)

The `resources.sync` option controls what happens when you remove a plugin from `minecart.yml`:

- **`sync: false` (default):** Previously installed plugins are left in `<directory>/plugins/`.
- **`sync: true`:** Any plugin tracked in `.manifest` that is no longer declared in `minecart.yml` is removed from `<directory>/plugins/` and `.manifest`. Unmanaged files that you placed in `plugins/` manually are never deleted.

::: warning Server type support
Plugins are only installed when `server.type` is `paper`. Declaring `resources.plugins` on any other server type logs a warning and skips plugin installation.
:::
