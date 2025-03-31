package main

import (
	"flag"
	"fmt"
	"github.com/MohammaedAlani/Mongo2ES/internal/config"
	"github.com/MohammaedAlani/Mongo2ES/internal/service"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "./config/config.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create service
	svc, err := service.NewMigrationService(cfg)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the service
	go func() {
		if err := svc.Start(); err != nil {
			log.Fatalf("Service error: %v", err)
		}
	}()

	fmt.Println("MongoDB to Elasticsearch migration service started")
	fmt.Printf("Metrics available at http://localhost:%d/metrics\n", cfg.App.MetricsPort)
	fmt.Printf("Health check available at http://localhost:%d/health\n", cfg.App.HealthCheckPort)

	// Wait for termination signal
	sig := <-sigChan
	fmt.Printf("Received signal %s, shutting down\n", sig)

	// Shutdown the service
	if err := svc.Shutdown(); err != nil {
		log.Fatalf("Error during shutdown: %v", err)
	}

	fmt.Println("Service shutdown complete")
}
