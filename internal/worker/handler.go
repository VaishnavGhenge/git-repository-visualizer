package worker

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"git-repository-visualizer/internal/auth"
	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/git"
	"git-repository-visualizer/internal/queue"

	"golang.org/x/oauth2"
)

// JobHandler implements the queue.JobHandler interface
type JobHandler struct {
	db           *database.DB
	storagePath  string
	authRegistry *auth.Registry
}

// NewJobHandler creates a new job handler
func NewJobHandler(db *database.DB, storagePath string, registry *auth.Registry) *JobHandler {
	return &JobHandler{
		db:           db,
		storagePath:  storagePath,
		authRegistry: registry,
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
	case queue.JobTypeDiscover:
		return h.handleDiscoverJob(ctx, job)
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

	// Construct local path for cloning
	localPath := filepath.Join(h.storagePath, fmt.Sprintf("%d", repoID))

	// Index the repository
	if err := git.IndexRepository(ctx, h.db, repoID, repo.URL, localPath); err != nil {
		h.db.UpdateRepositoryStatus(ctx, repoID, database.StatusFailed)
		return fmt.Errorf("failed to index repository: %w", err)
	}

	// Update repository status to completed
	now := time.Now()
	repo.LastIndexedAt = &now
	repo.Status = database.StatusCompleted
	repo.LocalPath = &localPath

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

// handleDiscoverJob fetches repositories from a provider and stores them in the DB
func (h *JobHandler) handleDiscoverJob(ctx context.Context, job *queue.Job) error {
	userIDRaw, ok := job.Payload["user_id"]
	if !ok {
		return fmt.Errorf("missing user_id in discover job payload")
	}
	userID := int64(userIDRaw.(float64))

	providerName, ok := job.Payload["provider"]
	if !ok {
		return fmt.Errorf("missing provider in discover job payload")
	}
	providerStr := providerName.(string)

	log.Printf("Discovering repositories for user %d from provider %s", userID, providerStr)

	// 1. Get user identity
	identity, err := h.db.GetUserIdentity(ctx, userID, providerStr)
	if err != nil {
		return fmt.Errorf("failed to get user identity: %w", err)
	}

	if identity.AccessToken == nil {
		return fmt.Errorf("no access token for user %d provider %s", userID, providerStr)
	}

	// 2. Get provider
	p, err := h.authRegistry.Get(providerStr)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// 3. Fetch repositories
	token := &oauth2.Token{
		AccessToken: *identity.AccessToken,
	}
	if identity.RefreshToken != nil {
		token.RefreshToken = *identity.RefreshToken
	}
	if identity.TokenExpiry != nil {
		token.Expiry = *identity.TokenExpiry
	}

	repos, err := p.FetchRepositories(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to fetch repositories from %s: %w", providerStr, err)
	}

	// 4. Save to DB
	for _, r := range repos {
		// Check if already exists
		_, err := h.db.GetRepositoryByURL(ctx, r.URL)
		if err == nil {
			// Already exists, maybe update metadata but don't overwrite user_id if already set by someone else?
			// For now, if it exists, we just ensure it's linked if needed or skip.
			continue
		}

		name := r.FullName
		description := r.Description
		provider := providerStr

		repo := &database.Repository{
			URL:           r.URL,
			Name:          &name,
			Description:   &description,
			IsPrivate:     r.IsPrivate,
			Provider:      &provider,
			UserID:        &userID,
			Status:        database.StatusDiscovered,
			DefaultBranch: r.DefaultBranch,
		}

		if err := h.db.CreateRepository(ctx, repo); err != nil {
			log.Printf("Error creating discovered repository %s: %v", r.URL, err)
			continue
		}
	}

	log.Printf("Successfully discovered %d repositories for user %d", len(repos), userID)
	return nil
}
