// @title Semantic Video API
// @version 1.0
// @description API for managing video indexing, frame extraction, and cloud uploads.
// @host localhost:8080
// @BasePath /
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"semanticvideo/internal/daemon"
	_ "semanticvideo/internal/docs"
)

func main() {
	server := daemon.NewServer()
	addr := ":8080"
	srv := &http.Server{Addr: addr, Handler: server.Routes()}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		log.Printf("Shutting down server on %s\n", addr)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		server.Cleanup()
	}()

	log.Printf("Starting server on %s\n", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
	server.Cleanup()
}
