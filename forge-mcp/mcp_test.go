package forgemcp

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/forge-cms/forge"
)

// testMCPPost is the canonical content type for all forge-mcp tests.
// It exercises: required fields, min constraints, a numeric field, a
// json: tag override, and the embedded Node.
type testMCPPost struct {
	forge.Node
	Title  string `forge:"required,min=3"`
	Body   string `forge:"required,min=10"`
	Rating int
	Tags   string `json:"tags"`
}

// newTestApp creates a minimal App with a single /posts module.
// Pass forge.Option values (e.g. forge.MCP(...)) to configure the module.
func newTestApp(t *testing.T, opts ...forge.Option) *forge.App {
	t.Helper()
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	repo := forge.NewMemoryRepo[*testMCPPost]()
	posts := forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(repo),
		forge.At("/posts"),
	)
	app.Content(posts, opts...)
	return app
}

// seedPost saves a testMCPPost with the given slug, status, title, and body
// directly into repo, bypassing lifecycle methods. This avoids a dependency
// on MCPPublish (Step 3) when setting up resource tests.
func seedPost(t *testing.T, repo *forge.MemoryRepo[*testMCPPost], slug string, status forge.Status, title, body string) *testMCPPost {
	t.Helper()
	post := &testMCPPost{
		Node:  forge.Node{ID: forge.NewID(), Slug: slug, Status: status},
		Title: title,
		Body:  body,
	}
	ctx := context.Background()
	if err := repo.Save(ctx, post); err != nil {
		t.Fatalf("seedPost: %v", err)
	}
	return post
}

// newTestCtx returns a forge.Context fit for MCPModule method calls.
func newTestCtx() forge.Context {
	return forge.NewTestContext(forge.GuestUser)
}

// TestNewServer verifies that New collects MCPModule values from the App.
func TestNewServer(t *testing.T) {
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	repo := forge.NewMemoryRepo[*testMCPPost]()

	// Two modules with MCP, one without.
	posts := forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(repo),
		forge.At("/posts"),
		forge.MCP(forge.MCPRead),
	)
	drafts := forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(forge.NewMemoryRepo[*testMCPPost]()),
		forge.At("/drafts"),
		forge.MCP(forge.MCPWrite),
	)
	noMCP := forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(forge.NewMemoryRepo[*testMCPPost]()),
		forge.At("/other"),
	)
	app.Content(posts)
	app.Content(drafts)
	app.Content(noMCP)

	srv := New(app)
	if srv == nil {
		t.Fatal("New returned nil")
	}
	if n := len(app.MCPModules()); n != 2 {
		t.Fatalf("MCPModules length = %d, want 2", n)
	}
}

// TestInputSchema verifies that inputSchema produces correct JSON Schema output.
func TestInputSchema(t *testing.T) {
	fields := []forge.MCPField{
		{Name: "Title", JSONName: "title", Type: "string", Required: true, MinLength: 3, MaxLength: 100},
		{Name: "Body", JSONName: "body", Type: "string"},
		{Name: "Rating", JSONName: "rating", Type: "number"},
		{Name: "Category", JSONName: "category", Type: "string", Enum: []string{"news", "blog"}},
	}

	schema := inputSchema(fields)

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties is not map[string]any")
	}
	if _, ok := props["title"]; !ok {
		t.Error("missing title property")
	}
	if _, ok := props["body"]; !ok {
		t.Error("missing body property")
	}
	req, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required is not []string")
	}
	if len(req) != 1 || req[0] != "title" {
		t.Errorf("required = %v, want [title]", req)
	}
	titleProp := props["title"].(map[string]any)
	if titleProp["minLength"] != 3 {
		t.Errorf("title minLength = %v, want 3", titleProp["minLength"])
	}
	if titleProp["maxLength"] != 100 {
		t.Errorf("title maxLength = %v, want 100", titleProp["maxLength"])
	}
	catProp, ok := props["category"].(map[string]any)
	if !ok {
		t.Fatal("category property missing")
	}
	enum, ok := catProp["enum"].([]string)
	if !ok || len(enum) != 2 {
		t.Errorf("category enum = %v, want [news blog]", catProp["enum"])
	}
}

