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

// Contributor represents a developer identity found in the git log
type Contributor struct {
	ID            int64      `json:"id"`
	RepositoryID  int64      `json:"repository_id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	FirstCommitAt *time.Time `json:"first_commit_at,omitempty"`
	LastCommitAt  *time.Time `json:"last_commit_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	// Note: Aggregate stats (CommitCount, LinesAdded) are removed to force
	// granular calculation from the Commit/CommitFile tables.
}

// File represents a file as it exists in the HEAD of the repository (Index)
// This is used for "State" insights: "How big is this system?" "What languages?"
type File struct {
	ID           int64     `json:"id"`
	RepositoryID int64     `json:"repository_id"`
	Path         string    `json:"path"`
	Language     string    `json:"language"` // e.g. "Go", "TypeScript" - inferred from extension
	Lines        int       `json:"lines"`    // Lines of Code at HEAD
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Commit represents a single point in the repository timeline
type Commit struct {
	ID           int64     `json:"id"`
	RepositoryID int64     `json:"repository_id"`
	Hash         string    `json:"hash"`
	AuthorEmail  string    `json:"author_email"` // Denormalized for easier querying
	AuthorName   string    `json:"author_name"`
	Message      string    `json:"message"`
	CommittedAt  time.Time `json:"committed_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// CommitFile records the modification of a specific file in a specific commit
// This is the atomic unit of "Data" for insights like Churn, Hotspots, and Knowledge Map.
type CommitFile struct {
	ID           int64  `json:"id"`
	CommitHash   string `json:"commit_hash"`
	RepositoryID int64  `json:"repository_id"`
	FilePath     string `json:"file_path"` // Captured at the time of commit
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
}

// BusFactorResult holds the calculated bus factor and ownership data
type BusFactorResult struct {
	BusFactor       int                    `json:"bus_factor"`
	Threshold       float64                `json:"threshold"` // e.g., 0.5 for 50%
	TotalFiles      int                    `json:"total_files"`
	TopContributors []ContributorOwnership `json:"top_contributors"`
	RiskLevel       string                 `json:"risk_level"` // "high", "medium", "low"
}

// ContributorOwnership represents a contributor's file ownership stats
type ContributorOwnership struct {
	Email        string  `json:"email"`
	Name         string  `json:"name"`
	FilesOwned   int     `json:"files_owned"`
	OwnershipPct float64 `json:"ownership_pct"`
}
