package database

import "time"

// RepositoryStatus represents the status of a repository indexing process
type RepositoryStatus string

const (
	StatusPending   RepositoryStatus = "pending"
	StatusIndexing  RepositoryStatus = "indexing"
	StatusCompleted RepositoryStatus = "completed"
	StatusFailed    RepositoryStatus = "failed"
)

// Repository represents a git repository being tracked
type Repository struct {
	ID            int64            `json:"id"`
	URL           string           `json:"url"`
	LocalPath     *string          `json:"local_path,omitempty"`
	DefaultBranch string           `json:"default_branch"`
	Status        RepositoryStatus `json:"status"`
	LastIndexedAt *time.Time       `json:"last_indexed_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// Contributor represents a contributor to a repository
type Contributor struct {
	ID            int64      `json:"id"`
	RepositoryID  int64      `json:"repository_id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	CommitCount   int        `json:"commit_count"`
	LinesAdded    int        `json:"lines_added"`
	LinesDeleted  int        `json:"lines_deleted"`
	FirstCommitAt *time.Time `json:"first_commit_at,omitempty"`
	LastCommitAt  *time.Time `json:"last_commit_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// FileStat represents statistics for a file in a repository
type FileStat struct {
	ID             int64      `json:"id"`
	RepositoryID   int64      `json:"repository_id"`
	FilePath       string     `json:"file_path"`
	TotalChanges   int        `json:"total_changes"`
	LinesAdded     int        `json:"lines_added"`
	LinesDeleted   int        `json:"lines_deleted"`
	LastModifiedAt *time.Time `json:"last_modified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Commit represents a single commit in a repository
type Commit struct {
	ID           int64     `json:"id"`
	RepositoryID int64     `json:"repository_id"`
	Hash         string    `json:"hash"`
	AuthorEmail  string    `json:"author_email"`
	AuthorName   string    `json:"author_name"`
	Message      string    `json:"message"`
	CommittedAt  time.Time `json:"committed_at"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	CreatedAt    time.Time `json:"created_at"`
}
