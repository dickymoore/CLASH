# Review Follow-Ups Inbox (BMM)
This file captures non-blocking review feedback raised during PRs (Claude Code, human review)
that is intentionally deferred to avoid destabilising merge-ready work.
Items here are candidates for tech debt investigations, hardening passes, refactors,
governance updates, and standards improvements.

## BMM-INBOX-2025-01-07-001
**Source:** Claude Code on PR #15 (secret redaction ordering)
**Type:** Tech Debt / Hardening
**Area:** Infra / Secrets handling
**Severity:** Low (non-blocking)
**Status:** Inbox

### Summary
Reorder secret redaction logic to avoid edge-case truncation leaks.

### Context
Current implementation already passes CI and acceptance; suggestion is purely defensive hardening around redaction order.

### Why deferred
Non-blocking and was intentionally postponed to avoid PR drift; needs follow-up outside current merge scope.

### Proposed follow-up
Investigate safer redaction ordering and consider introducing a shared helper for consistent secret redaction.

### Notes
Raised by Claude Code review on PR #15; no functional bug reported, but worth hardening.
