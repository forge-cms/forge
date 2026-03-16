package forge

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// — User methods ——————————————————————————————————————————————————————————

func TestUserHasRole(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		u := User{Roles: []Role{Editor}}
		if !u.HasRole(Editor) {
			t.Fatal("expected HasRole(Editor) to return true for Editor")
		}
	})
	t.Run("hierarchical — Admin satisfies Editor", func(t *testing.T) {
		u := User{Roles: []Role{Admin}}
		if !u.HasRole(Editor) {
			t.Fatal("expected HasRole(Editor) to return true for Admin")
		}
	})
	t.Run("insufficient — Author does not satisfy Editor", func(t *testing.T) {
		u := User{Roles: []Role{Author}}
		if u.HasRole(Editor) {
			t.Fatal("expected HasRole(Editor) to return false for Author")
		}
	})
	t.Run("guest user — no roles", func(t *testing.T) {
		if GuestUser.HasRole(Author) {
			t.Fatal("expected HasRole(Author) to return false for GuestUser")
		}
	})
}

func TestUserIs(t *testing.T) {
	t.Run("exact match returns true", func(t *testing.T) {
		u := User{Roles: []Role{Author}}
		if !u.Is(Author) {
			t.Fatal("expected Is(Author) to return true for Author")
		}
	})
	t.Run("higher role does not satisfy exact match", func(t *testing.T) {
		u := User{Roles: []Role{Admin}}
		if u.Is(Editor) {
			t.Fatal("expected Is(Editor) to return false for Admin")
		}
	})
	t.Run("guest user — no roles", func(t *testing.T) {
		if GuestUser.Is(Guest) {
			t.Fatal("expected Is(Guest) to return false for GuestUser (no roles set)")
		}
	})
}

// — SignToken / decodeToken —————————————————————————————————————————————

// compile-time: signToken's error return must satisfy forge.Error
var _ Error = ErrInternal

func TestSignTokenRoundTrip(t *testing.T) {
	secret := "test-secret-that-is-long-enough-32x"
	original := User{ID: "u1", Name: "Alice", Roles: []Role{Editor}}

	token, err := SignToken(original, secret, 0)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}

	got, err := decodeToken(token, secret)
	if err != nil {
		t.Fatalf("decodeToken: %v", err)
	}

	if got.ID != original.ID {
		t.Errorf("ID: got %q want %q", got.ID, original.ID)
	}
	if got.Name != original.Name {
		t.Errorf("Name: got %q want %q", got.Name, original.Name)
	}
	if len(got.Roles) != 1 || got.Roles[0] != Editor {
		t.Errorf("Roles: got %v want [editor]", got.Roles)
	}
}

func TestSignTokenTampered(t *testing.T) {
	secret := "test-secret-that-is-long-enough-32x"
	token, _ := SignToken(User{ID: "u1", Name: "Alice", Roles: []Role{Editor}}, secret, 0)

	// Tamper: replace first character of payload
	parts := strings.SplitN(token, ".", 2)
	if len(parts[0]) == 0 {
		t.Fatal("payload is empty")
	}
	// Flip the first byte of the payload
	payload := []byte(parts[0])
	payload[0] ^= 0x01
	tampered := string(payload) + "." + parts[1]

	_, err := decodeToken(tampered, secret)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestSignTokenWrongSecret(t *testing.T) {
	token, _ := SignToken(User{ID: "u1"}, "secret-a-long-enough-32-chars-00", 0)
	_, err := decodeToken(token, "secret-b-long-enough-32-chars-00")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestSignTokenWithExpiry(t *testing.T) {
	secret := "test-secret-that-is-long-enough-32x"
	user := User{ID: "u1", Roles: []Role{Author}}

	// A token with a future TTL must decode successfully.
	tok, err := SignToken(user, secret, 24*time.Hour)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	got, err := decodeToken(tok, secret)
	if err != nil {
		t.Fatalf("decodeToken on fresh token: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("ID: got %q want %q", got.ID, user.ID)
	}
}

func TestSignTokenExpiredRejects(t *testing.T) {
	secret := "test-secret-that-is-long-enough-32x"
	// Craft a token whose exp is 1 hour in the past.
	raw, _ := json.Marshal(tokenPayload{
		ID:    "u99",
		Name:  "old",
		Roles: []string{"guest"},
		Exp:   time.Now().Add(-time.Hour).Unix(),
	})
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	sig := tokenHMAC(encoded, secret)
	tok := encoded + "." + sig

	_, err := decodeToken(tok, secret)
	if err == nil {
		t.Fatal("expected ErrUnauth for expired token, got nil")
	}
}

// — BearerHMAC ————————————————————————————————————————————————————————————

const testSecret = "test-hmac-secret-that-is-32chars"

func signedToken(t *testing.T, user User) string {
	t.Helper()
	tok, err := SignToken(user, testSecret, 0)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	return tok
}

func TestBearerHMACValid(t *testing.T) {
	user := User{ID: "u42", Name: "Bob", Roles: []Role{Author}}
	tok := signedToken(t, user)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	fn := BearerHMAC(testSecret)
	got, ok := fn.authenticate(req)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.ID != user.ID {
		t.Errorf("ID: got %q want %q", got.ID, user.ID)
	}
}

func TestBearerHMACInvalid(t *testing.T) {
	tok := signedToken(t, User{ID: "u1"})

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	fn := BearerHMAC("different-secret-32chars-padding!")
	_, ok := fn.authenticate(req)
	if ok {
		t.Fatal("expected ok=false for wrong secret")
	}
}

func TestBearerHMACMissingHeader(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	fn := BearerHMAC(testSecret)
	_, ok := fn.authenticate(req)
	if ok {
		t.Fatal("expected ok=false for missing Authorization header")
	}
}

func TestBearerHMACMalformedHeader(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	fn := BearerHMAC(testSecret)
	_, ok := fn.authenticate(req)
	if ok {
		t.Fatal("expected ok=false for non-Bearer Authorization")
	}
}

// — CookieSession —————————————————————————————————————————————————————————

func TestCookieSessionValid(t *testing.T) {
	user := User{ID: "u99", Name: "Carol", Roles: []Role{Editor}}
	tok := signedToken(t, user)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "forge_session", Value: tok})

	fn := CookieSession("forge_session", testSecret)
	got, ok := fn.authenticate(req)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.ID != user.ID {
		t.Errorf("ID: got %q want %q", got.ID, user.ID)
	}
}

func TestCookieSessionInvalid(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "forge_session", Value: "not.avalid.token"})

	fn := CookieSession("forge_session", testSecret)
	_, ok := fn.authenticate(req)
	if ok {
		t.Fatal("expected ok=false for invalid cookie value")
	}
}