// TestMCPResourcesList verifies that resources/list returns only Published items
// and formats URIs as forge://{prefix}/{slug}.
func TestMCPResourcesList(t *testing.T) {
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	repo := forge.NewMemoryRepo[*testMCPPost]()
	mod := forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(repo),
		forge.At("/posts"),
		forge.MCP(forge.MCPRead),
	)
	app.Content(mod)

	seedPost(t, repo, "published-post", forge.Published, "Published Post", "body content here")
	seedPost(t, repo, "draft-post", forge.Draft, "Draft Post", "body content here")

	srv := New(app)
	ctx := newTestCtx()

	result := srv.handleResourcesList(ctx)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("handleResourcesList did not return map[string]any")
	}
	resources, ok := m["resources"].([]mcpResource)
	if !ok {
		t.Fatalf("resources field is %T, want []mcpResource", m["resources"])
	}
	if len(resources) != 1 {
		t.Fatalf("got %d resources, want 1 (Published only)", len(resources))
	}
	if resources[0].URI != "forge://posts/published-post" {
		t.Errorf("URI = %q, want %q", resources[0].URI, "forge://posts/published-post")
	}
}

// TestMCPResourcesRead_published verifies that resources/read returns the item's
// JSON-encoded content for a Published item.
// Flag D pattern: item fields are inspected via JSON round-trip to map[string]any.
func TestMCPResourcesRead_published(t *testing.T) {
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	repo := forge.NewMemoryRepo[*testMCPPost]()
	mod := forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(repo),
		forge.At("/posts"),
		forge.MCP(forge.MCPRead),
	)
	app.Content(mod)

	seedPost(t, repo, "hello-world", forge.Published, "Hello World", "body content here")

	srv := New(app)
	ctx := newTestCtx()

	params, _ := json.Marshal(map[string]string{"uri": "forge://posts/hello-world"})
	result, rpcErr := srv.handleResourcesRead(ctx, params)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %+v", rpcErr)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("result is not map[string]any")
	}
	contents, ok := m["contents"].([]resourceContent)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents = %T len=%d, want []resourceContent len=1", m["contents"], len(contents))
	}
	if contents[0].URI != "forge://posts/hello-world" {
		t.Errorf("contents[0].URI = %q, want %q", contents[0].URI, "forge://posts/hello-world")
	}
	if contents[0].MimeType != "application/json" {
		t.Errorf("MimeType = %q, want application/json", contents[0].MimeType)
	}
	// Flag D: JSON round-trip to inspect field values without importing testMCPPost directly.
	var fields map[string]any
	if err := json.Unmarshal([]byte(contents[0].Text), &fields); err != nil {
		t.Fatalf("text is not valid JSON: %v", err)
	}
	if fields["Title"] != "Hello World" {
		t.Errorf("Title = %v, want Hello World", fields["Title"])
	}
}

// TestMCPResourcesRead_draft verifies that resources/read returns a -32001 error
// for a Draft item (lifecycle enforcement).
func TestMCPResourcesRead_draft(t *testing.T) {
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	repo := forge.NewMemoryRepo[*testMCPPost]()
	mod := forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(repo),
		forge.At("/posts"),
		forge.MCP(forge.MCPRead),
	)
	app.Content(mod)

	seedPost(t, repo, "draft-item", forge.Draft, "Draft Item", "body content here")

	srv := New(app)
	ctx := newTestCtx()

	params, _ := json.Marshal(map[string]string{"uri": "forge://posts/draft-item"})
	_, rpcErr := srv.handleResourcesRead(ctx, params)
	if rpcErr == nil {
		t.Fatal("expected error for Draft item, got nil")
	}
	if rpcErr.Code != -32001 {
		t.Errorf("error code = %d, want -32001", rpcErr.Code)
	}
}

