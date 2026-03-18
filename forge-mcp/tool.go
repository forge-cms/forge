package forgemcp

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/forge-cms/forge"
)

// toolName builds the MCP tool name for a given operation and content type.
// The type name is converted to lower_snake_case by snakeCase; for example,
// toolName("create", "BlogPost") returns "create_blog_post".
func toolName(operation, typeName string) string {
	return operation + "_" + snakeCase(typeName)
}

// parseToolName splits a tool name of the form "operation_type_snake" on the
// first underscore. Returns ok=false when name contains no underscore.
func parseToolName(name string) (op, typeSnake string, ok bool) {
	op, typeSnake, ok = strings.Cut(name, "_")
	return
}

// moduleForType returns the first MCPWrite module whose TypeName, when
// converted to lower_snake_case, equals typeSnake.
// Returns (nil, false) when no matching module is found.
func (s *Server) moduleForType(typeSnake string) (forge.MCPModule, bool) {
	for _, m := range s.modules {
		if hasMCPOp(m, forge.MCPWrite) && snakeCase(m.MCPMeta().TypeName) == typeSnake {
			return m, true
		}
	}
	return nil, false
}

// moduleForAdminList returns the MCPWrite module for list_{type}s tool names.
// The list tool appends "s" to the type's snake_case name (e.g. "list_posts"
// targets the "post" type), so this helper tries typeSnake with a trailing
// "s" stripped when a direct lookup fails.
func (s *Server) moduleForAdminList(typeSnake string) (forge.MCPModule, bool) {
	if m, ok := s.moduleForType(typeSnake); ok {
		return m, true
	}
	if strings.HasSuffix(typeSnake, "s") {
		return s.moduleForType(typeSnake[:len(typeSnake)-1])
	}
	return nil, false
}

// authorise returns a -32001 error when the caller lacks the Author role,
// which is the minimum required for any MCPWrite operation.
func (s *Server) authorise(ctx forge.Context) *jsonRPCError {
	if forge.HasRole(ctx.User().Roles, forge.Author) {
		return nil
	}
	return &jsonRPCError{Code: -32001, Message: "forbidden"}
}

// authoriseEditor returns a -32001 error when the caller lacks Editor role.
// Editor is the minimum role required for admin read tools (list_{type}s,
// get_{type}). Admin also satisfies this check via the hierarchical role system.
func (s *Server) authoriseEditor(ctx forge.Context) *jsonRPCError {
	if forge.HasRole(ctx.User().Roles, forge.Editor) {
		return nil
	}
	return &jsonRPCError{Code: -32001, Message: "forbidden"}
}

// errorFor maps a forge error to a JSON-RPC error:
//   - [forge.ValidationError] → -32602 (invalid params) with the validation message
//   - [forge.ErrNotFound]      → -32001 (resource not found)
//   - [forge.ErrForbidden]     → -32001 (permission denied)
//   - all other errors         → -32603 (internal error)
func errorFor(err error) *jsonRPCError {
	var ve *forge.ValidationError
	if errors.As(err, &ve) {
		return &jsonRPCError{Code: -32602, Message: ve.Error()}
	}
	if errors.Is(err, forge.ErrNotFound) {
		return &jsonRPCError{Code: -32001, Message: "not found"}
	}
	if errors.Is(err, forge.ErrForbidden) {
		return &jsonRPCError{Code: -32001, Message: "forbidden"}
	}
	return &jsonRPCError{Code: -32603, Message: "internal error: " + err.Error()}
}

// handleToolsList returns the tools/list result: a "tools" array containing
// one entry per MCPWrite operation per registered MCPWrite module, plus two
// admin read tools (list_{type}s, get_{type}) per MCPWrite module.
func (s *Server) handleToolsList() any {
	var tools []mcpTool
	for _, m := range s.modules {
		if !hasMCPOp(m, forge.MCPWrite) {
			continue
		}
		tools = append(tools, mcpToolDefs(m)...)
		tools = append(tools, mcpAdminReadToolDefs(m)...)
	}
	return map[string]any{"tools": tools}
}