func TestCookieSessionNoCookie(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	fn := CookieSession("forge_session", testSecret)
	_, ok := fn.authenticate(req)
	if ok {
		t.Fatal("expected ok=false for missing cookie")
	}
}

func TestCookieSessionCSRFEnabled(t *testing.T) {
	fn := CookieSession("forge_session", testSecret)
	ca, ok := fn.(csrfAware)
	if !ok {
		t.Fatal("CookieSession must implement csrfAware")
	}
	if !ca.csrfEnabled() {
		t.Fatal("expected csrfEnabled()=true by default")
	}
}

func TestCookieSessionWithoutCSRF(t *testing.T) {
	fn := CookieSession("forge_session", testSecret, WithoutCSRF)
	ca, ok := fn.(csrfAware)
	if !ok {
		t.Fatal("CookieSession must implement csrfAware")
	}
	if ca.csrfEnabled() {
		t.Fatal("expected csrfEnabled()=false when WithoutCSRF passed")
	}
}

// — BasicAuth ————————————————————————————————————————————————————————————

func TestBasicAuthValid(t *testing.T) {
	fn := BasicAuth("alice", "s3cr3t")
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "s3cr3t")

	got, ok := fn.authenticate(req)
	if !ok {
		t.Fatal("expected ok=true for correct credentials")
	}
	if got.ID != "alice" {
		t.Errorf("ID: got %q want %q", got.ID, "alice")
	}
	if got.Name != "alice" {
		t.Errorf("Name: got %q want %q", got.Name, "alice")
	}
	if len(got.Roles) != 1 || got.Roles[0] != Guest {
		t.Errorf("Roles: got %v want [guest]", got.Roles)
	}
}

func TestBasicAuthInvalid(t *testing.T) {
	fn := BasicAuth("alice", "s3cr3t")
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "wrong")

	_, ok := fn.authenticate(req)
	if ok {
		t.Fatal("expected ok=false for wrong password")
	}
}

func TestBasicAuthMissingHeader(t *testing.T) {
	fn := BasicAuth("alice", "s3cr3t")
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	_, ok := fn.authenticate(req)
	if ok {
		t.Fatal("expected ok=false for missing Authorization header")
	}
}

func TestBasicAuthProductionWarn(t *testing.T) {
	fn := BasicAuth("alice", "s3cr3t")
	pw, ok := fn.(productionWarner)
	if !ok {
		t.Fatal("BasicAuth must implement productionWarner")
	}

	var buf bytes.Buffer
	pw.warnIfProduction(&buf)

	if !strings.Contains(buf.String(), "BasicAuth") {
		t.Errorf("warning should mention BasicAuth; got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "non-development") {
		t.Errorf("warning should mention non-development; got: %q", buf.String())
	}
}

// — AnyAuth ———————————————————————————————————————————————————————————————

func TestAnyAuthFirstWins(t *testing.T) {
	// First AuthFunc returns a user; second should not be consulted.
	user := User{ID: "first", Name: "First", Roles: []Role{Author}}
	tok := signedToken(t, user)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	// bearerHMAC succeeds; cookieSession would fail (no cookie).
	fn := AnyAuth(BearerHMAC(testSecret), CookieSession("forge_session", testSecret))
	got, ok := fn.authenticate(req)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.ID != user.ID {
		t.Errorf("ID: got %q want %q", got.ID, user.ID)
	}
}

func TestAnyAuthNoneMatch(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil) // no headers, no cookies
	fn := AnyAuth(BearerHMAC(testSecret), CookieSession("forge_session", testSecret))
	_, ok := fn.authenticate(req)
	if ok {
		t.Fatal("expected ok=false when no AuthFunc matches")
	}
}

