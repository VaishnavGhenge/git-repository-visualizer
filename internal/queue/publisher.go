package queue

import (
	"context"
	"fmt"
	"time"

	"git-repository-visualizer/internal/redis"

	"github.com/google/uuid"
)

// IPublisher defines the interface for publishing jobs to the queue
type IPublisher interface {
	PublishIndexJob(ctx context.Context, repoID int) error
	PublishUpdateJob(ctx context.Context, repoID int) error
	PublishDiscoverJob(ctx context.Context, userID int64, provider string) error
	GetQueueLength(ctx context.Context) (int64, error)
}

type publisherImpl struct {
	queue *Queue
}

// NewPublisher creates a publisher
func NewPublisher(redisClient *redis.Client, queueName string) IPublisher {
	return &publisherImpl{
		queue: NewQueue(redisClient, queueName),
	}
}

// PublishIndexJob creates a job to index a repository
func (p *publisherImpl) PublishIndexJob(ctx context.Context, repoID int) error {
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
func (p *publisherImpl) PublishUpdateJob(ctx context.Context, repoID int) error {
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

// PublishDiscoverJob creates a job to discover repositories for a user
func (p *publisherImpl) PublishDiscoverJob(ctx context.Context, userID int64, provider string) error {
	job := &Job{
		ID:           uuid.New().String(),
		RepositoryID: 0, // Not tied to a specific repo yet
		Type:         JobTypeDiscover,
		Payload: map[string]interface{}{
			"user_id":  userID,
			"provider": provider,
		},
		CreatedAt:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
	}

	if err := p.queue.Push(job); err != nil {
		return fmt.Errorf("failed to publish discover job: %w", err)
	}

	return nil
}

// GetQueueLength returns current queue size
func (p *publisherImpl) GetQueueLength(ctx context.Context) (int64, error) {
	length, err := p.queue.Length(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}
