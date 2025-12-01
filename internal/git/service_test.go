package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func createTempRepo(t testing.TB, commitCount int) string {
	dir, err := os.MkdirTemp("", "git-repo-")
	if err != nil {
		t.Fatal(err)
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}

	for i := 0; i < commitCount; i++ {
		filename := filepath.Join(dir, "file")
		if err := os.WriteFile(filename, []byte(time.Now().String()), 0644); err != nil {
			os.RemoveAll(dir)
			t.Fatal(err)
		}

		if _, err := w.Add("file"); err != nil {
			os.RemoveAll(dir)
			t.Fatal(err)
		}

		if _, err := w.Commit("commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "John Doe",
				Email: "john@example.com",
				When:  time.Now(),
			},
		}); err != nil {
			os.RemoveAll(dir)
			t.Fatal(err)
		}
	}

	return dir
}

func BenchmarkGetContributors(b *testing.B) {
	// Create a repo with enough commits to make the benchmark meaningful but not too slow to setup
	repoPath := createTempRepo(b, 1000)
	defer os.RemoveAll(repoPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetContributors(repoPath)
		if err != nil {
			b.Fatalf("GetContributors failed: %v", err)
		}
	}
}
