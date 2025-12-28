# Policy ladder

CLASH evaluates every command through these steps:

1. **Deterministic HARD BLOCK (no override)**
   - mkfs*/fdisk/wipefs/dd to block devices
   - shutdown/reboot
   - `rm -rf /` or `rm -rf ~` or recursive rm leaving repo root
   - `git reset --hard` with dirty tree
   - `git clean -fdx` without dry-run/target

2. **Deterministic ALLOW (fast path)**
   - Safe, read-only commands (`ls`, `cat`, `rg`, `pwd`, `git status/diff/log/show/branch`, `echo`)

3. **Deterministic CONFIRM** when risk signals fire
   - Mutations: rm/rmdir/mv/chmod/chown/git clean/reset/checkout/restore
   - Force flags: -f/--force/--hard/-r
   - Protected paths touched or leaving repo root
   - Network egress (curl/wget/scp/rsync)
   - Package installs/upgrades (npm/pnpm/yarn/pip/brew/apt)
   - `find ... -delete`, `rsync --delete`, `git clean`

4. **Previews**
   - `rm`: count resolved targets and sample list
   - `find -delete`: run without `-delete` and report matches
   - `git clean`: run `git clean -nd` to show would-remove
   - If preview fails or parsing is uncertain, fall back to CONFIRM (and optional arbiter).

5. **Arbiter (optional)**
   - Receives structured inputs (command, signals, reasons). Stub implementation only tightens decisions.

6. **Break-glass**
   - `--break-glass` prompts for the exact phrase `break glass for clash` and records `--break-glass-reason`.
   - Not allowed on hard blocks.

## Defaults & configuration
- Defaults embedded in `configs/default_policy.yaml`
- Repo-level override: create `clash.yaml` (use `clash init`)
- Thresholds: delete_count=50, modify_count=200, preview_sample=20
- Protected paths include system roots, `$HOME`, `.git`, `.env*`
- Options: `allow_outside_repo` (false), `require_clean_tree_for_break_glass` (false)

## Decision outputs
Each decision logs: timestamp, cwd, repo_root, git status counts, command, decision, signals, reasons, preview, approver/break-glass info, exit code.
