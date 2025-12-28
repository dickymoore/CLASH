package preview

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strings"

	"clash/internal/contextinfo"
)

// HintKind identifies which previewer to use.
type HintKind string

const (
	HintRM         HintKind = "rm"
	HintFindDelete HintKind = "find-delete"
	HintGitClean   HintKind = "git-clean"
)

// Hint carries preview parameters.
type Hint struct {
	Kind    HintKind
	Args    []string
	Targets []string
}

// Result holds preview counts and samples.
type Result struct {
	Count  int
	Sample []string
	Note   string
	Err    string
}

// Run executes a preview based on the provided hint.
func Run(hint Hint, ctx contextinfo.Info, sampleLimit int) Result {
	switch hint.Kind {
	case HintRM:
		return previewRm(hint, ctx, sampleLimit)
	case HintFindDelete:
		return previewFindDelete(hint, ctx, sampleLimit)
	case HintGitClean:
		return previewGitClean(hint, ctx, sampleLimit)
	default:
		return Result{Err: "no preview available"}
	}
}

func previewRm(hint Hint, ctx contextinfo.Info, sampleLimit int) Result {
	sample := []string{}
	count := 0
	for _, t := range hint.Targets {
		resolved, err := contextinfo.ResolvePath(ctx.Cwd, t)
		if err != nil {
			continue
		}
		if _, err := os.Stat(resolved); err == nil {
			count++
			if len(sample) < sampleLimit {
				sample = append(sample, resolved)
			}
		}
	}
	return Result{Count: count, Sample: sample, Note: "targets resolved from provided arguments"}
}

func previewFindDelete(hint Hint, ctx contextinfo.Info, sampleLimit int) Result {
	args := []string{}
	for _, a := range hint.Args {
		if a == "-delete" {
			continue
		}
		args = append(args, a)
	}
	if len(args) == 0 {
		return Result{Err: "no args for find"}
	}
	cmd := exec.Command("find", args[1:]...)
	cmd.Dir = ctx.Cwd
	out, err := cmd.Output()
	if err != nil {
		return Result{Err: err.Error()}
	}
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	count := len(lines)
	sample := []string{}
	for i, line := range lines {
		if i >= sampleLimit {
			break
		}
		sample = append(sample, string(line))
	}
	return Result{Count: count, Sample: sample, Note: "find output without -delete"}
}

func previewGitClean(hint Hint, ctx contextinfo.Info, sampleLimit int) Result {
	args := []string{"clean", "-nd"}
	for _, a := range hint.Args[2:] {
		if a == "-f" || a == "--force" || a == "-n" || a == "--dry-run" {
			continue
		}
		args = append(args, a)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = ctx.RepoRoot
	out, err := cmd.Output()
	if err != nil {
		return Result{Err: err.Error()}
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	sample := []string{}
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		count++
		if len(sample) < sampleLimit {
			sample = append(sample, line)
		}
	}
	return Result{Count: count, Sample: sample, Note: "git clean -nd preview"}
}
