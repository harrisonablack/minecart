ARG JAVA_VERSION=27

# ---- Build stage: compile the minecart binary ----
FROM golang:1.27-alpine AS build

# Version baked into the binary; set by the release workflow.
ARG VERSION=dev

WORKDIR /src

# Download module dependencies first so they cache across builds.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the source and build a static binary.
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X github.com/harrisonablack/minecart/internal/version.Version=${VERSION}" \
      -o /out/minecart ./cmd/minecart

# ---- Runtime stage: minimal image with the requested Java version ----
FROM eclipse-temurin:${JAVA_VERSION}-jre

# Run as an unprivileged user.
RUN useradd --system --uid 10001 --create-home minecart

WORKDIR /app

COPY --from=build /out/minecart /usr/local/bin/minecart

ENV MINECART_DIRECTORY=/data
RUN mkdir -p /data && chown minecart:minecart /data
VOLUME ["/data"]

EXPOSE 25565
USER minecart

ENTRYPOINT ["minecart"]
