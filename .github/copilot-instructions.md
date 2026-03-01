# Forge — Copilot Instructions

This is the Forge CMS project — a Go web framework designed for how you
actually think. Zero dependencies. AI-first. Production-ready by default.

## Before writing any code

1. Read `DECISIONS.md` — all 22 architectural decisions are locked here.
   Do not work around them. If a decision seems wrong, raise it explicitly.
2. Read `ARCHITECTURE.md` — package structure, request lifecycle, stable interfaces.
3. Read `BACKLOG.md` — current milestone and implementation order.

## Non-negotiable rules

- Zero third-party dependencies in the `forge` core package
- All errors implement `forge.Error` — never raw `errors.New`
- `forge.Context` is an interface, not a struct (Decision 21)
- `forge.DB` is an interface, not `*sql.DB` (Decision 22)
- Go 1.22 minimum — do not use features introduced after 1.22
- `gofmt` always — no exceptions
- godoc comments on every exported symbol

## Before planning or writing anything

**Apply DRY (Don't Repeat Yourself):**
Before proposing or implementing anything, check whether the logic,
type, or pattern already exists elsewhere in the codebase.
Reuse and extend — never duplicate.

**Analyse for performance bottlenecks first:**
Before planning or implementing any feature, identify where the
performance-critical paths are. Consider: allocations per request,
reflection usage (use the sync.Map cache pattern), goroutine overhead,
and SQL query efficiency. Propose the performant solution by default —
not the convenient one.

## Code style

- Single package: `forge` — no sub-packages
- File names are the organisation — keep logic in the correct file
- Prefer interfaces over concrete types in function signatures
- Table-driven tests with `t.Run`
- Benchmarks for anything on the hot path (request handling, validation, scanning)
