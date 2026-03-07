# Forge — Milestone 6 Backlog (v0.6.0)

Typed cookie declarations, category-enforced consent, and a machine-readable
compliance manifest at `/.well-known/cookies.json`.

**Key decision:** Decision 5 — Cookie consent enforcement (Locked 2025-06-01).
Architecture makes the wrong thing impossible: `Necessary` cookies use `SetCookie`;
all other categories must use `SetCookieIfConsented`, which silently skips if
consent is absent. Consent state is stored in a `Necessary` cookie (`forge_consent`).

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | cookies.go | ✅ Done | 2026-03-07 |
| 2 | cookiemanifest.go | 🔲 Not started | — |
| 3 | integration_full_test.go | 🔲 Not started | — |

---

## Layer 6.A — Cookie primitives (no prior M6 dependency)

### Step 1 — `cookies.go`

**Depends on:** `errors.go` only  
**Decisions:** Decision 5  
**Files:** `cookies.go`, `cookies_test.go`

#### 1.1 — CookieCategory type and constants

- [ ] Define `CookieCategory` as `type CookieCategory string`
- [ ] Define four constants: `Necessary`, `Preferences`, `Analytics`, `Marketing`
- [ ] Add godoc comment to each constant explaining GDPR relevance

#### 1.2 — Cookie struct

- [ ] Define `Cookie` struct with fields:
  - `Name     string`        — cookie name as set on the wire
  - `Category CookieCategory` — determines which set API is legal
  - `Path     string`        — URL path scope (default `"/"` applied in SetCookie)
  - `Domain   string`        — optional domain scope
  - `Secure   bool`          — HTTPS only
  - `HttpOnly bool`          — not accessible to JS
  - `SameSite http.SameSite` — `http.SameSiteStrictMode` recommended default
  - `MaxAge   int`           — seconds; 0 = session; -1 = delete on read
  - `Purpose  string`        — human-readable description for the manifest
- [ ] Add godoc comment explaining enforcement model

#### 1.3 — SetCookie (Necessary only)

- [ ] Implement `SetCookie(w http.ResponseWriter, c Cookie, value string)`
- [ ] Panics with descriptive message if `c.Category != Necessary` — makes misuse visible at first use, not in production
- [ ] Applies `c.Path = "/"` if empty
- [ ] Sets the `http.Cookie` and calls `http.SetCookie`
- [ ] Add godoc comment: "Use SetCookie only for Necessary cookies. For all other categories, use SetCookieIfConsented."

#### 1.4 — SetCookieIfConsented (non-Necessary)

- [ ] Implement `SetCookieIfConsented(w http.ResponseWriter, r *http.Request, c Cookie, value string) bool`
- [ ] Panics if `c.Category == Necessary` (wrong API for Necessary cookies)
- [ ] Calls `ConsentFor(r, c.Category)`; returns `false` without setting cookie if no consent
- [ ] Sets cookie and returns `true` if consent is present
- [ ] Applies `c.Path = "/"` if empty

#### 1.5 — ReadCookie and ClearCookie

- [ ] Implement `ReadCookie(r *http.Request, name string) (string, bool)` — wraps `r.Cookie`, returns ("", false) on miss
- [ ] Implement `ClearCookie(w http.ResponseWriter, c Cookie)` — sets MaxAge=-1+Expires past to expire immediately

#### 1.6 — Consent storage and ConsentFor

- [ ] Define unexported constant `consentCookieName = "forge_consent"` — the Necessary cookie that stores consent
- [ ] Define unexported `consentCookie Cookie` with `Category: Necessary`, `HttpOnly: true`, `Secure: true`, `SameSite: http.SameSiteStrictMode`
- [ ] Implement `ConsentFor(r *http.Request, cat CookieCategory) bool`
  - Reads `forge_consent` cookie value
  - Parses comma-separated category list
  - Returns true if `cat` is in the list OR `cat == Necessary`
