# Threat model

## Goals
- Reduce accidental and obvious destructive commands issued by agents.
- Provide human-in-the-loop confirmation for risky operations with previews.
- Audit every attempt for accountability.

## Non-goals
- Perfect security or sandboxing.
- Prevent malicious actors with shell access from bypassing CLASH.
- Deep inspection of arbitrary binaries or complex interpreter payloads.

## Potential bypasses
- Compiled binaries or scripts that perform destructive actions after an innocuous command name.
- `python -c` / `node -e` payloads that hide rm/dd equivalents.
- Privileged processes spawning child shells outside CLASH (e.g., editors running hooks).
- Direct systemd/service actions not routed through the `clash` binary.

## Mitigations
- Argument-aware checks (protected paths, repo boundaries, force flags).
- Previews for rm/find/git clean.
- Optional devcontainer/profile to keep tools inside the chokepoint (see `docs/integrations.md`).
- Audit logging with break-glass reason capture.

## Residual risks
- Shell parsing edge cases and environment variable expansions may evade heuristics.
- Massive glob expansions may be slow; previews limit samples but still rely on filesystem access.
- Arbiter is stubbed; integrate with an external LLM if you need smarter parsing, but keep the rule that it may only tighten decisions.

## Recommended defenses in depth
- Run agent CLIs inside a container where `/bin/sh` points to `clash run -- sh` or similar.
- Keep repos clean; require clean tree before break-glass for production systems.
- Pair CLASH with read-only IAM roles/volumes where appropriate.
