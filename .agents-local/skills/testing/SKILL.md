---
name: testing
description: Enforce project-specific test execution rules for agent sessions, especially avoiding manual integration flows in Dev Container contexts.
metadata:
  author: jonathan-tyler
  version: "1.0.0"
---

# Testing

Use this skill when deciding which test commands an agent should run in this repository.

## Rules

- Do not run manual integration tasks from agent sessions in Dev Container contexts.
- Specifically avoid:
  - `shell: manual-itest`
  - `shell: cross-platform-test` (this runs `go run . test`, which includes manual integration flow)
  - `tests/manual/run_manual_integration_tests.sh`
  - `tests/manual/run_manual_integration_tests.ps1`
- These are intended for human-invoked runs from a real WSL window and/or native Windows shell.
- Use this automated, non-manual validation path from agent sessions:
  - `go test ./tests/unit/...`
  - `go test -tags=integration ./tests/integration/...`
- Combined one-liner:
  - `go test ./tests/unit/... && go test -tags=integration ./tests/integration/...`
