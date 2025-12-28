# CLASH — Command Line Agent Safety Harness
[![CI](https://github.com/dickymoore/CLASH/actions/workflows/ci.yml/badge.svg)](https://github.com/dickymoore/CLASH/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-blue)](https://go.dev/)

CLASH is a universal chokepoint for agent-initiated command execution. It enforces an argument-aware policy ladder (BLOCK → ALLOW → CONFIRM → optional arbiter → break-glass) so LLM CLIs can act autonomously without sacrificing safety.

## Why not just an allowlist?
Simple binary allowlists miss risk hiding in arguments (`rm -rf ~`, `python -c 'shred'`, `find . -delete`). CLASH inspects command targets, force flags, repo boundaries, and provides previews before destructive steps.

## Quickstart
```bash
# build (requires Go 1.21)
go mod tidy
go build -o clash ./cmd/clash

# initialize policy in your repo
./clash init

# run commands through CLASH
./clash run -- ls
./clash run -- rm tmp.txt              # will confirm with preview
./clash run -- rm -rf /                 # blocked
```

## Example decisions
- **ALLOW**: `clash run -- git status`
- **CONFIRM**: `clash run -- rm foo` → shows count/sample, asks to proceed
- **BLOCK (hard)**: `clash run -- rm -rf /` → refused with safer suggestion

## Commands
- `clash run -- <cmd>`: core chokepoint
- `clash codex|gemini|claude|copilot -- [args]`: wrap those CLIs (best-effort logging/protection)
- `clash init`: write default `clash.yaml`
- `clash policy explain`: print effective policy
- `clash decision explain <audit-id>`: inspect a prior decision
- `clash doctor`: sanity checks (repo root, policy path, git snapshot)

Flags: `--policy` (custom path), `--yes` (auto-confirm), `--break-glass` + `--break-glass-reason` (controlled override; still not allowed for hard blocks).

## Policy ladder (summary)
1. **Hard BLOCK**: destructive tools (mkfs/fdisk/dd), catastrophic rm/git clean/reset cases.
2. **ALLOW**: safe read-only commands from policy allowlist.
3. **CONFIRM**: risk signals (mutations, force flags, protected paths, egress, package installs, leave repo root) with preview and safer alternative.
4. **Arbiter (optional)**: stubbed hook to tighten decisions when parsing is uncertain.
5. **Break-glass**: explicit phrase + reason recorded; disabled for hard blocks.

Details in `docs/policy-ladder.md`.

## Logging & audit
Every attempt is written to `.clash/audit.log` (JSONL) with timestamp, cwd, repo root, git summary, decision, signals, preview, approver, break-glass reason, and exit code. View with `clash decision explain <id>`.

## Integrations (MVP)
Wrapper commands run Codex/Gemini/Claude/Copilot via CLASH so their top-level executions are logged. Deep interception of child processes varies by tool; see `docs/integrations.md` for recommended container/devcontainer setup to enforce the chokepoint.

## Threat model & limitations
CLASH mitigates common destructive commands but is not a sandbox. Interpreter one-liners or compiled binaries may still evade detection. See `docs/threat-model.md`.

## Tests & CI
- Unit and golden tests in `internal/...`
- GitHub Actions workflow runs `go vet` and `go test ./...` (badge above reflects status)

## Development
- Requirements: Go 1.21+, git.
- Install deps & verify: `go mod tidy && go vet ./... && go test ./...`
- Build locally: `go build -o clash ./cmd/clash`

## Contributing
- Issues and PRs welcome; please include reproduction steps for bugs.
- Before sending a PR, run the CI commands above and keep changes policy-aware.
- For risky changes (runner/policy), add or update tests under `internal/...`.

## Security / responsible use
- CLASH reduces risk but is not a sandbox; review `docs/threat-model.md` before production use.
- Report security concerns privately via GitHub security advisories or direct contact (open an issue requesting contact details if needed).

## Quick demo session
```
$ ./clash run -- ls
CLASH: allowlisted read-only command

$ ./clash run -- rm temp.txt
CLASH: CONFIRM
- signal: mutating command
Preview: 1 items sample: /path/to/temp.txt
Proceed with execution? [y/N]: y

$ ./clash run -- rm -rf /
CLASH: BLOCKED (hard)
- catastrophic rm target
Safer: narrow path or remove -rf
```

## Status
MVP for Linux/macOS. Windows not tested (documented limitation). Arbiter hook is stubbed; wire to your provider in `arbiter` package.
