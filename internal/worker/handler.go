package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/queue"
)

// JobHandler implements the queue.JobHandler interface
type JobHandler struct {
	db          *database.DB
	storagePath string
}

// NewJobHandler creates a new job handler
func NewJobHandler(db *database.DB, storagePath string) *JobHandler {
	return &JobHandler{
		db:          db,
		storagePath: storagePath,
	}
}

// HandleJob processes a job from the queue
func (h *JobHandler) HandleJob(ctx context.Context, job *queue.Job) error {
	log.Printf("Processing job %s (type: %s, repo: %d)", job.ID, job.Type, job.RepositoryID)

	switch job.Type {
	case queue.JobTypeIndex:
		return h.handleIndexJob(ctx, job)
	case queue.JobTypeUpdate:
		return h.handleUpdateJob(ctx, job)
	case queue.JobTypeDelete:
		return h.handleDeleteJob(ctx, job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// handleIndexJob processes an index job
func (h *JobHandler) handleIndexJob(ctx context.Context, job *queue.Job) error {
	repoID := job.RepositoryID

	// Update status to indexing
	if err := h.db.UpdateRepositoryStatus(ctx, repoID, database.StatusIndexing); err != nil {
		return fmt.Errorf("failed to update repository status: %w", err)
	}

	// Get repository details
	repo, err := h.db.GetRepository(ctx, repoID)
	if err != nil {
		h.db.UpdateRepositoryStatus(ctx, repoID, database.StatusFailed)
		return fmt.Errorf("failed to get repository: %w", err)
	}

	log.Printf("Indexing repository: %s (ID: %d)", repo.URL, repoID)

	// TODO: Implement actual git cloning and analysis logic
	// For now, simulate processing with a delay
	time.Sleep(2 * time.Second)

	// Simulate successful indexing
	now := time.Now()
	repo.LastIndexedAt = &now
	repo.Status = database.StatusCompleted

	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.db.UpdateRepositoryStatus(ctx, repoID, database.StatusFailed)
		return fmt.Errorf("failed to update repository: %w", err)
	}

	log.Printf("Successfully indexed repository %d", repoID)
	return nil
}

// handleUpdateJob processes an update job
func (h *JobHandler) handleUpdateJob(ctx context.Context, job *queue.Job) error {
	repoID := job.RepositoryID

	// Update status to indexing
	if err := h.db.UpdateRepositoryStatus(ctx, repoID, database.StatusIndexing); err != nil {
		return fmt.Errorf("failed to update repository status: %w", err)
	}

	// Get repository details
	repo, err := h.db.GetRepository(ctx, repoID)
	if err != nil {
		h.db.UpdateRepositoryStatus(ctx, repoID, database.StatusFailed)
		return fmt.Errorf("failed to get repository: %w", err)
	}

	log.Printf("Updating repository: %s (ID: %d)", repo.URL, repoID)

	// TODO: Implement actual git pull and re-analysis logic
	// For now, simulate processing with a delay
	time.Sleep(2 * time.Second)

	// Simulate successful update
	now := time.Now()
	repo.LastIndexedAt = &now
	repo.Status = database.StatusCompleted

	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.db.UpdateRepositoryStatus(ctx, repoID, database.StatusFailed)
		return fmt.Errorf("failed to update repository: %w", err)
	}

	log.Printf("Successfully updated repository %d", repoID)
	return nil
}

// handleDeleteJob processes a delete job
func (h *JobHandler) handleDeleteJob(ctx context.Context, job *queue.Job) error {
	repoID := job.RepositoryID
	log.Printf("Deleting repository data for ID: %d", repoID)

	// TODO: Implement repository cleanup logic
	// - Remove local git clone
	// - Clean up database records

	log.Printf("Successfully deleted repository %d", repoID)
	return nil
}
