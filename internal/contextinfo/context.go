package contextinfo

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Info holds execution context for policy evaluation and logging.
type Info struct {
	Cwd       string
	RepoRoot  string
	InRepo    bool
	Git       GitSummary
}

// GitSummary captures a minimal git status snapshot.
type GitSummary struct {
	Changed  int
	Untracked int
}

// Detect collects cwd, repo root, and git status summary.
func Detect() (Info, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Info{}, err
	}

	repo, inRepo := findRepoRoot(cwd)
	gitSum := GitSummary{}
	if inRepo {
		gitSum = gitStatus(repo)
	}

	return Info{
		Cwd:      cwd,
		RepoRoot: repo,
		InRepo:   inRepo,
		Git:      gitSum,
	}, nil
}

func findRepoRoot(start string) (string, bool) {
	cur := start
	for {
		if cur == "/" {
			return start, false
		}
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur, true
		}
		next := filepath.Dir(cur)
		if next == cur {
			return start, false
		}
		cur = next
	}
}

func gitStatus(repo string) GitSummary {
	cmd := exec.Command("git", "-C", repo, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return GitSummary{}
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	changed := 0
	untracked := 0
	for _, l := range lines {
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "??") {
			untracked++
		} else {
			changed++
		}
	}
	return GitSummary{Changed: changed, Untracked: untracked}
}

// IsInsideRepo returns true if path is within repo root.
func IsInsideRepo(repoRoot, path string) bool {
	if repoRoot == "" {
		return false
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRepo, absPath)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."))
}

// ResolvePath resolves candidate relative to base, handling ~ expansion.
func ResolvePath(base, candidate string) (string, error) {
	if candidate == "" {
		return "", errors.New("empty path")
	}
	if strings.HasPrefix(candidate, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		candidate = filepath.Join(home, strings.TrimPrefix(candidate, "~"))
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(base, candidate)
	}
	return filepath.EvalSymlinks(candidate)
}
