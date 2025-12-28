package classifier

import (
	"os"
	"path/filepath"
	"strings"

	"clash/internal/contextinfo"
	"clash/internal/policy"
	"clash/internal/preview"
)

// DecisionType enumerates ladder outcomes.
type DecisionType string

const (
	DecisionAllow   DecisionType = "ALLOW"
	DecisionConfirm DecisionType = "CONFIRM"
	DecisionBlock   DecisionType = "BLOCK"
)

// Result captures classification output.
type Result struct {
	Decision         DecisionType
	Hard             bool
	Reasons          []string
	Signals          []string
	PreviewHint      *preview.Hint
	SaferAlternative string
}

// Evaluate applies the policy ladder to the requested command.
func Evaluate(args []string, ctx contextinfo.Info, p policy.Policy) Result {
	if len(args) == 0 {
		return Result{Decision: DecisionBlock, Hard: true, Reasons: []string{"no command provided"}}
	}

	cmd := args[0]
	lowerCmd := strings.ToLower(cmd)
	joined := strings.Join(args, " ")
	targets := extractTargets(args)

	// 1) Deterministic hard blocks
	if inListPrefix(lowerCmd, p.BlockCommands) {
		return Result{Decision: DecisionBlock, Hard: true, Reasons: []string{"command is in hard block list"}}
	}

	if isCatastrophicRm(lowerCmd, args, ctx) {
		return Result{Decision: DecisionBlock, Hard: true, Reasons: []string{"catastrophic rm target"}, SaferAlternative: "narrow path or remove -rf"}
	}

	if isUnsafeGitReset(args, ctx) {
		return Result{Decision: DecisionBlock, Hard: true, Reasons: []string{"git reset --hard with dirty tree"}, SaferAlternative: "commit or stash first"}
	}

	if isUnsafeGitClean(args) {
		return Result{Decision: DecisionBlock, Hard: true, Reasons: []string{"git clean -fdx without dry-run"}, SaferAlternative: "git clean -ndx"}
	}

	// 2) Deterministic allow list
	if inAllowList(args, p.AllowCommands) {
		return Result{Decision: DecisionAllow, Reasons: []string{"allowlisted read-only command"}}
	}

	// 3) Risk signals leading to confirm
	riskSignals := []string{}
	previewHint := (*preview.Hint)(nil)

	if isMutatingCommand(lowerCmd) {
		riskSignals = append(riskSignals, "mutating command")
	}

	if touchesProtected(targets, ctx, p.ProtectedPaths) {
		riskSignals = append(riskSignals, "touches protected path")
	}

	if hasForceFlag(args) {
		riskSignals = append(riskSignals, "force flag present")
	}

	if isOutsideRepo(targets, ctx) && !p.Options.AllowOutsideRepo {
		riskSignals = append(riskSignals, "outside repo root")
	}

	if isNetworkEgress(lowerCmd, p.NetworkEgress) {
		riskSignals = append(riskSignals, "network egress command")
	}

	if isPackageManager(lowerCmd, p.PackageManagers) {
		riskSignals = append(riskSignals, "package manager install/upgrade")
	}

	if isFindDelete(args) {
		riskSignals = append(riskSignals, "find -delete")
		previewHint = &preview.Hint{Kind: preview.HintFindDelete, Args: args}
	}

	if lowerCmd == "rm" {
		previewHint = &preview.Hint{Kind: preview.HintRM, Args: args, Targets: targets}
	}

	if isGitClean(args) {
		previewHint = &preview.Hint{Kind: preview.HintGitClean, Args: args}
	}

	if len(riskSignals) == 0 {
		return Result{Decision: DecisionAllow, Reasons: []string{"no risk signals"}}
	}

	return Result{
		Decision:    DecisionConfirm,
		Reasons:     []string{"risk signals present"},
		Signals:     riskSignals,
		PreviewHint: previewHint,
		SaferAlternative: suggestAlternative(lowerCmd, previewHint),
	}
}

func inAllowList(args []string, allow []string) bool {
	joined := strings.ToLower(strings.Join(args, " "))
	for _, a := range allow {
		if joined == strings.ToLower(a) || strings.HasPrefix(joined, strings.ToLower(a)+" ") {
			return true
		}
	}
	return false
}

func inListPrefix(cmd string, list []string) bool {
	for _, x := range list {
		if strings.HasPrefix(cmd, strings.ToLower(x)) {
			return true
		}
	}
	return false
}