// TestMCPResourcesTemplatesList verifies that resources/templates/list returns
// exactly one template per MCPRead module with the correct URITemplate format.
func TestMCPResourcesTemplatesList(t *testing.T) {
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	app.Content(forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(forge.NewMemoryRepo[*testMCPPost]()),
		forge.At("/posts"),
		forge.MCP(forge.MCPRead),
	))
	app.Content(forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(forge.NewMemoryRepo[*testMCPPost]()),
		forge.At("/news"),
		forge.MCP(forge.MCPRead),
	))
	// MCPWrite-only module — must not appear in templates list.
	app.Content(forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(forge.NewMemoryRepo[*testMCPPost]()),
		forge.At("/writeonly"),
		forge.MCP(forge.MCPWrite),
	))

	srv := New(app)
	result := srv.handleResourcesTemplatesList()
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("result is not map[string]any")
	}
	templates, ok := m["resourceTemplates"].([]resourceTemplate)
	if !ok {
		t.Fatalf("resourceTemplates is %T, want []resourceTemplate", m["resourceTemplates"])
	}
	if len(templates) != 2 {
		t.Fatalf("got %d templates, want 2 (MCPRead modules only)", len(templates))
	}
	uriTemplates := map[string]bool{}
	for _, tmpl := range templates {
		uriTemplates[tmpl.URITemplate] = true
		if tmpl.MimeType != "application/json" {
			t.Errorf("MimeType = %q, want application/json", tmpl.MimeType)
		}
	}
	if !uriTemplates["forge://posts/{slug}"] {
		t.Error("missing template for forge://posts/{slug}")
	}
	if !uriTemplates["forge://news/{slug}"] {
		t.Error("missing template for forge://news/{slug}")
	}
}

// newAuthorCtx returns a forge.Context with Author role for write operations.
func newAuthorCtx() forge.Context {
	return forge.NewTestContext(forge.User{ID: "u1", Roles: []forge.Role{forge.Author}})
}

// newWriteApp creates an App with a single /posts module registered with MCPWrite.
// Returns both the App and the underlying repo so tests can seed items directly.
func newWriteApp(t *testing.T, opts ...forge.Option) (*forge.App, *forge.MemoryRepo[*testMCPPost]) {
	t.Helper()
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	repo := forge.NewMemoryRepo[*testMCPPost]()
	allOpts := append([]forge.Option{
		forge.Repo(repo),
		forge.At("/posts"),
		forge.MCP(forge.MCPWrite),
	}, opts...)
	posts := forge.NewModule[*testMCPPost]((*testMCPPost)(nil), allOpts...)
	app.Content(posts)
	return app, repo
}

// — Tool naming ——————————————————————————————————————————————

// TestMCPToolName verifies toolName builds lower_snake_case tool names.
func TestMCPToolName(t *testing.T) {
	tests := []struct {
		op, typeName, want string
	}{
		{"create", "BlogPost", "create_blog_post"},
		{"publish", "testMCPPost", "publish_test_mcp_post"},
		{"delete", "MCPPost", "delete_mcp_post"},
		{"archive", "Post", "archive_post"},
	}
	for _, tc := range tests {
		got := toolName(tc.op, tc.typeName)
		if got != tc.want {
			t.Errorf("toolName(%q, %q) = %q, want %q", tc.op, tc.typeName, got, tc.want)
		}
	}
}

// — tools/list ———————————————————————————————————————————————

