# Forge — Milestone 7 Backlog (v0.7.0)

Production-ready `SQLRepo[T]`, automatic redirect tracking, 410 Gone on archive/delete,
chain collapse, optional DB persistence, and a `/.well-known/redirects.json` inspect
endpoint. Closes the "production-ready by default" gap for storage and content mobility.

**Key decisions:**
- Decision 22 — `forge.DB` interface (already locked; `DB` already lives in `storage.go`)
- Decision 23 — `SQLRepo[T]` SQL placeholder style locked to `$N` (PostgreSQL-compatible)
- Decision 24 — Redirect lookup on 404 path only; chain collapse max depth = 10
- Decision 17 — Redirects and content mobility (amending `RedirectEntry` to add `IsPrefix`)

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | storage.go | ✅ Done | 2026-03-07 |
| 2 | redirects.go | ✅ Done | 2026-03-07 |
| 3 | redirectmanifest.go | 🔲 Not started | — |
| 4 | integration_full_test.go | 🔲 Not started | — |

---

## Layer 7.A — Production storage (no prior M7 dependency)

### Step 1 — `storage.go` (Amendment A19)

**Depends on:** `storage.go` (existing — `forge.DB`, `dbFields`, `MemoryRepo[T]`)
**Decisions:** Decision 22, Decision 23, Amendment A19
**Files:** `storage.go` (extend), `storage_test.go` (extend)

**Scope:** Add `SQLRepo[T]` alongside the existing `MemoryRepo[T]`. Both implement
`Repository[T]`. No new file — one step = one logical unit (both repos live in `storage.go`).

#### 1.1 — SQLRepoOption type

- [x] Define `type SQLRepoOption interface{ isSQLRepoOption() }` marker interface
- [x] Define `type tableOption struct{ name string }` with `isSQLRepoOption()` method
- [x] Implement `func Table(name string) SQLRepoOption` — overrides auto-derived table name
- [x] Implement unexported `tableName[T any]() string`:
  - If `Table()` option was provided, use it
  - Otherwise: snake_case + plural of type name (`BlogPost` → `blog_posts`)
  - Pure stdlib — no third-party; ~10 lines using `strings` and `unicode`

#### 1.2 — SQLRepo[T] struct

- [x] Define `type SQLRepo[T any] struct` with unexported fields: `db DB`, `table string`
- [x] Implement `func NewSQLRepo[T any](db DB, opts ...SQLRepoOption) *SQLRepo[T]`
  - Derives table name from type parameter using `tableName[T]()`
  - Applies any `Table()` override
  - Returns ready-to-use `*SQLRepo[T]`
- [x] Add godoc: "SQLRepo is a production Repository[T] backed by forge.DB. T must embed forge.Node."

#### 1.3 — SQLRepo.FindByID

- [x] Implement `func (r *SQLRepo[T]) FindByID(ctx context.Context, id string) (T, error)`
  - Query: `SELECT * FROM {table} WHERE id = $1`
  - Scan result into T using `dbFields` cache (reuse existing — no duplication)
  - Return `ErrNotFound` wrapped in `forge.Error` when `sql.ErrNoRows`

#### 1.4 — SQLRepo.FindBySlug

- [x] Implement `func (r *SQLRepo[T]) FindBySlug(ctx context.Context, slug string) (T, error)`
  - Query: `SELECT * FROM {table} WHERE slug = $1`
  - Same scan + error-wrapping pattern as `FindByID`

#### 1.5 — SQLRepo.FindAll

- [x] Implement `func (r *SQLRepo[T]) FindAll(ctx context.Context, opts ListOptions) ([]T, error)`
  - Base query: `SELECT * FROM {table}`
  - If `opts.Status` is non-empty: append `WHERE status = ANY($1)` (PostgreSQL) / `WHERE status IN (...)` — use parameterised form
  - `opts.OrderBy`: append `ORDER BY {col}` if non-empty and passes allowlist check (prevent injection)
  - `opts.Limit` / `opts.Offset`: append `LIMIT $N OFFSET $N`
  - Scan each row using `dbFields` cache

#### 1.6 — SQLRepo.Save (upsert)

- [x] Implement `func (r *SQLRepo[T]) Save(ctx context.Context, item T) error`
  - Uses `INSERT INTO {table} ({cols}) VALUES ({$N...}) ON CONFLICT (id) DO UPDATE SET {col=$N...}`
  - Column list and placeholders derived from `dbFields` cache
  - Sets `UpdatedAt` to `time.Now().UTC()` before insert; sets `CreatedAt` only when zero

#### 1.7 — SQLRepo.Delete

