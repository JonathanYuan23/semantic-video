// @title Semantic Video API
// @version 1.0
// @description API for managing video indexing, frame extraction, and cloud uploads.
// @host localhost:8080
// @BasePath /
package main

import (
	"log"
	"net/http"

	"semanticvideo/internal/daemon"
	_ "semanticvideo/internal/docs"
)

func main() {
	server := daemon.NewServer()
	addr := ":8080"
	log.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
