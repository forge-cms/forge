package forgemcp

import (
	"testing"

	"github.com/forge-cms/forge"
)

// testPost is a minimal content type for forge-mcp tests.
type testPost struct {
	forge.Node
	Title string `forge:"required,min=3"`
	Body  string
}

func newTestApp(t *testing.T, opts ...forge.Option) *forge.App {
	t.Helper()
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	repo := forge.NewMemoryRepo[*testPost]()
	posts := forge.NewModule[*testPost](
		(*testPost)(nil),
		forge.Repo(repo),
		forge.At("/posts"),
	)
	app.Content(posts, opts...)
	return app
}

// TestNewServer verifies that New collects MCPModule values from the App.
func TestNewServer(t *testing.T) {
	cfg := forge.Config{
		BaseURL: "http://localhost",
		Secret:  []byte("test-secret-32-bytes-xxxxxxxxxxxx"),
	}
	app := forge.New(cfg)
	repo := forge.NewMemoryRepo[*testPost]()

	// Two modules with MCP, one without.
	posts := forge.NewModule[*testPost](
		(*testPost)(nil),
		forge.Repo(repo),
		forge.At("/posts"),
		forge.MCP(forge.MCPRead),
	)
	drafts := forge.NewModule[*testPost](
		(*testPost)(nil),
		forge.Repo(forge.NewMemoryRepo[*testPost]()),
		forge.At("/drafts"),
		forge.MCP(forge.MCPWrite),
	)
	noMCP := forge.NewModule[*testPost](
		(*testPost)(nil),
		forge.Repo(forge.NewMemoryRepo[*testPost]()),
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