- [x] Implement `func (r *SQLRepo[T]) Delete(ctx context.Context, id string) error`
  - Query: `DELETE FROM {table} WHERE id = $1`
  - Return `ErrNotFound` if `sql.Result.RowsAffected() == 0`

#### 1.8 — Tests (storage_test.go extension)

- [x] Implement `mockDB` in `storage_test.go` — satisfies `forge.DB`; records queries/args; returns canned `*sql.Rows` via `sql.OpenDB` with a driver-stub
- [x] `TestSQLRepo_tableName_auto` — `*BlogPost` → `blog_posts`, `*Article` → `articles`
- [x] `TestSQLRepo_tableName_override` — `Table("custom")` overrides derivation
- [x] `TestSQLRepo_FindByID_query` — correct `SELECT * FROM blog_posts WHERE id = $1`
- [x] `TestSQLRepo_FindBySlug_query` — correct `SELECT * FROM blog_posts WHERE slug = $1`
- [x] `TestSQLRepo_Save_insert` — correct `INSERT ... ON CONFLICT` SQL generated
- [x] `TestSQLRepo_Delete_query` — correct `DELETE FROM ... WHERE id = $1`
- [x] `TestSQLRepo_FindAll_noFilter` — no WHERE clause when `opts.Status` is nil
- [x] `TestSQLRepo_FindAll_statusFilter` — WHERE clause appended correctly

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestSQLRepo ./...` — all green
- [x] `go test ./...` — full suite green
- [x] `BACKLOG.md` — step 1 row and summary checkbox updated
- [x] `README.md` — no examples broken by this step
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 7.B — Redirects (depends on Layer 7.A)

### Step 2 — `redirects.go` (Amendment A20 in forge.go)

**Depends on:** `storage.go` (forge.DB), `errors.go`, `forge.go`
**Decisions:** Decision 17 (amended — adds `IsPrefix`), Decision 24, Amendment A20
**Files:** `redirects.go`, `redirects_test.go`

#### 2.1 — RedirectCode type and constants

- [x] Define `type RedirectCode int`
- [x] Define constants: `Permanent RedirectCode = 301`, `Gone RedirectCode = 410`
  *(Note: backlog said `MovedPermanently`; implemented as `Permanent` per README API and readability rules.)*
- [x] Add godoc: explain 301 vs 410 semantics per Decision 17

#### 2.2 — RedirectEntry struct

- [x] Define `RedirectEntry` struct:
  - `From     string`    — absolute path, e.g. `/posts/old-slug`
  - `To       string`    — absolute path; empty string = 410 Gone
  - `Code     RedirectCode`
  - `IsPrefix bool`      — if true, rewrites `/old-prefix/X` → `/To/X` at runtime (Decision 17 amendment)
- [x] Add godoc explaining `IsPrefix` prefix-rewrite semantics

#### 2.3 — From type and Redirects option

- [x] Define `type From string` with godoc
- [x] Define `type redirectsOption struct{ from From; to string }` satisfying `isOption()`
- [x] Implement `func Redirects(from From, to string) Option` — bulk prefix redirect option
  - Used as `app.Content(&BlogPost{}, forge.At("/articles"), forge.Redirects(forge.From("/posts"), "/articles"))`
  - Registers a prefix `RedirectEntry{IsPrefix: true, Code: Permanent}`

#### 2.4 — RedirectStore (memory layer)

- [x] Define `type RedirectStore struct` with unexported fields:
  - `mu      sync.RWMutex`
  - `exact   map[string]RedirectEntry`  — keyed by `From`
  - `prefix  []RedirectEntry`           — sorted descending by `len(From)` (longest-prefix first)
- [x] Implement `func NewRedirectStore() *RedirectStore`
- [x] Implement `func (s *RedirectStore) Add(e RedirectEntry)`:
  - Exact entries: if `e.To` already has an entry in `exact`, collapse chain (A→B + B→C = A→C)
  - Max collapse depth = 10 (panic with descriptive message if exceeded — Decision 24)
  - Prefix entries: append to `prefix` slice, re-sort descending by length
- [x] Implement `func (s *RedirectStore) Get(path string) (RedirectEntry, bool)`:
  - Exact lookup first (O(1) map read)
  - If miss: iterate `prefix` slice (longest first), check `strings.HasPrefix(path, e.From)`
  - Return first match or `(RedirectEntry{}, false)`
- [x] Implement `func (s *RedirectStore) All() []RedirectEntry`:
  - Returns all exact + prefix entries sorted by `From` for deterministic JSON output
- [x] Implement `func (s *RedirectStore) Len() int` — total count (exact + prefix)

#### 2.5 — DB persistence

- [x] Implement `func (s *RedirectStore) Load(ctx context.Context, db DB) error`:
  - `SELECT from_path, to_path, code, is_prefix FROM forge_redirects`
  - Calls `s.Add()` for each row (respects chain collapse)
- [x] Implement `func (s *RedirectStore) Save(ctx context.Context, db DB, e RedirectEntry) error`:
  - `INSERT INTO forge_redirects (from_path, to_path, code, is_prefix) VALUES ($1,$2,$3,$4) ON CONFLICT (from_path) DO UPDATE SET to_path=$2, code=$3, is_prefix=$4`
- [x] Implement `func (s *RedirectStore) Remove(ctx context.Context, db DB, from string) error`:
  - `DELETE FROM forge_redirects WHERE from_path = $1`
- [x] Add godoc to all three: "forge_redirects table must exist — see README for schema"
- [x] Document SQL schema in README.md `Redirects` section

#### 2.6 — HTTP handler (fallback)

- [x] Implement `func (s *RedirectStore) handler() http.Handler` (unexported):
  - Calls `s.Get(r.URL.Path)`
  - Match, `e.To` non-empty: `http.Redirect` — for prefix entry, appends the suffix
  - Match, `e.To` empty: `http.Error(w, "Gone", http.StatusGone)`
  - No match: `http.NotFound(w, r)`
  - Never called for successful requests — only fallback (Decision 24)

#### 2.7 — Amendment A20 in forge.go

- [x] Add `redirectStore *RedirectStore` field to `App` struct
- [x] `New()`: initialise `redirectStore: NewRedirectStore()`
- [x] Add `func (a *App) Redirect(from, to string, code RedirectCode)` with godoc
- [x] Add `func (a *App) RedirectStore() *RedirectStore` — exposes store for `Load`/`Save`/`Remove`
- [x] `App.Content()`: extract `redirectsOption`; if found, calls `a.redirectStore.Add(...)`
- [x] `App.Handler()`: register `a.mux.Handle("/", a.redirectStore.handler())` once

#### 2.8 — Tests (redirects_test.go)

- [x] `TestRedirectStore_exactMatch` — Add + Get returns entry
- [x] `TestRedirectStore_miss` — Get on unknown path returns `(RedirectEntry{}, false)`
- [x] `TestRedirectStore_chainCollapse_301` — A→B + B→C collapses to A→C
- [x] `TestRedirectStore_chainCollapse_goneIsTerminal` — Gone entries not collapsed through
- [x] `TestRedirectStore_prefixMatch` — `IsPrefix=true` entry matches `/posts/hello`
- [x] `TestRedirectStore_exactBeatsPrefix` — exact match wins over prefix
- [x] `TestRedirectStore_prefixRewrite` — `/posts/hello` rewritten to `/articles/hello`
- [x] `TestRedirectStore_handler_301` — handler writes 301 + `Location` header
- [x] `TestRedirectStore_handler_410` — handler writes 410 Gone
- [x] `TestRedirectStore_handler_404` — handler writes 404 for unknown path
- [x] `TestApp_Redirect_permanent` — `app.Redirect(old, new, Permanent)` → 301
- [x] `TestApp_Redirect_gone` — `app.Redirect("/removed", "", Gone)` → 410
- [x] `TestApp_Redirect_chain_collapsed` — two `app.Redirect` calls that chain are collapsed

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestRedirect|TestApp_Redirect ./...` — all green
- [x] `go test ./...` — full suite green
- [x] `BACKLOG.md` — step 2 row and summary checkbox updated
- [x] `README.md` — `forge_redirects` table schema in Redirects section
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 7.C — Redirect manifest (depends on Layer 7.B)