// TestMCPToolsList verifies that handleToolsList returns exactly 6 tools for
// an MCPWrite module and that their names follow the convention.
func TestMCPToolsList(t *testing.T) {
	app, _ := newWriteApp(t)
	srv := New(app)

	result := srv.handleToolsList()
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("handleToolsList did not return map[string]any")
	}
	tools, ok := m["tools"].([]mcpTool)
	if !ok {
		t.Fatalf("tools field is %T, want []mcpTool", m["tools"])
	}
	if len(tools) != 6 {
		t.Fatalf("got %d tools, want 6", len(tools))
	}
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{
		"create_test_mcp_post",
		"update_test_mcp_post",
		"publish_test_mcp_post",
		"schedule_test_mcp_post",
		"archive_test_mcp_post",
		"delete_test_mcp_post",
	} {
		if !names[want] {
			t.Errorf("missing tool %q", want)
		}
	}

	// MCPRead-only module must NOT contribute any tools.
	cfg := forge.Config{BaseURL: "http://localhost", Secret: []byte("test-secret-32-bytes-xxxxxxxxxxxx")}
	app2 := forge.New(cfg)
	app2.Content(forge.NewModule[*testMCPPost](
		(*testMCPPost)(nil),
		forge.Repo(forge.NewMemoryRepo[*testMCPPost]()),
		forge.At("/readonly"),
		forge.MCP(forge.MCPRead),
	))
	srv2 := New(app2)
	res2 := srv2.handleToolsList()
	m2 := res2.(map[string]any)
	tools2 := m2["tools"].([]mcpTool)
	if len(tools2) != 0 {
		t.Errorf("MCPRead-only module produced %d tools, want 0", len(tools2))
	}
}

// — tools/call ———————————————————————————————————————————————

// TestMCPToolsCall_create verifies that a valid create call creates a Draft
// item with a non-empty ID and Slug.
func TestMCPToolsCall_create(t *testing.T) {
	app, repo := newWriteApp(t)
	srv := New(app)
	ctx := newAuthorCtx()

	params, _ := json.Marshal(map[string]any{
		"name": "create_test_mcp_post",
		"arguments": map[string]any{
			"Title": "Hello World",
			"Body":  "This is a body that is long enough.",
		},
	})
	result, rpcErr := srv.handleToolsCall(ctx, params)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %+v", rpcErr)
	}
	post, ok := result.(*testMCPPost)
	if !ok {
		t.Fatalf("result is %T, want *testMCPPost", result)
	}
	if post.ID == "" {
		t.Error("created item has empty ID")
	}
	if post.Slug == "" {
		t.Error("created item has empty Slug")
	}
	if post.Status != forge.Draft {
		t.Errorf("created item status = %q, want Draft", post.Status)
	}

	// Verify item is actually in the repo via JSON round-trip.
	gotten, err := repo.FindBySlug(context.Background(), post.Slug)
	if err != nil {
		t.Fatalf("FindBySlug after create: %v", err)
	}
	if gotten.Title != "Hello World" {
		t.Errorf("repo Title = %q, want Hello World", gotten.Title)
	}
}

// TestMCPToolsCall_create_validation verifies that a missing required field
// returns a -32602 error with the validation message.
func TestMCPToolsCall_create_validation(t *testing.T) {
	app, _ := newWriteApp(t)
	srv := New(app)
	ctx := newAuthorCtx()

	// Title is required (min=3) — omit it entirely.
	params, _ := json.Marshal(map[string]any{
		"name": "create_test_mcp_post",
		"arguments": map[string]any{
			"Body": "This is a body that is long enough.",
		},
	})
	_, rpcErr := srv.handleToolsCall(ctx, params)
	if rpcErr == nil {
		t.Fatal("expected error for missing required Title, got nil")
	}
	if rpcErr.Code != -32602 {
		t.Errorf("error code = %d, want -32602", rpcErr.Code)
	}
}

