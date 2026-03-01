package forge

import "testing"

// TestMCP verifies that MCP returns a valid no-op Option for all valid
// combinations of MCPOperation arguments, including zero arguments.
func TestMCP(t *testing.T) {
	tests := []struct {
		name string
		ops  []MCPOperation
	}{
		{"MCPRead only", []MCPOperation{MCPRead}},
		{"MCPWrite only", []MCPOperation{MCPWrite}},
		{"MCPRead and MCPWrite", []MCPOperation{MCPRead, MCPWrite}},
		{"no args", []MCPOperation{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opt := MCP(tc.ops...)
			if opt == nil {
				t.Fatal("MCP() returned nil, want non-nil Option")
			}
			if _, ok := opt.(mcpOption); !ok {
				t.Errorf("MCP() type = %T, want mcpOption", opt)
			}
		})
	}
}
