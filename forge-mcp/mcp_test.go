package forgemcp

import (
	"context"
	"encoding/json"
	"testing"

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
