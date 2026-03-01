package forge

// MCPOperation is an option flag for the [MCP] function. Only [MCPRead] and
// [MCPWrite] are defined in v1; additional operations will be added in v2.
type MCPOperation string

const (
	// MCPRead signals that this module should be exposed as a read-only MCP
	// resource. No-op in v1 — reserved for v2 implementation.
	MCPRead MCPOperation = "read"

	// MCPWrite signals that this module should be exposed as a read+write MCP
	// resource. No-op in v1 — reserved for v2 implementation.
	MCPWrite MCPOperation = "write"
)

// mcpOption is the concrete [Option] returned by [MCP]. It carries no data in
// v1 because MCP support is not yet implemented.
type mcpOption struct{}

func (mcpOption) isOption() {}

// MCP marks a module as an MCP (Model Context Protocol) resource. In v1 this
// option compiles and satisfies the [Option] interface but has no effect at
// runtime. Implementation lands in v2. See Decision 19.
//
// Example:
//
//	app.Content(&BlogPost{},
//	    forge.At("/posts"),
//	    forge.MCP(forge.MCPRead),
//	)
func MCP(ops ...MCPOperation) Option {
	return mcpOption{}
}
