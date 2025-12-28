package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"clash/internal/audit"
	"clash/internal/contextinfo"
	"clash/internal/policy"
	"clash/internal/runner"
)

var (
	flagPolicyPath       string
	flagYes              bool
	flagBreakGlass       bool
	flagBreakGlassReason string
)

func main() {
	root := newRootCmd()
	root.SilenceUsage = true
	root.SilenceErrors = false
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clash",
		Short: "Command Line Agent Safety Harness",
		Long:  "CLASH provides a policy-aware chokepoint for command execution across agent CLIs.",
	}

	cmd.PersistentFlags().StringVar(&flagPolicyPath, "policy", "", "path to clash.yaml (defaults to repo root if present)")
	cmd.PersistentFlags().BoolVar(&flagYes, "yes", false, "auto-approve confirmation prompts")
	cmd.PersistentFlags().BoolVar(&flagBreakGlass, "break-glass", false, "enable controlled override flow")
	cmd.PersistentFlags().StringVar(&flagBreakGlassReason, "break-glass-reason", "", "reason to record when using break-glass")

	cmd.AddCommand(runCmd())
	cmd.AddCommand(initCmd())
	cmd.AddCommand(policyExplainCmd())
	cmd.AddCommand(decisionExplainCmd())
	cmd.AddCommand(doctorCmd())
	cmd.AddCommand(wrapperCmd("codex"))
	cmd.AddCommand(wrapperCmd("gemini"))
	cmd.AddCommand(wrapperCmd("claude"))
	cmd.AddCommand(wrapperCmd("copilot"))

	return cmd
}

func runCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "run -- <command>",
		Short: "Execute a command through CLASH",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("provide a command to run")
			}
			exitCode, err := runner.Run(args, runner.RunOptions{
				PolicyPath:       flagPolicyPath,
				AutoYes:          flagYes,
				BreakGlass:       flagBreakGlass,
				BreakGlassReason: flagBreakGlassReason,
			})
			if err != nil {
				os.Exit(exitCode)
			}
			os.Exit(exitCode)
			return nil
		},
	}
	return c
}

func wrapperCmd(name string) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("%s -- [args]", name),
		Short: fmt.Sprintf("Wrap %s CLI via CLASH", strings.Title(name)),
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			wrapped := append([]string{name}, args...)
			exitCode, err := runner.Run(wrapped, runner.RunOptions{
				PolicyPath:       flagPolicyPath,
				AutoYes:          flagYes,
				BreakGlass:       flagBreakGlass,
				BreakGlassReason: flagBreakGlassReason,
			})
			if err != nil {
				os.Exit(exitCode)
			}
			os.Exit(exitCode)
			return nil
		},
	}
}

func policyExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "policy explain",
		Short: "Print the effective policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, _ := contextinfo.Detect()
			path := flagPolicyPath
			if path == "" {
				candidate := filepath.Join(ctx.RepoRoot, "clash.yaml")
				if _, err := os.Stat(candidate); err == nil {
					path = candidate
				}
			}
			pol, err := policy.Load(path)
			if err != nil {
				return err
			}
			yamlStr, err := pol.ToYAML()
			if err != nil {
				return err
			}
			fmt.Print(yamlStr)
			return nil
		},
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a default clash.yaml in the repo root",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := contextinfo.Detect()
			if err != nil {
				return err
			}
			target := filepath.Join(ctx.RepoRoot, "clash.yaml")
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("clash.yaml already exists at %s", target)
			}
			if err := os.WriteFile(target, []byte(policy.DefaultYAML()), 0o644); err != nil {
				return err
			}
			fmt.Println("Created", target)
			return nil
		},
	}
}

func decisionExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "decision explain <audit-id>",
		Short: "Explain a prior decision",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, _ := contextinfo.Detect()
			logger, err := audit.New(ctx.RepoRoot)
			if err != nil {
				return err
			}
			e, err := logger.Find(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Decision: %s (hard=%t)\n", e.Decision, e.Hard)
			fmt.Printf("Command: %s\n", e.Command)
			if len(e.Signals) > 0 {
				fmt.Printf("Signals: %s\n", strings.Join(e.Signals, ", "))
			}
			if len(e.Reasons) > 0 {
				fmt.Printf("Reasons: %s\n", strings.Join(e.Reasons, ", "))
			}
			if e.Preview != nil {
				fmt.Printf("Preview: %d items", e.Preview.Count)
				if len(e.Preview.Sample) > 0 {
					fmt.Printf(" sample: %s", strings.Join(e.Preview.Sample, ", "))
				}
				if e.Preview.Err != "" {
					fmt.Printf(" (error: %s)", e.Preview.Err)
				}
				fmt.Println()
			}
			fmt.Printf("Outcome: %s exit=%d\n", e.Outcome, e.ExitCode)
			if e.ApprovedBy != "" {
				fmt.Printf("Approved by: %s\n", e.ApprovedBy)
			}
			if e.BreakGlass {
				fmt.Printf("Break-glass reason: %s\n", e.BreakGlassReason)
			}
			return nil
		},
	}
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate environment readiness",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := contextinfo.Detect()
			if err != nil {
				return err
			}
			fmt.Println("CLASH doctor")
			fmt.Println("cwd:", ctx.Cwd)
			if ctx.InRepo {
				fmt.Println("repo root:", ctx.RepoRoot)
				fmt.Printf("git status: %d changed, %d untracked\n", ctx.Git.Changed, ctx.Git.Untracked)
			} else {
				fmt.Println("repo root: none (using cwd)")
			}
			policyPath := flagPolicyPath
			if policyPath == "" {
				candidate := filepath.Join(ctx.RepoRoot, "clash.yaml")
				if _, err := os.Stat(candidate); err == nil {
					policyPath = candidate
				}
			}
			if policyPath == "" {
				fmt.Println("policy: using embedded default")
			} else {
				fmt.Println("policy:", policyPath)
			}
			return nil
		},
	}
}