- [ ] Implement `GrantConsent(w http.ResponseWriter, cats ...CookieCategory)` — sets/overwrites `forge_consent` with the given categories; `Necessary` is always implicit and must not be stored in the value (it is always true)
- [ ] Implement `RevokeConsent(w http.ResponseWriter)` — clears `forge_consent` cookie

#### 1.7 — Tests (cookies_test.go)

- [ ] `TestCookieCategory_constants` — verify the four constant values
- [ ] `TestSetCookie_necessary` — sets cookie on ResponseWriter correctly
- [ ] `TestSetCookie_panicsOnNonNecessary` — panics if Category != Necessary
- [ ] `TestSetCookieIfConsented_panicsOnNecessary` — panics if Category == Necessary
- [ ] `TestSetCookieIfConsented_noConsent` — returns false, no Set-Cookie header
- [ ] `TestSetCookieIfConsented_withConsent` — GrantConsent first, then returns true
- [ ] `TestReadCookie_hit` — reads cookie added to request
- [ ] `TestReadCookie_miss` — returns ("", false) for absent cookie
- [ ] `TestClearCookie` — Set-Cookie header has MaxAge=0 and past Expires
- [ ] `TestConsentFor_necessary_alwaysTrue` — Necessary always returns true without any consent cookie
- [ ] `TestConsentFor_absent` — returns false when no forge_consent cookie
- [ ] `TestConsentFor_granted` — GrantConsent then ConsentFor returns true
- [ ] `TestRevokeConsent` — after Revoke, ConsentFor returns false

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestCookie|TestSetCookie|TestReadCookie|TestClearCookie|TestConsentFor|TestGrantConsent|TestRevokeConsent ./...` — all green
- [ ] `BACKLOG.md` — step 1 row and summary checkbox updated
- [ ] `README.md` — no examples broken by this step
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 6.B — Compliance manifest (depends on Layer 6.A + forge.go)

### Step 2 — `cookiemanifest.go`

**Depends on:** `cookies.go`, `forge.go`  
**Decisions:** Decision 5  
**Files:** `cookiemanifest.go`, `cookiemanifest_test.go`

#### 2.1 — App.Cookies method and cookieDecls field

- [ ] Add `cookieDecls []Cookie` field to `App` struct in `forge.go`
- [ ] Implement `func (a *App) Cookies(decls ...Cookie)` — appends declarations; idempotent (dedup by name)
- [ ] Add godoc: "Cookies registers cookie declarations for the compliance manifest. Call once at startup."

#### 2.2 — CookieManifest JSON types

- [ ] Define unexported `cookieManifestEntry` struct with JSON tags:
  - `Name`, `Category`, `HttpOnly`, `Secure`, `SameSite` (string), `MaxAge`, `Purpose`
- [ ] Define unexported `cookieManifest` struct: `Site string`, `Generated string` (RFC3339), `Count int`, `Cookies []cookieManifestEntry`
- [ ] Implement unexported `buildManifest(site string, decls []Cookie) cookieManifest`
  — maps `[]Cookie` to `[]cookieManifestEntry`, sets `Site` and `Generated` (time.Now().UTC())

#### 2.3 — ManifestAuth option

- [ ] Define `manifestAuthOption` with `auth AuthFunc` field
- [ ] Implement `ManifestAuth(auth AuthFunc) Option` constructor
- [ ] Implement `applyOption` on `manifestAuthOption`: stores auth on `cookieManifestState`

#### 2.4 — cookieManifestHandler

- [ ] Define unexported `cookieManifestState` struct: `auth AuthFunc` (nil = public)
- [ ] Implement unexported `newCookieManifestHandler(site string, decls []Cookie, opts ...Option) http.Handler`
  - Builds manifest once at construction (static response — cookie declarations don't change at runtime)
  - Marshals to JSON (sorted by name for deterministic output)
  - Returns `http.HandlerFunc` that:
    - Checks auth if `state.auth != nil`; returns 401 on failure
    - Writes `Content-Type: application/json` + 200 + cached JSON body

#### 2.5 — Wire into App.Handler()

- [ ] In `forge.go` `App.Handler()`: if `len(a.cookieDecls) > 0`, mount `GET /.well-known/cookies.json` using `newCookieManifestHandler`
- [ ] Pass `a.cfg.BaseURL` hostname as the `site` argument

#### 2.6 — Tests (cookiemanifest_test.go)

- [ ] `TestCookieManifest_empty` — zero declarations → manifest with `"count": 0, "cookies": []`
- [ ] `TestCookieManifest_fields` — declared cookie appears in manifest with correct fields
- [ ] `TestCookieManifest_sortedByName` — multiple declarations appear sorted alphabetically
- [ ] `TestCookieManifest_endpoint_200` — `GET /.well-known/cookies.json` returns 200 + JSON
- [ ] `TestCookieManifest_contentType` — `Content-Type: application/json`
- [ ] `TestCookieManifest_noDecls_notMounted` — if no `app.Cookies(...)` called, endpoint returns 404
- [ ] `TestCookieManifest_manifestAuth_401` — ManifestAuth rejects unauthenticated request
- [ ] `TestCookieManifest_manifestAuth_200` — ManifestAuth allows authenticated request

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestCookieManifest ./...` — all green
- [ ] `go test ./...` — full suite green
- [ ] `BACKLOG.md` — step 2 row and summary checkbox updated
- [ ] `README.md` — no examples broken by this step
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 6.C — Cross-milestone integration + README (depends on Layers 6.A + 6.B)

