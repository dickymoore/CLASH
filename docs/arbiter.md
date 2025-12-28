# Arbiter mode

CLASH can call an optional "arbiter" agent for grey-zone commands where static parsing is uncertain (e.g., `bash -c` or interpreter one-liners).

## Contract
- Input: JSON with command string, signals, reasons, cwd, repo root, git summary.
- Output (strict JSON):
  - `decision`: `ALLOW | CONFIRM | BLOCK` (may only tighten — never relax — the existing decision)
  - `reason`: short text
  - `expected_impact` (optional)
  - `safer_alternative` (optional)

## Enforcement rules
- Arbiter cannot override deterministic hard blocks.
- If the arbiter is unsure, it must choose `CONFIRM`.
- Only decision tightening is allowed (ALLOW→CONFIRM/BLOCK, CONFIRM→BLOCK).

## Stub implementation
The current code provides a stub that always returns CONFIRM. Wire your provider by replacing `internal/arbiter/stub.go` with API calls (OpenAI/Anthropic/Google) and mapping responses into the contract above.

## Configuration
`clash.yaml`:
```yaml
arbiter:
  enabled: true
  provider: openai
  model: gpt-4.1-mini
  api_key_env: OPENAI_API_KEY
```

If `enabled` is false or keys are missing, CLASH skips arbiter calls.
