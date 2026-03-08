package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	transport "messenger-server/internal/transport"
)

func main() {
	cfg := transport.DefaultConfig()
	logger := log.New(os.Stdout, "[server] ", log.LstdFlags)
	srv := transport.New(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Fatalf("server stopped with error: %v", err)
	}

	logger.Println("server stopped")
}
