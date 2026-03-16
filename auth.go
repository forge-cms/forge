package forge

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AuthFunc authenticates an incoming HTTP request and returns the identified
// User and whether authentication succeeded. Use [BearerHMAC], [CookieSession],
// [BasicAuth], or [AnyAuth] to obtain an AuthFunc. Implement this interface to
// provide a custom authentication scheme.
//
// The unexported authenticate method is intentional: it prevents accidental
// direct calls and allows future additions to the interface without breaking
// existing implementations (consistent with [Option] and [Signal]).
type AuthFunc interface {
	authenticate(*http.Request) (User, bool)
}

// productionWarner is an optional capability interface implemented by AuthFunc
// values that should emit a warning when used outside of development.
// Step 11 (forge.go) type-asserts each registered AuthFunc to this interface.
type productionWarner interface {
	warnIfProduction(w io.Writer)
}

// csrfAware is an optional capability interface implemented by AuthFunc values
// that manage CSRF validation. Step 9 (middleware.go) type-asserts each registered
// AuthFunc to decide whether to validate CSRF tokens on non-safe HTTP methods.
type csrfAware interface {
	csrfEnabled() bool
}

// CSRFCookieName is the name of the CSRF cookie set by [CookieSession].
// Client-side AJAX code should read this cookie and send its value as the
// X-CSRF-Token request header on all non-safe methods (POST, PUT, PATCH, DELETE).
const CSRFCookieName = "forge_csrf"

// WithoutCSRF is an [Option] passed to [CookieSession] to disable automatic
// CSRF protection. This is strongly discouraged for production use.
var WithoutCSRF Option = withoutCSRFOption{}

// withoutCSRFOption is the unexported implementation of the WithoutCSRF option.
type withoutCSRFOption struct{}

func (withoutCSRFOption) isOption() {}

// HasRole reports whether the user holds at least the given role level.
// This is hierarchical: an Admin satisfies HasRole(forge.Editor).
// Delegates to the free function [HasRole] in roles.go.
func (u User) HasRole(role Role) bool {
	return HasRole(u.Roles, role)
}

// Is reports whether the user holds exactly the given role (exact match only).
// An Admin does not satisfy Is(forge.Editor).
// Delegates to the free function [IsRole] in roles.go.
func (u User) Is(role Role) bool {
	return IsRole(u.Roles, role)
}

// tokenPayload is the JSON structure embedded in signed tokens and session cookies.
type tokenPayload struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
	Exp   int64    `json:"exp,omitempty"` // Unix seconds; 0 means no expiry
}

// encodeToken JSON-marshals user, computes HMAC-SHA256 over the base64url payload,
// and returns "payload.signature" (both base64url-encoded, no padding).
// When ttl > 0 an expiry timestamp is embedded in the payload.
func encodeToken(user User, secret string, ttl time.Duration) (string, error) {
	roles := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roles[i] = string(r)
	}

	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).Unix()
	}

	raw, err := json.Marshal(tokenPayload{ID: user.ID, Name: user.Name, Roles: roles, Exp: exp})
	if err != nil {
		// json.Marshal on tokenPayload (string/[]string/int64 fields) is
		// unreachable in practice; return a forge.Error per Decision 16.
		return "", ErrInternal
	}

	payload := base64.RawURLEncoding.EncodeToString(raw)
	sig := tokenHMAC(payload, secret)
	return payload + "." + sig, nil
}

// decodeToken reverses encodeToken. Returns [ErrUnauth] if the token is missing,
// malformed, or the HMAC signature does not match.
func decodeToken(token, secret string) (User, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return GuestUser, ErrUnauth
	}
	payload, sig := parts[0], parts[1]

	expected := tokenHMAC(payload, secret)
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return GuestUser, ErrUnauth
	}

	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return GuestUser, ErrUnauth
	}

	var p tokenPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return GuestUser, ErrUnauth
	}

	// Reject expired tokens.
	if p.Exp != 0 && time.Now().Unix() > p.Exp {
		return GuestUser, ErrUnauth
	}

	roles := make([]Role, len(p.Roles))
	for i, r := range p.Roles {
		roles[i] = Role(r)
	}

	return User{ID: p.ID, Name: p.Name, Roles: roles}, nil
}

// tokenHMAC computes a base64url-encoded (no padding) HMAC-SHA256 of payload
// using secret as the key.
func tokenHMAC(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// SignToken produces a signed token encoding the given User. Pass the token to
// the client (e.g. as a JSON response body); validate it later with [BearerHMAC]
// or [CookieSession].
//
// When ttl > 0 the token contains an expiry timestamp; [decodeToken] rejects
// tokens whose expiry has passed. Use ttl = 0 for tokens with no expiry.
//
// The token format is: base64url(json(User)) + "." + base64url(hmac-sha256(secret, payload)).
// Roles are stored as strings for forward compatibility (Decision 15).
func SignToken(user User, secret string, ttl time.Duration) (string, error) {
	return encodeToken(user, secret, ttl)
}

// — BearerHMAC —————————————————————————————————————————————————————————————

// bearerAuthFn implements [AuthFunc] for HMAC-signed bearer tokens.
type bearerAuthFn struct {
	secret string
}

// BearerHMAC returns an [AuthFunc] that validates HMAC-signed bearer tokens
// from the Authorization header (format: "Bearer <token>"). Generate tokens
// with [SignToken].
func BearerHMAC(secret string) AuthFunc {
	return &bearerAuthFn{secret: secret}
}

func (b *bearerAuthFn) authenticate(r *http.Request) (User, bool) {
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return GuestUser, false
	}
	token := strings.TrimPrefix(hdr, "Bearer ")
	user, err := decodeToken(token, b.secret)
	if err != nil {
		return GuestUser, false
	}
	return user, true
}

