package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/substratusai/sandboxai/go/sandboxaid/client/docker"
	"github.com/substratusai/sandboxai/go/sandboxaid/handler"
)

const (
	defaultDefaultImage = "substratusai/sandboxai-box:v0.2.0"
)

func main() {
	host, ok := os.LookupEnv("SANDBOXAID_HOST")
	if !ok {
		host = "127.0.0.1"
	}
	port, ok := os.LookupEnv("SANDBOXAID_PORT")
	if !ok {
		port = "5266"
	}
	// SCOPE limits the containers that this server will manage.
	// It does this by labelling containers that it creates with the
	// scope value.
	// TODO: This is currently only implemented for the cleanup flow...
	//       Consider implementing for all operations (i.e. GET/DELETE /sandboxes/{id}).
	scope, ok := os.LookupEnv("SANDBOXAID_SCOPE")
	if !ok {
		scope = "default"
	}
	var deleteOnShutdown bool
	if val, ok := os.LookupEnv("SANDBOXAID_DELETE_ON_SHUTDOWN"); ok {
		deleteOnShutdown = strings.ToLower(strings.TrimSpace(val)) == "true"
	}

	defaultImage, ok := os.LookupEnv("SANDBOXAID_DEFAULT_IMAGE")
	if !ok {
		defaultImage = defaultDefaultImage
	}

	log := log.New(os.Stderr, "", log.LstdFlags)
	handler.SetLogger(log)
	docker.SetLogger(log)

	client, err := docker.NewSandboxClient(nil, &http.Client{}, scope)
	if err != nil {
		log.Fatalf("Failed to create sandbox client: %v", err)
	}

	// Cleanup on shutdown if specified (useful for embedded mode).
	// This is important for handling sandboxes that were created but not yet deleted.
	// The most likely scenario for this to happen would be when a client launches a
	// sandbox outside of a mechanism that catches shutdown signals, and never calls
	// DELETE on the sandbox it created:
	// ```py
	// box = Sandbox(timeout=60)
	// # Ctrl-C
	// ```
	if deleteOnShutdown {
		defer func() {
			log.Print("Cleanup: ensuring all sandboxes at deleted")
			cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancelCleanup()
			refs, err := client.ListAllSandboxes(cleanupCtx)
			if err != nil {
				log.Printf("Cleanup: failed to list sandbox IDs: %v", err)
				return
			}
			if len(refs) == 0 {
				log.Print("Cleanup: no sandboxes to delete")
				return
			}
			for i, ref := range refs {
				log.Printf("Cleanup: deleting %d/%d: sandbox %s", i+1, len(refs), refs)
				if err := client.DeleteSandbox(context.Background(), ref.Space, ref.Name); err != nil {
					log.Printf("Cleanup: failed to delete sandbox %q: in space %q: %v", ref.Name, ref.Space, err)
					return
				}
			}
			log.Printf("Cleanup: done deleting sandboxes (total = %d)", len(refs))
		}()
	}

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", host, port),
		Handler: handler.NewHandler(client, defaultImage),
	}

	go func() {
		ln, err := net.Listen("tcp", server.Addr)
		if err != nil {
			log.Fatalf("Failed to listen on address %s: %v", server.Addr, err)
		}
		addr := ln.Addr().(*net.TCPAddr)
		if port == "0" {
			// If "any free port" was specified, output the selected port.
			if err := json.NewEncoder(os.Stdout).Encode(serverInfo{Host: addr.IP.String(), Port: addr.Port}); err != nil {
				log.Fatalf("Failed to output server info: %v", err)
			}
		}
		log.Printf("Listening on address %s, starting HTTP server", addr.String())
		if err := server.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
		log.Print("Stopped serving new connections")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	gracePeriod := 30 * time.Second
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), gracePeriod)
	defer shutdownRelease()

	log.Printf("Received %v signal, shutting down with %s grace period...", sig, gracePeriod)

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Error shutting down HTTP server: %v", err)
	}
	log.Print("Graceful shutdown complete")
}

// serverInfo is outputted to stdout so that the program that started the server can determine
// the address it is listening on when ports are auto-selected.
type serverInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}
