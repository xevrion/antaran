package scanner

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

type RepoStatus struct {
	Path           string
	Name           string
	Dirty          bool
	StagedCount    int
	UnstagedCount  int
	UntrackedCount int
	UnpushedCount  int
	CurrentBranch  string
	Staledays      int
}

type Scanner struct {
	root     string
	maxDepth int
	timeout  time.Duration
}

func New(root string, maxDepth int) *Scanner {
	return &Scanner{
		root:     root,
		maxDepth: maxDepth,
		timeout:  5 * time.Second,
	}
}

func (s *Scanner) Scan(ctx context.Context) ([]RepoStatus, error) {
	var repos []RepoStatus
	err := walkGitRepos(s.root, s.maxDepth, func(repoPath string) {
		ctx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()

		status, err := inspectRepo(ctx, repoPath)
		if err != nil {
			return
		}
		repos = append(repos, status)
	})
	return repos, err
}

func walkGitRepos(root string, maxDepth int, fn func(string)) error {
	return walkDir(root, 0, maxDepth, fn)
}

func walkDir(dir string, depth, maxDepth int, fn func(string)) error {
	if depth > maxDepth {
		return nil
	}

	gitDir := filepath.Join(dir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		fn(dir)
		return nil // don't recurse into a git repo
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // skip unreadable dirs silently
	}

	for _, e := range entries {
		if !e.IsDir() || e.Name()[0] == '.' {
			continue
		}
		if err := walkDir(filepath.Join(dir, e.Name()), depth+1, maxDepth, fn); err != nil {
			return err
		}
	}
	return nil
}
