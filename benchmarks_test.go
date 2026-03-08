// Package forge — benchmark suite for M5–M8 hot paths.
//
// These benchmarks complement the per-file benchmarks that already exist
// for M1–M4 (node, storage, middleware, schema, sitemap, templatehelpers,
// module, forge). Each benchmark here targets a code path introduced in
// Milestone 5 (auth tokens, RSS feeds), 6 (cookie consent), 7 (redirects),
// or 8 (scheduler tick).
//
// Run all benchmarks with:
//
//	go test -run "^$" -bench "^Benchmark" -benchtime=3s ./...
package forge

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const benchSecret = "bench-secret-key-32-bytes-padding"

// — Auth (M5) ——————————————————————————————————————————————————————————————

// BenchmarkSignToken measures the cost of signing a JWT-style HMAC token.
// This is the hot path called when a user logs in or a session is refreshed.
func BenchmarkSignToken(b *testing.B) {
	user := User{ID: "bench-user", Name: "Bench User", Roles: []Role{Editor}}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, err := SignToken(user, benchSecret, time.Hour)
		if err != nil {
			b.Fatalf("SignToken: %v", err)
		}
	}
}

// BenchmarkBearerHMAC_verify measures the hot path of verifying an
// already-signed bearer token on every authenticated request.
func BenchmarkBearerHMAC_verify(b *testing.B) {
	user := User{ID: "bench-user", Name: "Bench User", Roles: []Role{Editor}}
	token, err := SignToken(user, benchSecret, time.Hour)
	if err != nil {
		b.Fatalf("SignToken: %v", err)
	}

	fn := BearerHMAC(benchSecret)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		got, ok := fn.authenticate(req)
		if !ok {
			b.Fatal("authenticate returned false")
		}
		_ = got
	}
}

// — Cookie consent (M6) ————————————————————————————————————————————————————

// BenchmarkConsentFor_granted measures the cost of checking consent on a
// request that carries a forge_consent cookie granting Analytics access.
// This runs on every non-Necessary SetCookieIfConsented call.
func BenchmarkConsentFor_granted(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  consentCookieName,
		Value: string(Analytics),
	})

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if !ConsentFor(req, Analytics) {
			b.Fatal("ConsentFor returned false")
		}
	}
}

// — Redirect lookup (M7) ———————————————————————————————————————————————————

// BenchmarkRedirectStore_Get_exact measures O(1) exact-match lookup in a
// RedirectStore seeded with 100 entries, targeting the middle entry.
func BenchmarkRedirectStore_Get_exact(b *testing.B) {
	s := NewRedirectStore()
	for i := range 100 {
		s.Add(RedirectEntry{
			From: fmt.Sprintf("/old/article-%d", i),
			To:   fmt.Sprintf("/posts/article-%d", i),
			Code: Permanent,
		})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		e, ok := s.Get("/old/article-50")
		if !ok {
			b.Fatal("Get returned false")
		}
		_ = e
	}
}

// BenchmarkRedirectStore_Get_prefix measures prefix-match lookup in a
// RedirectStore seeded with 50 prefix entries, matching the longest prefix.
// Prefix rules are sorted longest-first so worst case is a full scan of the
// slice — this benchmark captures that.
func BenchmarkRedirectStore_Get_prefix(b *testing.B) {
	s := NewRedirectStore()
	for i := range 50 {
		s.Add(RedirectEntry{
			From:     fmt.Sprintf("/legacy/section-%d/", i),
			To:       fmt.Sprintf("/docs/section-%d/", i),
			Code:     Permanent,
			IsPrefix: true,
		})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		e, ok := s.Get("/legacy/section-25/old-page-slug")
		if !ok {
			b.Fatal("Get returned false")
		}
		_ = e
	}
}

// — Scheduler tick (M8) ————————————————————————————————————————————————————

// BenchmarkScheduler_tick_noop measures the overhead of a scheduler tick
// when the MemoryRepo contains no Scheduled items. This is the steady-state
// cost between publishing events.
func BenchmarkScheduler_tick_noop(b *testing.B) {
	repo := NewMemoryRepo[*testPost]()
	// Seed some Published items — only Scheduled items trigger work.
	for i := range 20 {
		p := &testPost{
			Node:  Node{ID: NewID(), Slug: fmt.Sprintf("post-%d", i), Status: Published},
			Title: fmt.Sprintf("Post %d", i),
		}
		if err := repo.Save(context.Background(), p); err != nil {
			b.Fatalf("seed: %v", err)
		}
	}

	m := NewModule((*testPost)(nil), Repo(repo), At("/posts"))
	bgCtx := NewBackgroundContext("bench.example.com")
	sched := newScheduler([]schedulableModule{m}, bgCtx)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = sched.tick()
	}
}

// — Feed serving (M5) —————————————————————————————————————————————————————

// BenchmarkFeedStore_serve measures the cost of serving an RSS feed from a
// FeedStore pre-seeded with 20 items. This is the hot path on every
// GET /{prefix}/feed.xml request.
func BenchmarkFeedStore_serve(b *testing.B) {
	store := NewFeedStore("Bench Site", "https://bench.example.com")
	items := make([]rssItem, 20)
	pub := time.Now().UTC().Add(-24 * time.Hour)
	for i := range 20 {
		pub = pub.Add(time.Hour)
		items[i] = rssItem{
			Title:       fmt.Sprintf("Item %d", i),
			Link:        fmt.Sprintf("https://bench.example.com/posts/item-%d", i),
			Description: fmt.Sprintf("Description for item %d, covering important topics.", i),
			PubDate:     pub.Format(time.RFC1123Z),
			GUID:        rssGUID{IsPermaLink: "true", Value: fmt.Sprintf("https://bench.example.com/posts/item-%d", i)},
		}
	}
	store.Set("/posts", FeedConfig{Title: "Bench Feed"}, items)

	h := store.ModuleHandler("/posts")
	req := httptest.NewRequest(http.MethodGet, "/posts/feed.xml", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("status = %d; want 200", w.Code)
		}
	}
}
