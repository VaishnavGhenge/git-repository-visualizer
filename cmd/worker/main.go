package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"git-repository-visualizer/internal/config"
	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/queue"
	"git-repository-visualizer/internal/redis"
	"git-repository-visualizer/internal/worker"

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Create job handler
	handler := worker.NewJobHandler(db, cfg.Worker.StoragePath)

	// Create consumer
	consumer := queue.NewConsumer(
		redisClient,
		cfg.Redis.QueueName,
		handler,
		cfg.Worker.Concurrency,
	)

	// Start consumer
	log.Printf("Starting worker with %d concurrent workers...", cfg.Worker.Concurrency)
	if err := consumer.Start(ctx); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")
	cancel()
	consumer.Stop()

	log.Println("Worker exited")
}
