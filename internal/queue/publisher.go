package queue

import (
	"context"
	"fmt"
	"time"

	"git-repository-visualizer/internal/redis"

	"github.com/google/uuid"
)

type Publisher struct {
	queue *Queue
}

// NewPublisher creates a publisher
func NewPublisher(redisClient *redis.Client, queueName string) *Publisher {
	return &Publisher{
		queue: NewQueue(redisClient, queueName),
	}
}

// PublishIndexJob creates a job to index a repository
func (p *Publisher) PublishIndexJob(ctx context.Context, repoID int) error {
	job := &Job{
		ID:           uuid.New().String(),
		RepositoryID: int64(repoID),
		Type:         JobTypeIndex,
		Payload:      make(map[string]interface{}),
		CreatedAt:    time.Now(),
		Retries:      0,
		MaxRetries:   3,
	}

	if err := p.queue.Push(job); err != nil {
		return fmt.Errorf("failed to publish index job: %w", err)
	}

	return nil
}

// PublishUpdateJob creates a job to update a repository
func (p *Publisher) PublishUpdateJob(ctx context.Context, repoID int) error {
	job := &Job{
		ID:           uuid.New().String(),
		RepositoryID: int64(repoID),
		Type:         JobTypeUpdate,
		Payload:      make(map[string]interface{}),
		CreatedAt:    time.Now(),
		Retries:      0,
		MaxRetries:   3,
	}

	if err := p.queue.Push(job); err != nil {
		return fmt.Errorf("failed to publish update job: %w", err)
	}

	return nil
}

// GetQueueLength returns current queue size
func (p *Publisher) GetQueueLength(ctx context.Context) (int64, error) {
	length, err := p.queue.Length(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}
