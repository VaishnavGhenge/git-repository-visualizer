package database

import (
	"context"
	"fmt"
	"sort"
)

// GetBusFactor calculates the bus factor for a repository
// Bus factor = minimum contributors who own threshold% of files
func (db *DB) GetBusFactor(ctx context.Context, repositoryID int64, threshold float64) (*BusFactorResult, error) {
	// Query: For each file, find the contributor with most additions (the "owner")
	query := `
		WITH file_contributions AS (
			SELECT 
				cf.file_path,
				c.author_email,
				c.author_name,
				SUM(cf.additions) as total_additions
			FROM commit_files cf
			JOIN commits c ON c.hash = cf.commit_hash AND c.repository_id = cf.repository_id
			WHERE cf.repository_id = $1
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
	`

	rows, err := db.pool.Query(ctx, query, repositoryID)
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
			Threshold:       threshold,
			TotalFiles:      0,
			TopContributors: []ContributorOwnership{},
			RiskLevel:       "unknown",
		}, nil
	}

	// Calculate ownership percentages
	for i := range contributors {
		contributors[i].OwnershipPct = float64(contributors[i].FilesOwned) * 100.0 / float64(totalFiles)
	}

	// Sort by files owned descending (should already be sorted, but ensure)
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].FilesOwned > contributors[j].FilesOwned
	})

	// Calculate bus factor: count contributors needed to reach threshold
	busFactor := 0
	cumulativeOwnership := 0.0
	thresholdPct := threshold * 100.0

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
		Threshold:       threshold,
		TotalFiles:      totalFiles,
		TopContributors: contributors,
		RiskLevel:       riskLevel,
	}, nil
}