// TestMCPToolsCall_publish verifies that publish transitions a Draft item to
// Published and that PublishedAt is set to a non-zero time.
func TestMCPToolsCall_publish(t *testing.T) {
	app, repo := newWriteApp(t)
	srv := New(app)
	ctx := newAuthorCtx()

	t0 := time.Now().UTC().Add(-time.Second)
	seedPost(t, repo, "my-post", forge.Draft, "My Post", "body content here ok")

	params, _ := json.Marshal(map[string]any{
		"name":      "publish_test_mcp_post",
		"arguments": map[string]any{"slug": "my-post"},
	})
	result, rpcErr := srv.handleToolsCall(ctx, params)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %+v", rpcErr)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is %T, want map[string]any", result)
	}
	if m["status"] != "published" {
		t.Errorf("status = %v, want published", m["status"])
	}

	// Verify state in repo.
	stored, err := repo.FindBySlug(context.Background(), "my-post")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if stored.Status != forge.Published {
		t.Errorf("stored status = %q, want Published", stored.Status)
	}
	if stored.PublishedAt.IsZero() {
		t.Error("PublishedAt is zero, want non-zero")
	}
	if stored.PublishedAt.Before(t0) {
		t.Errorf("PublishedAt %v is before t0 %v", stored.PublishedAt, t0)
	}
}

// TestMCPToolsCall_publish_already_published verifies that publishing an
// already-Published item succeeds without firing AfterPublish a second time
// (Flag H idempotency).
func TestMCPToolsCall_publish_already_published(t *testing.T) {
	var fired int32
	app, repo := newWriteApp(t, forge.On(forge.AfterPublish, func(_ forge.Context, _ *testMCPPost) error {
		atomic.AddInt32(&fired, 1)
		return nil
	}))
	srv := New(app)
	ctx := newAuthorCtx()

	// Seed an already-Published item — AfterPublish was NOT fired during seed.
	seedPost(t, repo, "live-post", forge.Published, "Live Post", "body content here ok")

	params, _ := json.Marshal(map[string]any{
		"name":      "publish_test_mcp_post",
		"arguments": map[string]any{"slug": "live-post"},
	})
	result, rpcErr := srv.handleToolsCall(ctx, params)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %+v", rpcErr)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is %T, want map[string]any", result)
	}
	if m["status"] != "published" {
		t.Errorf("status = %v, want published", m["status"])
	}
	if atomic.LoadInt32(&fired) != 0 {
		t.Errorf("AfterPublish fired %d time(s), want 0 for already-Published item", fired)
	}
}

// TestMCPToolsCall_schedule verifies that a schedule call sets the item to
// Scheduled with ScheduledAt matching the provided RFC3339 time.
func TestMCPToolsCall_schedule(t *testing.T) {
	app, repo := newWriteApp(t)
	srv := New(app)
	ctx := newAuthorCtx()

	seedPost(t, repo, "sched-post", forge.Draft, "Sched Post", "body content here ok")
	futureStr := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	params, _ := json.Marshal(map[string]any{
		"name": "schedule_test_mcp_post",
		"arguments": map[string]any{
			"slug":         "sched-post",
			"scheduled_at": futureStr,
		},
	})
	result, rpcErr := srv.handleToolsCall(ctx, params)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %+v", rpcErr)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is %T, want map[string]any", result)
	}
	if m["status"] != "scheduled" {
		t.Errorf("status = %v, want scheduled", m["status"])
	}
	if m["scheduled_at"] != futureStr {
		t.Errorf("scheduled_at = %v, want %v", m["scheduled_at"], futureStr)
	}

	// Verify state in repo.
	stored, err := repo.FindBySlug(context.Background(), "sched-post")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if stored.Status != forge.Scheduled {
		t.Errorf("stored status = %q, want Scheduled", stored.Status)
	}
	if stored.ScheduledAt == nil {
		t.Error("ScheduledAt is nil, want non-nil")
	}
}

// TestMCPToolsCall_archive verifies that an archive call sets the item to
// Archived.
func TestMCPToolsCall_archive(t *testing.T) {
	app, repo := newWriteApp(t)
	srv := New(app)
	ctx := newAuthorCtx()

	seedPost(t, repo, "arch-post", forge.Published, "Arch Post", "body content here ok")

	params, _ := json.Marshal(map[string]any{
		"name":      "archive_test_mcp_post",
		"arguments": map[string]any{"slug": "arch-post"},
	})
	result, rpcErr := srv.handleToolsCall(ctx, params)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %+v", rpcErr)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is %T, want map[string]any", result)
	}
	if m["status"] != "archived" {
		t.Errorf("status = %v, want archived", m["status"])
	}

	stored, err := repo.FindBySlug(context.Background(), "arch-post")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if stored.Status != forge.Archived {
		t.Errorf("stored status = %q, want Archived", stored.Status)
	}
}

