# Configuration

> [!IMPORTANT]
> Minecart is currently in early development and its configuration schema may change between releases. Features such as `resources.plugins` are currently limited to `paper` servers and will be expanded to additional server types and resource types in the future.

Minecart loads its configuration from `minecart.yml`, `minecart.yaml`, or `minecart.json` in the current working directory. You can set `MINECART_CONFIG` to specify a different file path.

## Server settings

The `server` block defines the Minecraft server to install and the arguments used to launch the JVM:

```yaml
server:
  type: paper
  version: latest
  eula: true
  memory: 4G
  jvm_flags:
    - "-XX:+UseG1GC"
  args:
    - "nogui"
```

- **`type`** *(required)* — The server platform: [`paper`](./server-types/paper.md) or [`vanilla`](./server-types/vanilla.md).
- **`version`** *(required)* — The Minecraft version to install (e.g. `"1.21.4"`), or `latest` for the newest stable release.
- **`eula`** *(required)* — Must be `true` to accept the [Minecraft EULA](https://www.minecraft.net/en-us/eula). Minecart writes `eula.txt` prior to launch.
- **`memory`** *(optional)* — Maximum JVM heap size, passed as `-Xmx<memory>` (for example, `2G` or `4096M`).
- **`jvm_flags`** *(optional)* — Additional JVM flags placed before `-jar`.
- **`args`** *(optional)* — Additional arguments passed to the server after the jar filename (such as `["nogui"]`).

### Server directory

By default, Minecart installs and runs the server in `./server` relative to its working directory. In the official Docker image, `MINECART_DIRECTORY=/data` is set by default so all server data lives in `/data`.

When running outside the default container layout, you can set `directory` at the top level of `minecart.yml`:

```yaml
directory: server
```

### Environment variable overrides

Configuration fields can be overridden using `MINECART_` environment variables, which take precedence over file values. Dots in configuration paths map to underscores:

- `MINECART_DIRECTORY=/data` $\rightarrow$ `directory`
- `MINECART_SERVER_TYPE=paper` $\rightarrow$ `server.type`
- `MINECART_SERVER_VERSION=latest` $\rightarrow$ `server.version`
- `MINECART_SERVER_MEMORY=4G` $\rightarrow$ `server.memory`

## Managing server files (`config.files`)

You can declare server and plugin configuration values under `config.files`. Before every launch, Minecart merges these values into the corresponding files inside the server directory:

```yaml
server:
  type: paper
  version: latest
  eula: true

config:
  files:
    server.properties:
      motd: "My Minecraft Server"
      max-players: 20
      online-mode: true
      rcon.port: 25575
      resource-pack: ""

    config/paper-global.yml:
      proxies:
        velocity:
          enabled: true
          online-mode: true

    plugins/example/settings.properties:
      enabled: true
```

### Merge behavior and file ownership

Configuration updates run on every startup, even when the server jar itself is already up to date:

- **Declared keys override existing values:** Minecart reads the target file on disk, overlays the keys declared in `minecart.yml`, and writes the result atomically.
- **Undeclared keys are preserved:** Any settings present in the file on disk that are not listed in `minecart.yml` remain untouched. Removing a key from `minecart.yml` stops managing that setting—it does not delete or reset its value on disk.
- **Missing files are created:** If a target file or its parent directories do not exist yet, Minecart creates them before the server starts; Minecraft, Paper, or the plugin will populate its remaining defaults on boot.
- **Paths are relative to `directory`:** Absolute paths and `..` segments that escape the server directory are rejected.
- **No schema validation:** Minecart does not validate Minecraft- or plugin-specific key names; keys are written exactly as declared.

### `.properties` files

Any `.properties` file is supported. Declarations must be a flat map with non-empty keys and string, number, or boolean values (empty strings `""` are allowed; lists, nested maps, and `null` values return an error).

```yaml
config:
  files:
    server.properties:
      motd: "A Minecraft Server"
      pvp: true
      spawn-protection: 0
      rcon.port: 25575
```

- Dots in property keys (such as `rcon.port`) are treated as literal key names.
- `${...}` expressions are written as literal text without variable interpolation.
- Quote values when their exact string formatting matters (for example, `"00123"`).

### `.yml` and `.yaml` files

Any `.yml` or `.yaml` file is supported, including Paper (`config/paper-global.yml`) and plugin configuration files. Use nested YAML maps for nested settings; dots inside a key name remain literal.

```yaml
config:
  files:
    config/paper-global.yml:
      chunk-loading-basic:
        player-max-chunk-load-rate: 100.0
      spam-limiter:
        recipe-spam-increment: 1
        tab-spam-increment: 1
```

When merging declared YAML values over an existing file on disk:

| Declared value | Merge result |
| --- | --- |
| **Mapping over a mapping** | Recursively merges keys and preserves undeclared child keys. |
| **List** (e.g. `[one, two]` or `[]`) | Replaces the entire existing list rather than appending. |
| **Scalar or type change** | Replaces the existing value with the declared value. |
| **Explicit `null`** | Keeps the key present with a `null` value. |
| **Empty mapping (`{}`) over a mapping** | Preserves the existing mapping's children. |

::: warning YAML formatting and comments
Target YAML files must contain a single mapping document. When Minecart updates a managed `.yml` or `.yaml` file, the document is re-serialized, so comments, custom formatting, and YAML anchors are not preserved.
:::
