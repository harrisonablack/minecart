# Configuration

Minecart reads `minecart.yml`, `minecart.yaml`, or `minecart.json` from the current
directory. Set `MINECART_CONFIG` to select a different file. Existing application
environment overrides, such as `MINECART_SERVER_MEMORY=4G`, take precedence over
file values.

## Declaring server configuration

Declare the values Minecart should manage under `config.files`:

```yaml
directory: server

server:
  type: paper
  version: latest
  eula: true
  memory: 2G

config:
  files:
    server.properties:
      motd: "My server"
      max-players: 20
      online-mode: true
      rcon.port: 25575
      resource-pack: ""

    plugins/example/settings.properties:
      enabled: true

    config/paper-global.yml:
      proxies:
        velocity:
          enabled: true
          online-mode: true
```

Destinations are relative to `directory` (itself relative to Minecart's working
directory). Absolute paths and `..` paths escaping that directory are rejected.
Filenames and setting keys are literal: `server.properties` names one file and
`rcon.port` names one property.

### Properties files

Any `.properties` destination is supported. Each declaration must be a flat map
with non-empty keys and string, number, or boolean values. Empty string values
are allowed; nulls, lists, and nested maps are errors. Quote a value when its
exact textual representation matters, for example `"00123"`.

Properties files are read and written as UTF-8 using a Java properties library.
The library handles escaping and continuation lines. `${...}` expressions are
literal text, without property or environment interpolation.
If the library cannot serialize the merged keys and values without loss, Minecart
reports an error and preserves the original file.

### YAML files

Any `.yml` or `.yaml` destination is supported, including Paper and plugin
configuration files. Each file's declaration must be a mapping. Use nested YAML
maps for nested settings; dots in keys remain literal.

Minecart uses a separate Koanf instance for each destination, loading the
existing YAML and overlaying the declaration with Koanf's native merge behavior:

| Declaration | Result |
| --- | --- |
| Mapping over a mapping | Recursively merge; preserve undeclared children. |
| List | Replace the entire existing value, including lists of mappings. |
| Empty list (`[]`) | Replace the existing value with an empty list. |
| Explicit `null` | Store null; the key remains present. |
| Different value type | Replace the old value with the declared value. |
| Empty mapping (`{}`) over a mapping | Preserve the existing mapping's children. |

For example, declaring `plugins: [one, two]` replaces the stored list; it does
not append to it. Removing that declaration leaves the stored list alone.
Declaring `plugins: null` instead stores a null value. An empty mapping replaces
a scalar or list with an empty mapping, but does not clear an existing mapping.

Existing YAML must contain a single mapping document. Empty files and null
documents are treated as empty configurations. Malformed YAML, duplicate keys,
non-mapping roots, and multiple documents are errors. YAML comments, formatting,
and anchor notation are not preserved.

## Ownership and launch behavior

On every invocation, Minecart installs or updates the server, then applies the
declared configurations before starting the server process. Configuration edits
apply even when the installation is already current.

- Existing files are read and declared values override their stored values.
- Undeclared settings are preserved. Removing a declaration stops managing that
  setting; it does not reset or delete its stored value.
- Missing files and parent directories are created. Omitting `config.files`
  means no configuration files are managed.
- Minecraft, Paper, and plugins supply their own defaults. Minecart does not
  validate Minecraft-specific setting names or supply default configuration
  schemas; spelling a setting name correctly is your responsibility.
- Comments and formatting may change when a managed file is rewritten.
- Read, parse, merge, and write errors abort the launch. Correct the problem and
  invoke Minecart again to retry configuration application.
- Each file is replaced atomically, preserving existing file permissions. New
  files use `0644` restricted by the process umask. Files are processed in filename
  order; successful earlier writes remain if a later file fails. The installation
  manifest records installation independently of configuration application.
