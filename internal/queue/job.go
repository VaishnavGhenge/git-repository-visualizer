package queue

import (
	"encoding/json"
	"time"
)

type JobType string

// JobType constants - different types of worker jobs
const (
	JobTypeIndex    = JobType("index")
	JobTypeUpdate   = JobType("update")
	JobTypeDelete   = JobType("delete")
	JobTypeDiscover = JobType("discover")
)

// Job represents a unit of work
type Job struct {
	ID           string
	RepositoryID int64
	Type         JobType
	Payload      map[string]interface{}
	CreatedAt    time.Time
	Retries      int
	MaxRetries   int
}

// ToJSON - Convert job to JSON string for Redis storage
func (j *Job) ToJSON() (string, error) {
	bytes, err := json.Marshal(j)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON - Parse JSON string back to Job
func FromJSON(data string) (*Job, error) {
	var job Job
	err := json.Unmarshal([]byte(data), &job)
	if err != nil {
		return nil, err
	}
	return &job, nil
}
