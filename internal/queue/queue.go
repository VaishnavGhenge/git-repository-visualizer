package queue

import (
	"context"
	"fmt"
	"git-repository-visualizer/internal/redis"
	"time"
)

// Queue wraps Redis operations for job queue
type Queue struct {
    redis     *redis.Client
    queueName string  // e.g., "git_index_jobs"
}

// NewQueue creates a queue instance
func NewQueue(redis *redis.Client, queueName string) *Queue {
    return &Queue{
        redis:     redis,
        queueName: queueName,
    }
}

// Push adds a job to the queue (RPUSH)
func (q *Queue) Push(job *Job) error {
    ctx := context.Background()
    return q.redis.RPush(ctx, q.queueName, job).Err()
}

// Pop removed and return a job (BLOP with timeout)
func (q *Queue) Pop(ctx context.Context, timeout time.Duration) (*Job, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    result, err := q.redis.BLPop(ctx, timeout, q.queueName).Result()
    if err != nil {
        return nil, err
    }

	if len(result) < 2 {
		return nil, fmt.Errorf("unexpected BLPop result length: %d", len(result))
	}

    return FromJSON(result[1])
}

// Length returns queue size (LLEN)
func (q *Queue) Length(ctx context.Context) (int64, error) {
    return q.redis.LLen(ctx, q.queueName).Result()
}