### Step 3 — `redirectmanifest.go` (Amendment A21 in forge.go)

**Depends on:** `redirects.go`, `forge.go` (A20), `cookiemanifest.go` (reuses `manifestAuthOption`)
**Decisions:** Decision 17, Amendment A21
**Files:** `redirectmanifest.go`, `redirectmanifest_test.go`

#### 3.1 — JSON types

- [ ] Define unexported `redirectManifestEntry` struct with JSON tags:
  `From`, `To`, `Code` (int), `IsPrefix`
- [ ] Define unexported `redirectManifest` struct:
  `Site string`, `Generated string` (RFC3339), `Count int`, `Entries []redirectManifestEntry`
- [ ] Implement `buildRedirectManifest(site string, entries []RedirectEntry) redirectManifest`
  - Maps `[]RedirectEntry` → `[]redirectManifestEntry`
  - Sorts by `From` for deterministic output
  - Sets `Generated` to `time.Now().UTC().Format(time.RFC3339)`

#### 3.2 — Handler

- [ ] Implement `newRedirectManifestHandler(site string, store *RedirectStore, opts ...Option) http.Handler`:
  - Serialises on each request (unlike cookie manifest — redirect entries change at runtime)
  - Reuses `manifestAuthOption` from `cookiemanifest.go` for auth guard
  - Returns 401 if auth set and request fails authentication
  - `Content-Type: application/json`, `Cache-Control: no-store`
  - Empty store → `{"count": 0, "entries": []}`  — never 404

