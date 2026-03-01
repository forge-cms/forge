# Forge — Copilot Instructions

This is the Forge CMS project — a Go web framework designed for how you
actually think. Zero dependencies. AI-first. Production-ready by default.

## Before writing any code

1. Read `DECISIONS.md` — all 22 architectural decisions are locked here.
   Do not work around them. If a decision seems wrong, raise it explicitly.
2. Read `ARCHITECTURE.md` — package structure, request lifecycle, stable interfaces.
3. Read `BACKLOG.md` — current milestone and implementation order.
4. Read the milestone backlog file for the current milestone (e.g. `Milestone1_BACKLOG.md`).
   This is the authoritative task list. Do not implement anything not listed there.
   Do not skip steps — the order is load-bearing (dependency layers).

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

---

## Milestone planning process

Before implementing any milestone, a dedicated backlog file must be created and
agreed upon. This file is the single source of truth for that milestone.

### When to create a milestone backlog

Create `Milestone{N}_BACKLOG.md` in the repo root before writing any code for
that milestone. The file must be reviewed and confirmed before implementation starts.

### Structure of a milestone backlog file

The file follows this structure exactly:

```
# Forge — Milestone {N} Backlog (v{semver})

One-line description of the milestone goal.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1    | foo.go | 🔲 Not started | — |
...

---

## Layer {N} — {Layer name} ({dependency note})

### Step {N} — {filename}

**Depends on:** {nothing | list of files}
**Decisions:** {Decision numbers and Amendment IDs}
**Files:** `{impl file}`, `{test file}`

#### {N}.{M} — {Sub-section name}

- [ ] Specific, atomic implementation task
- [ ] Another task
...

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run Test{Name} ./...` — all green

---

## Completion criteria for Milestone {N}

- [ ] Criterion 1
...
```

### Rules for steps

- **One step = one file** (implementation + test file). Never mix two files in one step.
- **Steps are ordered by dependency layer** — a step may not be started until all
  steps it depends on are marked ✅.
- **Sub-sections (N.M)** break the step into logical implementation chunks: define
  the type, implement the logic, write the tests, verify. Keep sub-sections small
  enough that each can be completed and verified in one sitting.
- **Every sub-section ends with a verification block** for the step it belongs to,
  or the step ends with a shared verification block if substeps are tightly coupled.
- **Checkboxes are atomic** — each `- [ ]` item must be a single, unambiguous task.
  Never write "implement X" without specifying what X requires.
- **Every step ends with an architecture and decision review.** After the verification
  block passes, review `ARCHITECTURE.md` and `DECISIONS.md` and ask:
  - Does the implementation reveal a gap, ambiguity, or conflict in an existing decision?
  - Did any implementation choice introduce a pattern or constraint not yet captured?
  - Does the file's dependency graph still match the rules in `ARCHITECTURE.md`?
  If yes to any of the above, a new Decision or Amendment must be proposed and agreed
  upon before the next step begins. The step is not complete until this review is done.
  Add the following checkbox at the end of every step's verification block:
  ```
  - [ ] Review ARCHITECTURE.md and DECISIONS.md — no new decisions required,
        or new Decision/Amendment drafted and agreed upon
  ```

### Progress tracking

- Mark a step `🔲 In progress` in the Progress table when work begins.
- Mark a step `✅ Done` and record the date when all its checkboxes are ticked
  and its verification block passes.
- Never mark a step done if `go test ./...` is red.
- Update `Milestone1_BACKLOG.md` (or the relevant file) after every step — do not
  batch updates.

### Naming convention

| Milestone | File |
|-----------|------|
| Milestone 1 | `Milestone1_BACKLOG.md` |
| Milestone 2 | `Milestone2_BACKLOG.md` |
| … | … |