// TestMCPToolsCall_delete verifies that a delete call permanently removes the
// item and returns {"deleted": true, "slug": ...} (Flag F).
func TestMCPToolsCall_delete(t *testing.T) {
	app, repo := newWriteApp(t)
	srv := New(app)
	ctx := newAuthorCtx()

	seedPost(t, repo, "del-post", forge.Draft, "Del Post", "body content here ok")

	params, _ := json.Marshal(map[string]any{
		"name":      "delete_test_mcp_post",
		"arguments": map[string]any{"slug": "del-post"},
	})
	result, rpcErr := srv.handleToolsCall(ctx, params)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %+v", rpcErr)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is %T, want map[string]any", result)
	}
	if m["deleted"] != true {
		t.Errorf("deleted = %v, want true", m["deleted"])
	}
	if m["slug"] != "del-post" {
		t.Errorf("slug = %v, want del-post", m["slug"])
	}

	// The module should now return an error for the deleted slug.
	mods := app.MCPModules()
	if len(mods) == 0 {
		t.Fatal("no MCP modules registered")
	}
	if _, err := mods[0].MCPGet(newAuthorCtx(), "del-post"); err == nil {
		t.Error("MCPGet after MCPDelete should return error, got nil")
	}
}

// TestMCPToolsCall_forbidden verifies that a Guest context receives a -32001
// error before any module method is invoked.
func TestMCPToolsCall_forbidden(t *testing.T) {
	app, _ := newWriteApp(t)
	srv := New(app)
	guestCtx := newTestCtx() // GuestUser — no Author role

	params, _ := json.Marshal(map[string]any{
		"name": "create_test_mcp_post",
		"arguments": map[string]any{
			"Title": "Hello World",
			"Body":  "This is a body that is long enough.",
		},
	})
	_, rpcErr := srv.handleToolsCall(guestCtx, params)
	if rpcErr == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if rpcErr.Code != -32001 {
		t.Errorf("error code = %d, want -32001", rpcErr.Code)
	}
}

// TestMCPToolsCall_update_cannot_clear_field verifies Flag G: attempting to
// clear a required string field by passing "" returns a -32602 validation error
// and leaves the stored value unchanged. This documents the zero-value
// limitation: required fields cannot be cleared via the update tool.
// See the handleToolsCall godoc NOTE for details.
func TestMCPToolsCall_update_cannot_clear_field(t *testing.T) {
	app, repo := newWriteApp(t)
	srv := New(app)
	ctx := newAuthorCtx()

	seedPost(t, repo, "upd-post", forge.Draft, "Original Title", "original body content ok")

	// Attempt to clear Body by passing an empty string.
	// Body has required,min=10 — the overlay will produce a validation error,
	// which prevents the save. The stored Body must remain unchanged.
	params, _ := json.Marshal(map[string]any{
		"name": "update_test_mcp_post",
		"arguments": map[string]any{
			"slug": "upd-post",
			"Body": "",
		},
	})
	_, rpcErr := srv.handleToolsCall(ctx, params)
	if rpcErr == nil {
		t.Fatal("expected validation error for clearing required field, got nil")
	}
	if rpcErr.Code != -32602 {
		t.Errorf("error code = %d, want -32602 (validation)", rpcErr.Code)
	}

	// Body must remain unchanged because the empty overlay was rejected.
	stored, err := repo.FindBySlug(context.Background(), "upd-post")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if stored.Body != "original body content ok" {
		t.Errorf("Body = %q after failed clear, want %q",
			stored.Body, "original body content ok")
	}
}