#### 3.3 — Amendment A21 in forge.go

- [ ] Add `redirectManifestReg bool` field to `App` struct
- [ ] `App.Handler()`: when `!a.redirectManifestReg`, mount `GET /.well-known/redirects.json`
  with `newRedirectManifestHandler(hostname, a.redirectStore)`
- [ ] Always mounted (even empty store returns valid JSON) — unlike cookie manifest

#### 3.4 — Tests (redirectmanifest_test.go)

- [ ] `TestRedirectManifest_empty` — zero entries → `{"count":0,"entries":[]}`
- [ ] `TestRedirectManifest_fields` — entry fields map correctly including `is_prefix`
- [ ] `TestRedirectManifest_sortedByFrom` — entries sorted alphabetically by `from`
- [ ] `TestRedirectManifest_endpoint_200` — `GET /.well-known/redirects.json` returns 200
- [ ] `TestRedirectManifest_contentType` — `Content-Type: application/json`
- [ ] `TestRedirectManifest_alwaysMounted` — 200 even when store is empty (no entries)
- [ ] `TestRedirectManifest_manifestAuth_401` — rejects unauthenticated request
- [ ] `TestRedirectManifest_manifestAuth_200` — accepts signed Editor token

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestRedirectManifest ./...` — all green
- [ ] `go test ./...` — full suite green
- [ ] `BACKLOG.md` — step 3 row and summary checkbox updated
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 7.D — Cross-milestone integration + README (depends on Layers 7.A + 7.B + 7.C)

### Step 4 — `integration_full_test.go`

**Depends on:** `storage.go` (SQLRepo), `redirects.go`, `redirectmanifest.go`
**Decisions:** Decision 17, Decision 23, Decision 24
**Files:** `integration_full_test.go` (append only — never replace or renumber existing groups)

#### 4.1 — G16: Redirect enforcement (M7, Decision 17)

- [ ] G16 group: `app.Redirect(old, new, MovedPermanently)` → 301 + Location header;
      `app.Redirect("/removed", "", Gone)` → 410; unknown path → 404;
      two `app.Redirect` calls that chain are collapsed (A→C, not A→B→C)

#### 4.2 — G17: Prefix redirect via Redirects(From) (M7 + M2)

- [ ] G17 group: `app.Content(m, forge.At("/articles"), forge.Redirects(forge.From("/posts"), "/articles"))`;
      `GET /posts/hello` → 301 to `/articles/hello`;
      exact entry beats prefix (register exact `/posts/about` → `/about`, prefix entries do not shadow it)

#### 4.3 — G18: Full M7 stack — SQLRepo + manifest + ManifestAuth (M7 + M6 + M1)

- [ ] G18 group:
  - `NewSQLRepo[*testPost](mockDB)` satisfies `Repository[*testPost]`
  - `GET /.well-known/redirects.json` always mounted — 200 + JSON even when store empty
  - After `app.Redirect()` entries added, manifest reflects them
  - `ManifestAuth(BearerHMAC(secret))` → 401 unauthenticated, 200 Editor token

#### 4.4 — README badges

- [ ] Update README.md: Redirects section badge from `🔲 Coming in Milestone 7` → `✅ Available`
- [ ] Update README.md: SQLRepo section badge from `🔲 Coming in Milestone 7` → `✅ Available`

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run "TestFull_redirect|TestFull_manifest_redirect|TestFull_sqlrepo" ./...` — all green
- [ ] `go test ./...` — full suite green
- [ ] `BACKLOG.md` — step 4 row and summary checkbox updated; M7 milestone row marked ✅
- [ ] `README.md` — Redirects + SQLRepo badges updated to ✅ Available
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Completion criteria for Milestone 7

- [ ] `SQLRepo[T]` — satisfies `Repository[T]`; table name auto-derived; `Table()` override; full CRUD
- [ ] `RedirectStore` — exact + prefix lookup; chain collapse; optional DB persistence via `forge.DB`
- [ ] `App.Redirect()` and `Redirects(From)` option — complete public API per Decision 17
- [ ] `GET /.well-known/redirects.json` — always mounted; live serialisation; `ManifestAuth` optional
- [ ] Integration tests G16–G18 appended to `integration_full_test.go` and passing
- [ ] README Redirects + SQLRepo badges updated to ✅ Available
- [ ] `go test ./...` green; `go vet ./...` clean; `gofmt -l .` empty
