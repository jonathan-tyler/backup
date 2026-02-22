# AGENTS.md

## Project Intent

This project is a thin, predictable wrapper around `restic`.
The goal is a familiar CLI interface and gap-filling behavior, not a full backup engine rewrite.

## Design Priorities

- Keep it simple and maintainable.
- Prefer explicit behavior over clever abstractions.
- Use positional command arguments where appropriate.
- Keep startup fast and dependency footprint small.
- Preserve script-friendly behavior (stable stdout, errors to stderr, meaningful exit codes).

## Recommended Architecture (Minimal)

Use a lightweight layered design instead of a heavy Command Pattern:

1. **CLI Parse Layer**
   - Validate args and map to a small command model.
   - No business logic beyond input validation.

2. **Dispatch Layer**
   - Route command names to plain handler functions (`run`, `report`, `restore`).
   - Use a simple map/switch dispatch.

3. **Planning Layer**
   - Build the intended action from config/include/exclude/cadence.
   - This is where custom report semantics live.

4. **Restic Adapter Layer**
   - Translate planned actions into `restic` CLI invocations.
   - Keep command construction centralized.

5. **Execution Layer**
   - Execute subprocess commands and capture output/exit status.
   - Isolate side effects here for testability.

## Pattern Guidance

- Prefer **functional-core / imperative-shell**:
  - Pure functions for parsing/planning/report modeling.
  - Imperative side effects only in process execution and file I/O boundaries.
- Avoid introducing full OO Command Pattern unless complexity grows significantly.
- Keep interfaces minimal; one executor interface is usually enough for tests.

## Known Restic Capability Gaps (Important)

Restic is excellent as the backup engine, but some UX goals here are wrapper responsibilities:

- `report ... new`:
  - A first-class "new items that will be backed up" report is not a direct restic UX primitive.
  - This requires wrapper-side planning/comparison logic.

- `report ... excluded`:
  - A first-class "all excluded items" report in your exact format is not directly provided by restic.
  - This also requires wrapper-side scanning and rule evaluation.

Treat these reports as product features of this wrapper, powered by local logic plus restic data where useful.

## Implementation Boundaries

- Do not implement backup/report internals until command surface and tests are stable.
- Keep command semantics explicit in help text and README.
- Build only what current commands require; avoid speculative frameworks.

## Testing Strategy

- Focus first on argument parsing and command dispatch.
- Add tests for planner behavior (include/exclude resolution) before wiring full execution.
- Use integration tests for end-to-end process invocation once planner is stable.

## Markdown Output Rules

- For command examples, always use standalone fenced code blocks.
- Do not embed script/command text inside bullet points.
- Avoid markdown link syntax in command text.
- Prefer plain filesystem paths in command blocks (for example `./tests/integration/...`).

## Future Refactor Trigger

Only consider a heavier command abstraction when:

- many subcommands share complex pre/post hooks,
- command-specific state handling diverges significantly, or
- plugin-like command loading is needed.
