package scanner

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func inspectRepo(ctx context.Context, repoPath string) (RepoStatus, error) {
	name := filepath.Base(repoPath)
	status := RepoStatus{Path: repoPath, Name: name}

	branch, err := gitCurrentBranch(ctx, repoPath)
	if err != nil {
		return status, err
	}
	status.CurrentBranch = branch

	staged, unstaged, untracked, err := gitDirtyCount(ctx, repoPath)
	if err != nil {
		return status, err
	}
	status.StagedCount = staged
	status.UnstagedCount = unstaged
	status.UntrackedCount = untracked
	status.Dirty = staged+unstaged+untracked > 0

	unpushed, err := gitUnpushedCount(ctx, repoPath, branch)
	if err == nil {
		status.UnpushedCount = unpushed
	}

	lastCommitAge, err := gitLastCommitAge(ctx, repoPath)
	if err == nil && lastCommitAge > 0 {
		status.Staledays = int(lastCommitAge.Hours() / 24)
	}

	return status, nil
}

func gitCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	out, err := gitCmd(ctx, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func gitDirtyCount(ctx context.Context, repoPath string) (staged, unstaged, untracked int, err error) {
	out, err := gitCmd(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		return 0, 0, 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		if x == '?' && y == '?' {
			untracked++
		} else {
			if x != ' ' && x != '?' {
				staged++
			}
			if y != ' ' && y != '?' {
				unstaged++
			}
		}
	}
	return staged, unstaged, untracked, nil
}

func gitUnpushedCount(ctx context.Context, repoPath, branch string) (int, error) {
	out, err := gitCmd(ctx, repoPath, "rev-list", "--count", "origin/"+branch+"..HEAD")
	if err != nil {
		return 0, err // no remote tracking branch — not an error worth surfacing
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, err
	}
	return n, nil
}

func gitLastCommitAge(ctx context.Context, repoPath string) (time.Duration, error) {
	out, err := gitCmd(ctx, repoPath, "log", "-1", "--format=%ct")
	if err != nil {
		return 0, err
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Since(time.Unix(ts, 0)), nil
}

func gitCmd(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
