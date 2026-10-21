# Minecart - Dockerised Minecraft Server

![](https://skillicons.dev/icons?i=golang,docker,git,markdown)

![GitHub commit activity](https://img.shields.io/github/commit-activity/w/harrisonablack/minecart)
![GitHub Closed Pull Requests](https://img.shields.io/github/issues-pr-closed/harrisonablack/minecart)
![GitHub Issues](https://img.shields.io/github/issues/harrisonablack/minecart)


> [!IMPORTANT]
> Minecart is currently in very early development and is subject to frequent breaking changes

## Configuration

Start with [`minecart.example.yml`](minecart.example.yml). Declare server settings
under `config.files` to merge them into configuration files before every launch.
Undeclared values are preserved; removing a declaration leaves its stored value
in place.

See the [configuration documentation](docs/configuration.md) for supported formats,
examples, and launch behavior.
