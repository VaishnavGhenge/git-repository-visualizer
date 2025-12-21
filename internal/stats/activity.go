package stats

import (
	"context"
	"fmt"
	"time"

	"git-repository-visualizer/internal/database"
)

// ActivityLevel represents the commit count for a specific date
type ActivityLevel struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
	Level int    `json:"level"` // 0-4 based on quantile
}

// GetCommitActivity returns the daily commit activity for a repository
func GetCommitActivity(ctx context.Context, pool database.PgxIface, repositoryID int64, days int) ([]ActivityLevel, error) {
	if days <= 0 {
		days = 365 // Default to 1 year
	}

	startDate := time.Now().AddDate(0, 0, -days)

	query := `
        SELECT 
            TO_CHAR(committed_at, 'YYYY-MM-DD') as date,
            COUNT(*) as count
        FROM commits
        WHERE repository_id = $1 AND committed_at >= $2
        GROUP BY date
        ORDER BY date ASC
    `

	rows, err := pool.Query(ctx, query, repositoryID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query commit activity: %w", err)
	}
	defer rows.Close()

	activityMap := make(map[string]int)
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			return nil, fmt.Errorf("failed to scan activity row: %w", err)
		}
		activityMap[date] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Fill missing dates
	var activity []ActivityLevel
	currentDate := startDate
	now := time.Now()

	for !currentDate.After(now) {
		dateStr := currentDate.Format("2006-01-02")
		count := activityMap[dateStr]

		// Calculate level (simple relative scale for now)
		level := 0
		if count > 0 {
			switch {
			case count <= 2:
				level = 1
			case count <= 5:
				level = 2
			case count <= 10:
				level = 3
			default:
				level = 4
			}
		}

		activity = append(activity, ActivityLevel{
			Date:  dateStr,
			Count: count,
			Level: level,
		})
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return activity, nil
}