// handleToolsCall dispatches a tools/call request to the appropriate module
// operation. Author-level access is enforced before any module method is
// called.
//
// NOTE (zero-value limitation): the update operation works by JSON-merging the
// caller's fields onto the stored item. Fields with required or minimum-length
// constraints cannot be cleared to "" — the merge overlay will trigger a
// -32602 validation error. Unconstrained integer fields set to 0 and
// unconstrained string fields set to "" are accepted through the overlay.
// Callers that need to reset a required field must delete and recreate the
// item.
func (s *Server) handleToolsCall(ctx forge.Context, params json.RawMessage) (any, *jsonRPCError) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &jsonRPCError{Code: -32602, Message: "invalid params: " + err.Error()}
	}
	if p.Name == "" {
		return nil, &jsonRPCError{Code: -32602, Message: "invalid params: name required"}
	}

	if rpcErr := s.authorise(ctx); rpcErr != nil {
		return nil, rpcErr
	}

	op, typeSnake, ok := parseToolName(p.Name)
	if !ok {
		return nil, &jsonRPCError{Code: -32602, Message: "invalid tool name: " + p.Name}
	}

	m, ok := s.moduleForType(typeSnake)
	if !ok && op != "list" {
		return nil, &jsonRPCError{Code: -32602, Message: "unknown tool: " + p.Name}
	}

	args := p.Arguments
	if args == nil {
		args = map[string]any{}
	}

	switch op {
	case "create":
		item, err := m.MCPCreate(ctx, args)
		if err != nil {
			return nil, errorFor(err)
		}
		return toolResult(item), nil

	case "update":
		slug, ok := stringArg(args, "slug")
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "invalid params: slug required"}
		}
		item, err := m.MCPUpdate(ctx, slug, args)
		if err != nil {
			return nil, errorFor(err)
		}
		return toolResult(item), nil

	case "publish":
		slug, ok := stringArg(args, "slug")
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "invalid params: slug required"}
		}
		// Idempotency: avoid double AfterPublish fire and PublishedAt re-stamp
		// when the item is already Published (Flag H).
		existing, err := m.MCPGet(ctx, slug)
		if err != nil {
			return nil, errorFor(err)
		}
		type statuser interface{ GetStatus() forge.Status }
		if st, isStatuser := existing.(statuser); isStatuser && st.GetStatus() == forge.Published {
			return toolResult(map[string]any{"slug": slug, "status": "published"}), nil
		}
		if err := m.MCPPublish(ctx, slug); err != nil {
			return nil, errorFor(err)
		}
		return toolResult(map[string]any{"slug": slug, "status": "published"}), nil

	case "schedule":
		slug, ok := stringArg(args, "slug")
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "invalid params: slug required"}
		}
		atStr, ok := stringArg(args, "scheduled_at")
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "invalid params: scheduled_at required"}
		}
		t, err := time.Parse(time.RFC3339, atStr)
		if err != nil {
			return nil, &jsonRPCError{Code: -32602, Message: "invalid params: scheduled_at must be RFC3339"}
		}
		if err := m.MCPSchedule(ctx, slug, t); err != nil {
			return nil, errorFor(err)
		}
		return toolResult(map[string]any{"slug": slug, "status": "scheduled", "scheduled_at": atStr}), nil

	case "archive":
		slug, ok := stringArg(args, "slug")
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "invalid params: slug required"}
		}
		if err := m.MCPArchive(ctx, slug); err != nil {
			return nil, errorFor(err)
		}
		return toolResult(map[string]any{"slug": slug, "status": "archived"}), nil

	case "delete":
		if rpcErr := s.authoriseEditor(ctx); rpcErr != nil {
			return nil, rpcErr
		}
		slug, ok := stringArg(args, "slug")
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "invalid params: slug required"}
		}
		if err := m.MCPDelete(ctx, slug); err != nil {
			return nil, errorFor(err)
		}
		return toolResult(map[string]any{"deleted": true, "slug": slug}), nil

	case "list":
		lm, ok := s.moduleForAdminList(typeSnake)
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "unknown tool: " + p.Name}
		}
		if rpcErr := s.authoriseEditor(ctx); rpcErr != nil {
			return nil, rpcErr
		}
		var statuses []forge.Status
		if statusStr, ok := stringArg(args, "status"); ok {
			statuses = []forge.Status{forge.Status(statusStr)}
		}
		items, err := lm.MCPList(ctx, statuses...)
		if err != nil {
			return nil, errorFor(err)
		}
		if items == nil {
			items = []any{}
		}
		return toolResult(map[string]any{"items": items}), nil

	case "get":
		gm, ok := s.moduleForType(typeSnake)
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "unknown tool: " + p.Name}
		}
		if rpcErr := s.authoriseEditor(ctx); rpcErr != nil {
			return nil, rpcErr
		}
		slug, ok := stringArg(args, "slug")
		if !ok {
			return nil, &jsonRPCError{Code: -32602, Message: "invalid params: slug required"}
		}
		item, err := gm.MCPGet(ctx, slug)
		if err != nil {
			return nil, errorFor(err)
		}
		return toolResult(item), nil

	default:
		return nil, &jsonRPCError{Code: -32602, Message: "unknown operation: " + op}
	}
}

// handleToolMethod dispatches tool-related JSON-RPC methods.
// Returns (response, true) when the method is handled, (zero, false) otherwise.
// This allows the main handle switch in mcp.go to delegate cleanly.
func (s *Server) handleToolMethod(ctx forge.Context, req jsonRPCRequest) (jsonRPCResponse, bool) {
	switch req.Method {
	case "tools/list":
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  s.handleToolsList(),
		}, true
	case "tools/call":
		result, rpcErr := s.handleToolsCall(ctx, req.Params)
		if rpcErr != nil {
			return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: rpcErr}, true
		}
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result}, true
	}
	return jsonRPCResponse{}, false
}

// toolResult wraps v in the MCP CallToolResult envelope that MCP clients
// require for tools/call responses. The payload is marshalled to JSON and
// embedded as the text of a single content item. This format applies to all
// successful results: create, update, publish, schedule, archive, delete,
// list, and get.
func toolResult(v any) map[string]any {
	data, _ := json.Marshal(v)
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(data)},
		},
		"isError": false,
	}
}

// stringArg extracts a non-empty string value from args under the given key.
// Returns ("", false) if the key is absent, the value is not a string, or the
// value is an empty string.
func stringArg(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}
