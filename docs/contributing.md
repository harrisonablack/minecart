# Contributing

Contributions, bug reports, and feature requests are welcome. This guide covers how to build, test, and work on Minecart locally.

## Development setup

To build and test Minecart locally, you will need:

- **Go** (version specified in `go.mod`, currently Go 1.27)
- **Java** (any recent JRE/JDK if you want to run a Minecraft server process locally)
- **Node.js 22+** (required for building or previewing the VitePress documentation)

Clone the repository and fetch the Go module dependencies:

```bash
git clone https://github.com/harrisonablack/minecart.git
cd minecart
go mod download
```

## Build and test

Use the standard Go toolchain to run tests, format code, and build the binary:

```bash
# Run all unit tests
go test ./...

# Format source files (enforced by CI)
gofmt -w .

# Compile the minecart binary
go build -o minecart ./cmd/minecart
```

CI also runs [`golangci-lint`](https://golangci-lint.run/). If you have it installed locally, run `golangci-lint run` before opening a pull request.

::: info Pull Request titles
Pull request titles are checked by CI and must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification (for example, `feat: add curseforge provider` or `fix: handle empty properties file`).
:::

## Contributing to the docs

The documentation lives in `docs/` and is built with [VitePress](https://vitepress.dev/).

To run the local development server with hot reload:

```bash
npm install
npm run docs:dev
```

To test the production documentation build:

```bash
npm run docs:build
```
