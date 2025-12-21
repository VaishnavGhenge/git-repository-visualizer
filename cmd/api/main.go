package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git-repository-visualizer/internal/config"
	"git-repository-visualizer/internal/database"
	internalHttp "git-repository-visualizer/internal/http"
	"git-repository-visualizer/internal/queue"
	"git-repository-visualizer/internal/redis"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize context
	ctx := context.Background()

	// Connect to database
	log.Println("Connecting to database...")
	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Database connected successfully")

	// Connect to Redis
	log.Println("Connecting to Redis...")
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Redis connected successfully")

	// Initialize queue publisher
	publisher := queue.NewPublisher(redisClient, cfg.Redis.QueueName)
	log.Printf("Queue publisher initialized with queue: %s", cfg.Redis.QueueName)

	// Initialize HTTP handler with dependencies
	h := internalHttp.NewHandler(db, publisher, cfg)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s...", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
