package git

import (
	"fmt"
	"sort"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Repository struct {
	repo *git.Repository
	ref  *plumbing.Reference
}

type Contributor struct {
	Name    string
	Email   string
	Commits int
	Changes int
}

func OpenRepository(repoPath string) (*Repository, error) {
	// Open the repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get the reference to HEAD
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	return &Repository{repo: repo, ref: ref}, nil
}

func GetContributors(repoPath string) ([]*Contributor, error) {
	repo, err := OpenRepository(repoPath)
	if err != nil {
		return nil, err
	}

	// Get the commit history
	commits, err := repo.repo.Log(&git.LogOptions{From: repo.ref.Hash()})
	if err != nil {
		return nil, err
	}

	// Map to store unique users
	contributors := make(map[string]*Contributor)
	var mu sync.Mutex

	// WaitGroup for limiting parallel worker
	var wg sync.WaitGroup

	// Commit hash channel to create commit hash queue
	commitHash := make(chan *plumbing.Hash)

	// Assign workers for processing commits
	numWorkers := 10
	for range numWorkers {
		wg.Add(1)

		go func ()  {
			defer wg.Done()
			
			// Create local repo object to avoid race condition (go-git's internal state is not thread-safe)
			repo, err := git.PlainOpen(repoPath)
			if err != nil {
				return
			}

			// Process commits
			for hash := range commitHash {
				commit, err := repo.CommitObject(*hash)
				if err != nil {
					return
				}

				var commitChanges int
				commitStats, err := commit.Stats()
				if err != nil {
					return
				}

				email := commit.Author.Email
				for _, stat := range commitStats {
					commitChanges += stat.Addition + stat.Deletion
				}

				// Update contributors map in thread-safe manner
				mu.Lock()
				contributor, ok := contributors[email]
				if !ok {
					contributors[email] = &Contributor{
						Email: email,
						Name: commit.Author.Name,
						Commits: 1,
						Changes: commitChanges,
					}
				} else {
					contributor.Commits += 1
					contributor.Changes += commitChanges
				}
				mu.Unlock()
			}
		}()
	}

	// Push commits to commit hash queue (channel)
	commits.ForEach(func(commit *object.Commit) error {
		commitHash <- &commit.Hash
		return nil
	})

	// Close commit hash channel
	close(commitHash)

	// Wait for all workers to complete
	wg.Wait()

	// Prepare list of contributors
	var contributorList []*Contributor
	for _, c := range contributors {
		contributorList = append(contributorList, c)
	}

	// Sort contributors by number of commits
	sort.Slice(contributorList, func(i, j int) bool {
		return contributorList[i].Changes > contributorList[j].Changes
	})
	return contributorList, nil
}