func TestAnyAuthForwardsWarn(t *testing.T) {
	fn := AnyAuth(BearerHMAC(testSecret), BasicAuth("admin", "pass"))
	pw, ok := fn.(productionWarner)
	if !ok {
		t.Fatal("AnyAuth must implement productionWarner")
	}

	var buf bytes.Buffer
	pw.warnIfProduction(&buf)

	// BasicAuth's warning should have been forwarded.
	if !strings.Contains(buf.String(), "BasicAuth") {
		t.Errorf("expected forwarded BasicAuth warning; got: %q", buf.String())
	}
}

func TestAnyAuthCSRFAware(t *testing.T) {
	t.Run("CSRF enabled when CookieSession present", func(t *testing.T) {
		fn := AnyAuth(BearerHMAC(testSecret), CookieSession("forge_session", testSecret))
		ca, ok := fn.(csrfAware)
		if !ok {
			t.Fatal("AnyAuth must implement csrfAware")
		}
		if !ca.csrfEnabled() {
			t.Fatal("expected csrfEnabled()=true when CookieSession is in the chain")
		}
	})
	t.Run("CSRF disabled when only BearerHMAC", func(t *testing.T) {
		fn := AnyAuth(BearerHMAC(testSecret))
		ca, ok := fn.(csrfAware)
		if !ok {
			t.Fatal("AnyAuth must implement csrfAware")
		}
		if ca.csrfEnabled() {
			t.Fatal("expected csrfEnabled()=false with only BearerHMAC in chain")
		}
	})
	t.Run("CSRF disabled via WithoutCSRF", func(t *testing.T) {
		fn := AnyAuth(CookieSession("forge_session", testSecret, WithoutCSRF))
		ca := fn.(csrfAware)
		if ca.csrfEnabled() {
			t.Fatal("expected csrfEnabled()=false with WithoutCSRF")
		}
	})
}

func TestCSRFCookieName(t *testing.T) {
	if CSRFCookieName != "forge_csrf" {
		t.Errorf("CSRFCookieName: got %q want %q", CSRFCookieName, "forge_csrf")
	}
}

func TestWithoutCSRFImplementsOption(t *testing.T) {
	// Compile-time: var WithoutCSRF Option — this test documents the runtime check.
	var _ Option = WithoutCSRF
}

// — VerifyBearerToken ———————————————————————————————————————————————————————

func TestVerifyBearerToken(t *testing.T) {
	secret := []byte("test-secret-32-bytes-xxxxxxxxxxxx")
	u := User{ID: "u1", Name: "Alice", Roles: []Role{Editor}}

	t.Run("valid token returns user", func(t *testing.T) {
		tok, err := SignToken(u, string(secret), 0)
		if err != nil {
			t.Fatalf("SignToken: %v", err)
		}
		r, _ := http.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+tok)
		got, ok := VerifyBearerToken(r, secret)
		if !ok {
			t.Fatal("expected ok=true for valid token")
		}
		if got.ID != u.ID {
			t.Errorf("user ID: got %q want %q", got.ID, u.ID)
		}
		if len(got.Roles) == 0 || got.Roles[0] != Editor {
			t.Errorf("user roles: got %v want [Editor]", got.Roles)
		}
	})

	t.Run("missing Authorization header returns GuestUser", func(t *testing.T) {
		r, _ := http.NewRequest(http.MethodGet, "/", nil)
		got, ok := VerifyBearerToken(r, secret)
		if ok {
			t.Fatal("expected ok=false for missing header")
		}
		if got.ID != GuestUser.ID {
			t.Errorf("expected GuestUser, got %+v", got)
		}
	})

	t.Run("wrong secret returns GuestUser", func(t *testing.T) {
		tok, err := SignToken(u, string(secret), 0)
		if err != nil {
			t.Fatalf("SignToken: %v", err)
		}
		r, _ := http.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+tok)
		got, ok := VerifyBearerToken(r, []byte("wrong-secret-32-bytes-xxxxxxxxxxxx"))
		if ok {
			t.Fatal("expected ok=false for wrong secret")
		}
		if got.ID != GuestUser.ID {
			t.Errorf("expected GuestUser, got %+v", got)
		}
	})
}
