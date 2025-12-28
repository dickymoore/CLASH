package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"clash/internal/arbiter"
	"clash/internal/audit"
	"clash/internal/classifier"
	"clash/internal/contextinfo"
	"clash/internal/policy"
	"clash/internal/preview"
	"clash/internal/ui"
)

// RunOptions controls execution behaviour.
type RunOptions struct {
	PolicyPath       string
	AutoYes          bool
	BreakGlass       bool
	BreakGlassReason string
}

// Run executes the command through CLASH.
func Run(args []string, opts RunOptions) (int, error) {
	ctx, err := contextinfo.Detect()
	if err != nil {
		return 1, err
	}

	policyPath := opts.PolicyPath
	if policyPath == "" {
		candidate := filepath.Join(ctx.RepoRoot, "clash.yaml")
		if _, err := os.Stat(candidate); err == nil {
			policyPath = candidate
		}
	}

	pol, err := policy.Load(policyPath)
	if err != nil {
		return 1, err
	}

	result := classifier.Evaluate(args, ctx, pol)
	logger, err := audit.New(ctx.RepoRoot)
	if err != nil {
		return 1, err
	}

	if pol.Arbiter.Enabled && result.Decision == classifier.DecisionConfirm {
		arb := arbiter.Decide(arbiter.Input{
			Command: strings.Join(args, " "),
			Signals: result.Signals,
			Reasons: result.Reasons,
		})
		if arb.Decision == classifier.DecisionBlock {
			result.Decision = classifier.DecisionBlock
			result.Hard = false
			result.Reasons = append(result.Reasons, "arbiter: "+arb.Reason)
		}
	}

	auditEntry := audit.Entry{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Command:   strings.Join(args, " "),
		Cwd:       ctx.Cwd,
		RepoRoot:  ctx.RepoRoot,
		Git:       ctx.Git,
		Decision:  string(result.Decision),
		Hard:      result.Hard,
		Signals:   result.Signals,
		Reasons:   result.Reasons,
		SaferAlternative: result.SaferAlternative,
	}

	var previewRes *preview.Result
	if result.PreviewHint != nil {
		pr := preview.Run(*result.PreviewHint, ctx, pol.Thresholds.PreviewSample)
		previewRes = &pr
		if auditEntry.Preview == nil {
			auditEntry.Preview = &audit.PreviewRecord{Count: pr.Count, Sample: pr.Sample, Note: pr.Note, Err: pr.Err}
		}
	}

	switch result.Decision {
	case classifier.DecisionBlock:
		if result.Hard {
			fmt.Println("CLASH: BLOCKED (hard)")
		} else {
			fmt.Println("CLASH: BLOCKED")
		}
		for _, r := range result.Reasons {
			fmt.Println("-", r)
		}
		if result.SaferAlternative != "" {
			fmt.Println("Safer:", result.SaferAlternative)
		}
		auditEntry.Outcome = "blocked"
		logger.Record(auditEntry)
		return 1, fmt.Errorf("command blocked")

	case classifier.DecisionConfirm:
		fmt.Println("CLASH: CONFIRM")
		for _, s := range result.Signals {
			fmt.Println("- signal:", s)
		}
		if previewRes != nil {
			fmt.Printf("Preview: %d items", previewRes.Count)
			if len(previewRes.Sample) > 0 {
				fmt.Printf(" sample: %s", strings.Join(previewRes.Sample, ", "))
			}
			if previewRes.Err != "" {
				fmt.Printf(" (preview error: %s)", previewRes.Err)
			}
			fmt.Println()
		}
		approved := opts.AutoYes
		approver := ""
		if approved {
			approver = "--yes"
		}
		if !approved {
			approved = ui.Confirm("Proceed with execution?")
			if approved {
				approver = "user"
			}
		}
		if !approved {
			fmt.Println("Cancelled.")
			auditEntry.Outcome = "cancelled"
			logger.Record(auditEntry)
			return 1, fmt.Errorf("cancelled")
		}
		auditEntry.ApprovedBy = approver

	case classifier.DecisionAllow:
		fmt.Println("CLASH: ALLOW (fast path)")
	}

	if opts.BreakGlass {
		phrase := "break glass for clash"
		if !ui.RequirePhrase("Break-glass override requested.", phrase) {
			fmt.Println("Break-glass phrase mismatch; aborting.")
			auditEntry.Outcome = "cancelled"
			logger.Record(auditEntry)
			return 1, fmt.Errorf("break-glass phrase mismatch")
		}
		auditEntry.BreakGlass = true
		auditEntry.BreakGlassReason = opts.BreakGlassReason
	}

	exitCode, runErr := execute(args, ctx)
	auditEntry.ExitCode = exitCode
	if runErr != nil {
		auditEntry.Outcome = "failed"
		auditEntry.Error = runErr.Error()
	} else {
		auditEntry.Outcome = "executed"
	}
	logger.Record(auditEntry)
	return exitCode, runErr
}

func execute(args []string, ctx contextinfo.Info) (int, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = ctx.Cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), err
		}
		return 1, err
	}
	return 0, nil
}
