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
- Each file is replaced atomically, preserving existing file permissions. Files
  are processed in filename order; successful earlier writes remain if a later
  file fails. The installation manifest records installation independently of
  configuration application.
