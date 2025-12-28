package classifier

import (
	"path/filepath"
	"testing"

	"clash/internal/contextinfo"
	"clash/internal/policy"
)

func TestAllowList(t *testing.T) {
	ctx := contextinfo.Info{Cwd: "/", RepoRoot: "/", InRepo: true}
	pol, _ := policy.Load("")
	res := Evaluate([]string{"ls"}, ctx, pol)
	if res.Decision != DecisionAllow {
		t.Fatalf("expected allow, got %s", res.Decision)
	}
}

func TestCatastrophicRmBlock(t *testing.T) {
	tmp := t.TempDir()
	ctx := contextinfo.Info{Cwd: tmp, RepoRoot: tmp, InRepo: true}
	pol, _ := policy.Load("")
	res := Evaluate([]string{"rm", "-rf", "/"}, ctx, pol)
	if res.Decision != DecisionBlock || !res.Hard {
		t.Fatalf("expected hard block, got %v hard=%v", res.Decision, res.Hard)
	}
}

func TestRmConfirm(t *testing.T) {
	tmp := t.TempDir()
	ctx := contextinfo.Info{Cwd: tmp, RepoRoot: tmp, InRepo: true}
	pol, _ := policy.Load("")
	res := Evaluate([]string{"rm", filepath.Join(tmp, "foo")}, ctx, pol)
	if res.Decision != DecisionConfirm {
		t.Fatalf("expected confirm, got %s", res.Decision)
	}
}