// VerifyBearerToken extracts and verifies the HMAC-signed bearer token from r's
// Authorization header. It returns the authenticated [User] and true on success,
// or [GuestUser] and false if the header is absent, malformed, or the signature
// is invalid. secret must be the same value used to sign the token with [SignToken].
// This is the public counterpart to the unexported authenticate method on
// [BearerHMAC] and is intended for use outside the forge package (e.g. forge-mcp
// SSE transport) where [AuthFunc] is not directly callable.
func VerifyBearerToken(r *http.Request, secret []byte) (User, bool) {
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return GuestUser, false
	}
	token := strings.TrimPrefix(hdr, "Bearer ")
	user, err := decodeToken(token, string(secret))
	if err != nil {
		return GuestUser, false
	}
	return user, true
}

// — CookieSession ——————————————————————————————————————————————————————————

// cookieAuthFn implements [AuthFunc] and [csrfAware] for cookie-based sessions.
type cookieAuthFn struct {
	name   string
	secret string
	csrf   bool
}

// CookieSession returns an [AuthFunc] that reads a named cookie containing a
// signed user token (same format as [BearerHMAC]). CSRF protection is enabled
// by default — pass [WithoutCSRF] to opt out (strongly discouraged).
//
// The CSRF cookie is named [CSRFCookieName]. See [Amendment S6].
func CookieSession(name, secret string, opts ...Option) AuthFunc {
	csrf := true
	for _, o := range opts {
		if _, ok := o.(withoutCSRFOption); ok {
			csrf = false
		}
	}
	return &cookieAuthFn{name: name, secret: secret, csrf: csrf}
}

func (c *cookieAuthFn) authenticate(r *http.Request) (User, bool) {
	cookie, err := r.Cookie(c.name)
	if err != nil {
		return GuestUser, false
	}
	user, err := decodeToken(cookie.Value, c.secret)
	if err != nil {
		return GuestUser, false
	}
	return user, true
}

func (c *cookieAuthFn) csrfEnabled() bool {
	return c.csrf
}

// — BasicAuth ——————————————————————————————————————————————————————————————

const basicAuthWarn = `WARN  forge: BasicAuth is enabled in a non-development environment.
      BasicAuth sends credentials on every request and has no session management.
      Consider forge.BearerHMAC or forge.CookieSession for production use.`

// basicAuthFn implements [AuthFunc] and [productionWarner] for HTTP Basic Auth.
type basicAuthFn struct {
	username string
	password string
}

// BasicAuth returns an [AuthFunc] that validates HTTP Basic Auth credentials.
// On success it returns a synthetic User with ID and Name set to the username
// and Roles set to [Guest].
//
// BasicAuth should not be used in production. Consider [BearerHMAC] or
// [CookieSession] for production use. See Amendment S7.
func BasicAuth(username, password string) AuthFunc {
	return &basicAuthFn{username: username, password: password}
}

func (b *basicAuthFn) authenticate(r *http.Request) (User, bool) {
	u, p, ok := r.BasicAuth()
	if !ok {
		return GuestUser, false
	}
	uMatch := subtle.ConstantTimeCompare([]byte(u), []byte(b.username))
	pMatch := subtle.ConstantTimeCompare([]byte(p), []byte(b.password))
	if uMatch&pMatch != 1 {
		return GuestUser, false
	}
	return User{ID: b.username, Name: b.username, Roles: []Role{Guest}}, true
}

func (b *basicAuthFn) warnIfProduction(w io.Writer) {
	fmt.Fprintln(w, basicAuthWarn)
}

// — AnyAuth ————————————————————————————————————————————————————————————————

// anyAuthFn implements [AuthFunc], [productionWarner], and [csrfAware].
// It wraps a list of AuthFunc values and returns the first successful result.
type anyAuthFn struct {
	fns []AuthFunc
}

// AnyAuth returns an [AuthFunc] that tries each provided AuthFunc in order and
// returns the first successful result. If none match, it returns [GuestUser].
//
// AnyAuth forwards [productionWarner] and [csrfAware] capability calls to any
// child that implements them.
func AnyAuth(fns ...AuthFunc) AuthFunc {
	return &anyAuthFn{fns: fns}
}

func (a *anyAuthFn) authenticate(r *http.Request) (User, bool) {
	for _, fn := range a.fns {
		if user, ok := fn.authenticate(r); ok {
			return user, true
		}
	}
	return GuestUser, false
}

func (a *anyAuthFn) warnIfProduction(w io.Writer) {
	for _, fn := range a.fns {
		if pw, ok := fn.(productionWarner); ok {
			pw.warnIfProduction(w)
		}
	}
}

func (a *anyAuthFn) csrfEnabled() bool {
	for _, fn := range a.fns {
		if ca, ok := fn.(csrfAware); ok {
			if ca.csrfEnabled() {
				return true
			}
		}
	}
	return false
}
