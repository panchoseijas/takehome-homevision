package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/panchoseijas/takehome-homevision/backend/internal/httpapi"
	"github.com/panchoseijas/takehome-homevision/backend/internal/vision"
)

const (
	defaultAddr     = ":8080"
	readTimeout     = 10 * time.Second
	writeTimeout    = 30 * time.Second
	idleTimeout     = 60 * time.Second
	shutdownTimeout = 10 * time.Second
)

func main() {
	addr := flag.String("addr", defaultAddr, "TCP address for the HTTP server to listen on")
	maxConcurrent := flag.Int("max-concurrent", runtime.GOMAXPROCS(0), "maximum detections running at once")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *addr, *maxConcurrent, logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, addr string, maxConcurrent int, logger *slog.Logger) error {
	server := newServer(addr, maxConcurrent, logger)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	logger.Info("server listening", "addr", listener.Addr().String())

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func newServer(addr string, maxConcurrent int, logger *slog.Logger) *http.Server {
	params := vision.DefaultParams()
	handler := httpapi.New(vision.NewDetector(params), httpapi.Config{
		MaxPixels:     params.MaxPixels,
		MaxConcurrent: maxConcurrent,
		Logger:        logger,
	})
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}
