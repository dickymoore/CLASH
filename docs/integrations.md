# Integrations

## Wrapper commands (MVP)
- `clash codex -- [args]`
- `clash gemini -- [args]`
- `clash claude -- [args]`
- `clash copilot -- [args]`

These wrappers run the requested CLI through `clash run` so top-level executions are logged and policy-checked. Child processes started internally by the tool may bypass CLASH depending on the CLI design.

## Recommended devcontainer profile (level 2)
- Run your agent CLI inside a devcontainer or container image where `/usr/local/bin/codex`, `gemini`, `claude`, `copilot` are symlinked to `clash <tool>`.
- Mount only the repo (read/write) and provide minimal additional mounts.
- Avoid mounting host root or `$HOME` unless required; this keeps protected paths small.
- Optionally set `SHELL=/usr/local/bin/clash run -- sh` inside the container to force shell escapes through the harness.

## Known limitations by tool
- **Codex CLI**: Wrapper covers the CLI invocation; internal spawned shells depend on tool settings. Prefer running Codex inside the devcontainer.
- **Gemini CLI**: Similar limitations; ensure the container image removes direct `/bin/bash` access if you want strict chokepointing.
- **Claude Code**: If running as an editor plugin, the wrapper cannot intercept calls launched directly by the editor; use the containerized approach.
- **GitHub Copilot CLI**: Wrapper logs commands but cannot prevent bypass via manual shell commands; containerization recommended.

## Signals for deeper hooks (future)
- Implement a small shim that exposes `CLASH_EXEC=1` and `CLASH_SOCKET` so CLIs can forward every `exec` call through CLASH explicitly.
- Add per-tool plugins under `internal/integrations/` once APIs stabilize.
