package main

import (
	"context"
	"os"

	"github.com/harrisonablack/minecart/internal/apply"
	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/install"
	"github.com/harrisonablack/minecart/internal/log"
	"github.com/harrisonablack/minecart/internal/run"
)

func main() {
	logger := log.New(os.Stderr)

	cfg, err := config.LoadDefault()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if err := install.Run(ctx, logger, *cfg); err != nil {
		logger.Error("install", "error", err)
		os.Exit(1)
	}

	if err := apply.Run(logger, *cfg); err != nil {
		logger.Error("apply configuration", "error", err)
		os.Exit(1)
	}

	if err := run.Server(ctx, logger, *cfg); err != nil {
		logger.Error("run", "error", err)
		os.Exit(1)
	}
}
