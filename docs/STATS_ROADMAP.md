# Git Repository Visualizer - Stats Roadmap

A comprehensive list of analytics and insights to help development teams identify individual skills, gaps, and repository health.

---

## 👤 Individual Developer Stats (Skills & Gaps Identification)

| Stat | Description | Insight |
|------|-------------|---------|
| **Language Proficiency** | Files touched per language (from `CommitFiles` joined with `Files.Language`) | Shows expertise areas |
| **Commit Frequency** | Commits per day/week/month over time | Activity patterns |
| **Code Churn Ratio** | `(additions + deletions) / total_lines_touched` | Writing style - high churn may indicate refactoring habits |
| **Average Commit Size** | Avg lines changed per commit | Work style indicator |
| **Late-Night/Weekend Commits** | Commits by hour/day-of-week | Work pattern visibility |
| **Consistency Score** | Std deviation of commits per week | Steady contributor vs burst contributor |
| **Multi-Area Contributions** | Distinct file paths/directories touched | Generalist vs specialist |
| **First/Last Commit Gap** | Time from first to last commit | Tenure in project |

---

## 🏆 Team/Repo-Level Stats

| Stat | Description | Insight |
|------|-------------|---------|
| **Bus Factor** | Min contributors who own X% of the codebase | Risk assessment |
| **File Hotspots** | Files with most changes (high churn) | Potential refactoring candidates |
| **Knowledge Map** | Which developer knows which files best | Team planning, code review assignments |
| **Ownership Matrix** | % of lines per file attributed to each contributor | "Who wrote this?" |
| **Active Contributors** | Contributors with commits in last 30/90 days | Team health |
| **Contribution Distribution** | Gini coefficient of commits per contributor | Is work evenly spread? |
| **Average PR Size** | (If you track PRs) Lines changed per merge | Review burden indicator |
| **Language Distribution** | Total lines by language at HEAD | Tech stack overview |
| **Dead File Detection** | Files not touched in 6+ months | Potential tech debt |

---

## 📈 Trend/Time-Series Stats

| Stat | Description |
|------|-------------|
| **Commit Velocity** | Commits per week over time |
| **Team Growth** | New contributors per month |
| **Codebase Growth** | Lines of code over time |
| **Contributor Churn** | Contributors who stopped committing |

---

## 🎯 Recommended Priority (Quick Wins)

Based on the current schema (`Commits`, `CommitFiles`, `Contributors`, `Files`), here are high-impact stats to implement first:

1. **Bus Factor** - High-impact, unique insight for risk assessment
2. **File Hotspots** - Directly queryable from `CommitFiles`
3. **Developer Language Proficiency** - Join `CommitFiles` with `Files`
4. **Contribution Distribution** - Simple aggregation on `Commits`
5. **Knowledge Map** - Per-file ownership percentages

---

## Schema Reference

The current database schema supports these analytics:

- **`Commits`** - Hash, author, message, timestamp
- **`CommitFiles`** - Per-file additions/deletions per commit (atomic unit)
- **`Contributors`** - Developer identity with first/last commit dates
- **`Files`** - Current HEAD inventory with language and line counts