func extractTargets(args []string) []string {
	targets := []string{}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			targets = append(targets, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		targets = append(targets, arg)
	}
	return targets
}

func isMutatingCommand(cmd string) bool {
	switch cmd {
	case "rm", "rmdir", "mv", "chmod", "chown", "truncate":
		return true
	case "git":
		return true
	}
	return false
}

func touchesProtected(targets []string, ctx contextinfo.Info, protected []string) bool {
	for _, t := range targets {
		resolved, err := contextinfo.ResolvePath(ctx.Cwd, t)
		if err != nil {
			continue
		}
		for _, p := range protected {
			if p == "" {
				continue
			}
			candidate := p
			if strings.HasPrefix(p, "~") {
				h, _ := os.UserHomeDir()
				candidate = filepath.Join(h, strings.TrimPrefix(p, "~"))
			}
			if strings.HasPrefix(p, "$") {
				env := os.Getenv(strings.TrimPrefix(p, "$"))
				if env != "" {
					candidate = env
				}
			}
			if strings.HasPrefix(resolved, candidate) {
				return true
			}
		}
	}
	return false
}

func isOutsideRepo(targets []string, ctx contextinfo.Info) bool {
	if !ctx.InRepo {
		return len(targets) > 0
	}
	for _, t := range targets {
		resolved, err := contextinfo.ResolvePath(ctx.Cwd, t)
		if err != nil {
			continue
		}
		if !contextinfo.IsInsideRepo(ctx.RepoRoot, resolved) {
			return true
		}
	}
	return false
}

func hasForceFlag(args []string) bool {
	for _, a := range args {
		if a == "-f" || a == "--force" || a == "--hard" || a == "-rf" || a == "-fr" {
			return true
		}
	}
	return false
}

func isNetworkEgress(cmd string, list []string) bool {
	for _, c := range list {
		if cmd == c {
			return true
		}
	}
	return false
}

func isPackageManager(cmd string, list []string) bool {
	for _, c := range list {
		if cmd == c {
			return true
		}
	}
	return false
}

func isFindDelete(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if strings.ToLower(args[0]) != "find" {
		return false
	}
	for _, a := range args[1:] {
		if a == "-delete" {
			return true
		}
	}
	return false
}

func isGitClean(args []string) bool {
	return len(args) >= 2 && args[0] == "git" && args[1] == "clean"
}

func isUnsafeGitClean(args []string) bool {
	if !isGitClean(args) {
		return false
	}
	hasDryRun := false
	hasTarget := false
	for _, a := range args[2:] {
		if a == "-n" || a == "--dry-run" {
			hasDryRun = true
		}
		if !strings.HasPrefix(a, "-") {
			hasTarget = true
		}
	}
	flags := strings.Join(args, " ")
	if strings.Contains(flags, "-fdx") && !hasDryRun && !hasTarget {
		return true
	}
	return false
}

func isUnsafeGitReset(args []string, ctx contextinfo.Info) bool {
	if len(args) >= 3 && args[0] == "git" && args[1] == "reset" && args[2] == "--hard" {
		if ctx.Git.Changed > 0 || ctx.Git.Untracked > 0 {
			return true
		}
	}
	return false
}

func isCatastrophicRm(cmd string, args []string, ctx contextinfo.Info) bool {
	if cmd != "rm" {
		return false
	}
	if !(hasFlag(args, "-r") || hasFlag(args, "-rf") || hasFlag(args, "-fr")) {
		return false
	}
	targets := extractTargets(args)
	for _, t := range targets {
		resolved, err := contextinfo.ResolvePath(ctx.Cwd, t)
		if err != nil {
			continue
		}
		if resolved == "/" {
			return true
		}
		home, _ := os.UserHomeDir()
		if resolved == home {
			return true
		}
		if ctx.InRepo && !contextinfo.IsInsideRepo(ctx.RepoRoot, resolved) {
			return true
		}
	}
	return false
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func suggestAlternative(cmd string, hint *preview.Hint) string {
	switch cmd {
	case "rm":
		return "add --dry-run or target fewer files"
	case "git":
		return "run with --dry-run or limit path"
	case "find":
		return "run find without -delete first"
	case "npm", "pnpm", "yarn", "pip", "pip3":
		return "pin versions and review diff before install"
	}
	if hint != nil && hint.Kind == preview.HintGitClean {
		return "use git clean -ndx first"
	}
	return ""
}
