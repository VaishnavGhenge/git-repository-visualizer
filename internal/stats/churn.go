package stats

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultChurnLimit = 10

	// Weights for churn score calculation
	CommitFrequencyWeight = 0.7
	ChangeVolumeWeight    = 0.3

	// Thresholds for categorization (percentage of max)
	HotspotFrequencyThreshold = 0.6
	HotspotVolumeThreshold    = 0.6

	// Categories
	CategoryHotspot  = "hotspot"
	CategoryFrequent = "frequent"
	CategoryMassive  = "massive"
	CategoryStable   = "stable"
)

// ChurnOptions contains optional filters for churn calculation
type ChurnOptions struct {
	Limit int // Top N files
	Days  int // Only count commits in last N days (0 = all time)
}

// FileChurn represents churn statistics for a single file
type FileChurn struct {
	FilePath     string    `json:"file_path"`
	CommitCount  int       `json:"commit_count"`
	LinesChanged int       `json:"lines_changed"`
	ChurnScore   float64   `json:"churn_score"`
	Category     string    `json:"category"` // "hotspot", "frequent", "massive", or "stable"
	LastModified time.Time `json:"last_modified"`
}

// GetHighChurnFiles calculates the churn for files in a repository
func GetHighChurnFiles(ctx context.Context, pool *pgxpool.Pool, repositoryID int64, opts ChurnOptions) ([]FileChurn, error) {
	// if opts.Limit <= 0 {
	// 	opts.Limit = DefaultChurnLimit
	// }

	var timeFilter string
	var args []interface{}
	args = append(args, repositoryID)

	if opts.Days > 0 {
		cutoffDate := time.Now().AddDate(0, 0, -opts.Days)
		timeFilter = "AND c.committed_at > $2"
		args = append(args, cutoffDate)
	}

	query := fmt.Sprintf(`
		SELECT
			cf.file_path,
			COUNT(DISTINCT cf.commit_hash) as commit_count,
			SUM(cf.additions + cf.deletions) as lines_changed,
			MAX(c.committed_at) as last_modified
		FROM commit_files cf
		JOIN commits c ON cf.commit_hash = c.hash AND cf.repository_id = c.repository_id
		WHERE cf.repository_id = $1 %s
		GROUP BY cf.file_path
		ORDER BY commit_count DESC, lines_changed DESC
		LIMIT $%d
	`, timeFilter, len(args)+1)
	args = append(args, opts.Limit)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query file churn: %w", err)
	}
	defer rows.Close()

	var results []FileChurn
	var maxCommits, maxLines int

	for rows.Next() {
		var fc FileChurn
		if err := rows.Scan(&fc.FilePath, &fc.CommitCount, &fc.LinesChanged, &fc.LastModified); err != nil {
			return nil, fmt.Errorf("failed to scan churn row: %w", err)
		}

		if fc.CommitCount > maxCommits {
			maxCommits = fc.CommitCount
		}
		if fc.LinesChanged > maxLines {
			maxLines = fc.LinesChanged
		}

		results = append(results, fc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Calculate scores and categories
	for i := range results {
		results[i].ChurnScore = calculateScore(results[i].CommitCount, results[i].LinesChanged, maxCommits, maxLines)
		results[i].Category = categorizeFile(results[i].CommitCount, results[i].LinesChanged, maxCommits, maxLines)
	}

	return results, nil
}

// calculateScore returns a weighted score between 0 and 100
func calculateScore(commits, lines, maxCommits, maxLines int) float64 {
	if maxCommits == 0 || maxLines == 0 {
		return 0
	}

	// Normalize metrics
	normCommits := float64(commits) / float64(maxCommits)
	normLines := math.Log1p(float64(lines)) / math.Log1p(float64(maxLines)) // Use log for lines to dampen huge outliers

	// Weight calculation using constants
	score := (normCommits * CommitFrequencyWeight) + (normLines * ChangeVolumeWeight)
	return math.Round(score*1000) / 10
}

func categorizeFile(commits, lines, maxCommits, maxLines int) string {
	if maxCommits == 0 {
		return CategoryStable
	}

	// Thresholds for categorization using constants
	freqThreshold := float64(maxCommits) * HotspotFrequencyThreshold
	volumeThreshold := float64(maxLines) * HotspotVolumeThreshold

	isHighFreq := float64(commits) >= freqThreshold
	isHighVolume := float64(lines) >= volumeThreshold

	if isHighFreq && isHighVolume {
		return CategoryHotspot
	}
	if isHighFreq {
		return CategoryFrequent
	}
	if isHighVolume {
		return CategoryMassive
	}
	return CategoryStable
}