### Step 3 — `integration_full_test.go`

**Depends on:** `cookies.go`, `cookiemanifest.go`  
**Decisions:** Decision 5  
**Files:** `integration_full_test.go` (append only — never replace existing groups)

#### 3.1 — G13: Cookie consent enforcement (Decision 5)

- [ ] G13 group: `SetCookie` for Necessary sets header; `ConsentFor` returns false without forge_consent; `SetCookieIfConsented` skips without consent; `GrantConsent` + `SetCookieIfConsented` succeeds; `RevokeConsent` clears consent

#### 3.2 — G14: Consent lifecycle + M1 role integration

- [ ] G14 group: full consent lifecycle — Grant → all categories consented; Revoke → all false; ClearCookie expires header; `ConsentFor(Necessary)` always true regardless of forge_consent state; wired through a module handler that sets a Preferences cookie

#### 3.3 — G15: Full M6 stack — manifest + M2 App integration

- [ ] G15 group: `app.Cookies(...)` + `GET /.well-known/cookies.json` returns correct JSON; multiple declarations sort by name; ManifestAuth with Editor+ role blocks Guest (403) and allows Editor (200); manifest not mounted when no declarations

#### 3.4 — README badges

- [ ] Update README.md: Cookies & Compliance section badge from `🔲 Coming in Milestone 6` → `✅ Available`

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run "TestIntegrationFull/G1[3-5]" ./...` — G13–G15 all green
- [ ] `go test ./...` — full suite green
- [ ] `BACKLOG.md` — step 3 row and summary checkbox updated; M6 milestone row marked ✅
- [ ] `README.md` — Cookies & Compliance badge updated to ✅ Available
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Completion criteria for Milestone 6

- [ ] `cookies.go`: all types, constants, and functions implemented and tested
- [ ] `cookiemanifest.go`: manifest handler + App.Cookies() wired
- [ ] `forge_consent` cookie mechanism: GrantConsent / RevokeConsent / ConsentFor all correct
- [ ] `/.well-known/cookies.json` served when declarations registered; 404 otherwise
- [ ] Integration tests G13–G15 appended to `integration_full_test.go` and passing
- [ ] `README.md` — Cookies & Compliance badge updated to ✅ Available
- [ ] `go test ./...` green; `go vet ./...` clean; `gofmt -l .` empty
