package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"git-repository-visualizer/internal/redis"
)

// Consumer handles consuming jobs from Redis
type Consumer struct {
	queue       *Queue
	handler     JobHandler // Interface to process jobs
	concurrency int        // Number of goroutines
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// JobHandler interface - you'll implement this
type JobHandler interface {
	HandleJob(ctx context.Context, job *Job) error
}

// NewConsumer creates a consumer
func NewConsumer(
	redisClient *redis.Client,
	queueName string,
	handler JobHandler,
	concurrency int,
) *Consumer {
	return &Consumer{
		queue:       NewQueue(redisClient, queueName),
		handler:     handler,
		concurrency: concurrency,
		stopChan:    make(chan struct{}),
	}
}

// Start begins consuming jobs (runs goroutines)
func (c *Consumer) Start(ctx context.Context) error {
	if c.concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1")
	}

	log.Printf("Starting consumer with %d workers", c.concurrency)

	for i := 0; i < c.concurrency; i++ {
		c.wg.Add(1)
		go c.worker(ctx, i)
	}

	return nil
}

// worker is a goroutine that processes jobs from the queue
func (c *Consumer) worker(ctx context.Context, id int) {
	defer c.wg.Done()

	log.Printf("Worker %d started", id)

	for {
		select {
		case <-c.stopChan:
			log.Printf("Worker %d stopping", id)
			return
		case <-ctx.Done():
			log.Printf("Worker %d context cancelled", id)
			return
		default:
			// Pop job with 5 second timeout
			job, err := c.queue.Pop(ctx, 5*time.Second)
			if err != nil {
				// Timeout or Redis error - continue polling
				if err == context.DeadlineExceeded || err.Error() == "redis: nil" {
					continue
				}
				log.Printf("Worker %d error popping job: %v", id, err)
				continue
			}

			if job == nil {
				continue
			}

			log.Printf("Worker %d processing job %s (type: %s, repo: %d)", id, job.ID, job.Type, job.RepositoryID)

			// Handle the job
			if err := c.handler.HandleJob(ctx, job); err != nil {
				log.Printf("Worker %d failed to handle job %s: %v", id, job.ID, err)
				// TODO: Implement retry logic or dead letter queue
			} else {
				log.Printf("Worker %d completed job %s", id, job.ID)
			}
		}
	}
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	log.Println("Stopping consumer...")
	close(c.stopChan)
	c.wg.Wait()
	log.Println("Consumer stopped")
}
