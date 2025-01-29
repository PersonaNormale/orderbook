package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"orderbook/internal/api" // adjust this import path
	"orderbook/internal/orderbook"
)

const (
	defaultPort = ":8080"
)

func main() {
	// Initialize orderbook
	book := orderbook.NewOrderBook("MAIN")

	// Initialize handler
	handler := api.NewHandler(book)

	// Initialize router
	router := api.NewRouter(handler)
	mux := router.SetupRoutes()

	// Create server
	server := &http.Server{
		Addr:    defaultPort,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", defaultPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down server...")
}
