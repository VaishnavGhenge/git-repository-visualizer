package stats

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"git-repository-visualizer/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BusFactorOptions contains optional filters for bus factor calculation
type BusFactorOptions struct {
	Threshold       float64 // Ownership threshold (e.g., 0.5 = 50%)
	ActiveDays      int     // Only count contributors active in last N days (0 = all time)
	ExcludePatterns bool    // Whether to exclude files matching exclusion patterns
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

// CalculateBusFactor calculates the bus factor for a repository
// Bus factor = minimum contributors who own threshold% of files
func CalculateBusFactor(ctx context.Context, pool *pgxpool.Pool, repositoryID int64, opts BusFactorOptions) (*BusFactorResult, error) {
	// Build dynamic WHERE clauses
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, fmt.Sprintf("cf.repository_id = $%d", argIndex))
	args = append(args, repositoryID)
	argIndex++

	// Active contributors filter
	var activeContributorFilter string
	if opts.ActiveDays > 0 {
		cutoffDate := time.Now().AddDate(0, 0, -opts.ActiveDays)
		activeContributorFilter = fmt.Sprintf(`
			AND c.author_email IN (
				SELECT DISTINCT author_email FROM commits 
				WHERE repository_id = $1 AND committed_at > $%d
			)`, argIndex)
		args = append(args, cutoffDate)
		argIndex++
	}

	// File exclusion filter
	var exclusionFilter string
	if opts.ExcludePatterns {
		patterns := config.GetExclusionPatterns()
		if len(patterns) > 0 {
			var notLikes []string
			for _, pattern := range patterns {
				notLikes = append(notLikes, fmt.Sprintf("cf.file_path NOT LIKE $%d", argIndex))
				args = append(args, pattern)
				argIndex++
			}
			exclusionFilter = " AND " + strings.Join(notLikes, " AND ")
		}
	}

	query := fmt.Sprintf(`
		WITH file_contributions AS (
			SELECT 
				cf.file_path,
				c.author_email,
				c.author_name,
				SUM(cf.additions) as total_additions
			FROM commit_files cf
			JOIN commits c ON c.hash = cf.commit_hash AND c.repository_id = cf.repository_id
			WHERE %s%s%s
			GROUP BY cf.file_path, c.author_email, c.author_name
		),
		file_owners AS (
			SELECT DISTINCT ON (file_path) 
			file_path,
			author_email,
			author_name
			FROM file_contributions
			WHERE total_additions > 0
			ORDER BY file_path, total_additions DESC
		)
		SELECT 
			author_email,
			author_name,
			COUNT(*) as files_owned
		FROM file_owners
		GROUP BY author_email, author_name
		ORDER BY files_owned DESC
	`, strings.Join(conditions, " AND "), activeContributorFilter, exclusionFilter)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query file ownership: %w", err)
	}
	defer rows.Close()

	var contributors []ContributorOwnership
	totalFiles := 0

	for rows.Next() {
		var co ContributorOwnership
		if err := rows.Scan(&co.Email, &co.Name, &co.FilesOwned); err != nil {
			return nil, fmt.Errorf("failed to scan ownership: %w", err)
		}
		totalFiles += co.FilesOwned
		contributors = append(contributors, co)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if totalFiles == 0 {
		return &BusFactorResult{
			BusFactor:       0,
			Threshold:       opts.Threshold,
			TotalFiles:      0,
			TopContributors: []ContributorOwnership{},
			RiskLevel:       "unknown",
		}, nil
	}

	// Calculate ownership percentages
	for i := range contributors {
		contributors[i].OwnershipPct = float64(contributors[i].FilesOwned) * 100.0 / float64(totalFiles)
	}

	// Sort by files owned descending
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].FilesOwned > contributors[j].FilesOwned
	})

	// Calculate bus factor: count contributors needed to reach threshold
	busFactor := 0
	cumulativeOwnership := 0.0
	thresholdPct := opts.Threshold * 100.0

	for _, c := range contributors {
		busFactor++
		cumulativeOwnership += c.OwnershipPct
		if cumulativeOwnership >= thresholdPct {
			break
		}
	}

	// Determine risk level
	riskLevel := "low"
	if busFactor == 1 {
		riskLevel = "high"
	} else if busFactor <= 3 {
		riskLevel = "medium"
	}

	return &BusFactorResult{
		BusFactor:       busFactor,
		Threshold:       opts.Threshold,
		TotalFiles:      totalFiles,
		TopContributors: contributors,
		RiskLevel:       riskLevel,
	}, nil
}
