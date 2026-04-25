package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// MCPServer is a minimal hand-rolled HTTP MCP server. It speaks
// JSON-RPC 2.0 over HTTP POST and implements the three methods every
// MCP client (mcp2cli, claude-code, …) calls during a typical session:
//
//   initialize  — handshake / capability negotiation
//   tools/list  — enumerate the tools this server exposes
//   tools/call  — invoke a named tool with arguments
//
// Notifications and the streaming SSE half of "streamable HTTP" are
// not implemented — our gws-backed tools are synchronous and can
// answer in a single response. If we add streaming tools later,
// upgrade this to handle SSE on GET.
//
// MCPServer satisfies the Element interface so it slots into the
// gate's registry alongside ProxyElement.
type MCPServer struct {
	name        string
	displayName string
	version     string

	mu    sync.RWMutex
	tools []*MCPTool

	// Refresh is called by the gate's scheduler. Default no-op
	// (gws-backed servers handle their own auth via env vars).
	OnRefresh func(ctx context.Context) error
}

// MCPTool is one callable tool exposed by an MCPServer.
type MCPTool struct {
	Name        string
	Description string
	// InputSchema is a JSON Schema describing the tool's arguments.
	// Encoded as raw JSON because we hand it through verbatim to MCP
	// clients without round-tripping through Go types.
	InputSchema json.RawMessage
	// Handler runs the tool. It receives parsed arguments and must
	// return either text content (string), a JSON-serializable
	// structure, or an error.
	Handler func(ctx context.Context, args map[string]any) (any, error)
}

// NewMCPServer creates an empty server with the given identity.
// Add tools with AddTool before serving requests.
func NewMCPServer(name, displayName, version string) *MCPServer {
	return &MCPServer{
		name:        name,
		displayName: displayName,
		version:     version,
	}
}

// AddTool registers a tool. Safe to call before Start; not safe to
// call concurrently with serving requests (callers add at startup).
func (s *MCPServer) AddTool(t *MCPTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, t)
}

// Name implements Element.
func (s *MCPServer) Name() string { return s.name }

// Handler implements Element. Returns an http.Handler that speaks
// JSON-RPC 2.0 over HTTP POST.
func (s *MCPServer) Handler() http.Handler { return http.HandlerFunc(s.serveHTTP) }

// Refresh implements Element. Calls OnRefresh if set.
func (s *MCPServer) Refresh(ctx context.Context) error {
	if s.OnRefresh != nil {
		return s.OnRefresh(ctx)
	}
	return nil
}

// --- JSON-RPC framing ---

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP error code conventions mirror JSON-RPC's, plus the MCP-specific
// -32000 "method not implemented" range.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

func (s *MCPServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// MCP "streamable HTTP" also defines GET for SSE streams. We
		// don't stream, so respond 405 to GET so clients fall back to
		// POST — no broken half-open SSE.
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, nil, codeParseError, "read body: "+err.Error())
		return
	}
	var req jsonrpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCError(w, nil, codeParseError, "parse: "+err.Error())
		return
	}
	if req.JSONRPC != "2.0" {
		writeJSONRPCError(w, req.ID, codeInvalidRequest, "jsonrpc must be 2.0")
		return
	}

	switch req.Method {
	case "initialize":
		s.handleInitialize(w, req)
	case "notifications/initialized":
		// Notification: no response.
		w.WriteHeader(http.StatusAccepted)
	case "tools/list":
		s.handleToolsList(w, req)
	case "tools/call":
		s.handleToolsCall(r.Context(), w, req)
	case "ping":
		writeJSONRPCResult(w, req.ID, map[string]any{})
	default:
		writeJSONRPCError(w, req.ID, codeMethodNotFound, fmt.Sprintf("method %q not implemented", req.Method))
	}
}

func (s *MCPServer) handleInitialize(w http.ResponseWriter, req jsonrpcRequest) {
	writeJSONRPCResult(w, req.ID, map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities": map[string]any{
			// Only tools today. Add resources/prompts capabilities
			// when we expose those primitives.
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"title":   s.displayName,
			"version": s.version,
		},
	})
}

func (s *MCPServer) handleToolsList(w http.ResponseWriter, req jsonrpcRequest) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.tools))
	for _, t := range s.tools {
		entry := map[string]any{
			"name":        t.Name,
			"description": t.Description,
		}
		if len(t.InputSchema) > 0 {
			entry["inputSchema"] = json.RawMessage(t.InputSchema)
		} else {
			// MCP requires an inputSchema; default to an empty object schema.
			entry["inputSchema"] = map[string]any{"type": "object"}
		}
		out = append(out, entry)
	}
	writeJSONRPCResult(w, req.ID, map[string]any{"tools": out})
}

func (s *MCPServer) handleToolsCall(ctx context.Context, w http.ResponseWriter, req jsonrpcRequest) {
	var p struct {
		Name      string                 `json:"name"`
		Arguments map[string]any         `json:"arguments,omitempty"`
		Meta      map[string]any         `json:"_meta,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		writeJSONRPCError(w, req.ID, codeInvalidParams, "parse params: "+err.Error())
		return
	}

	s.mu.RLock()
	var tool *MCPTool
	for _, t := range s.tools {
		if t.Name == p.Name {
			tool = t
			break
		}
	}
	s.mu.RUnlock()

	if tool == nil {
		writeJSONRPCError(w, req.ID, codeMethodNotFound, fmt.Sprintf("tool %q not found", p.Name))
		return
	}

	result, err := tool.Handler(ctx, p.Arguments)
	if err != nil {
		// MCP convention: errors during tool execution come back as a
		// successful tools/call response with isError=true and the
		// error message as content. Reserves JSON-RPC-level errors
		// for protocol-level failures.
		writeJSONRPCResult(w, req.ID, map[string]any{
			"isError": true,
			"content": []map[string]any{
				{"type": "text", "text": err.Error()},
			},
		})
		return
	}

	// Stringify non-string results as JSON; pass strings through as
	// text content directly.
	content, err := toolResultContent(result)
	if err != nil {
		writeJSONRPCError(w, req.ID, codeInternalError, err.Error())
		return
	}
	writeJSONRPCResult(w, req.ID, map[string]any{"content": content})
}

func toolResultContent(v any) ([]map[string]any, error) {
	switch x := v.(type) {
	case string:
		return []map[string]any{{"type": "text", "text": x}}, nil
	case []byte:
		return []map[string]any{{"type": "text", "text": string(x)}}, nil
	default:
		j, err := json.Marshal(x)
		if err != nil {
			return nil, fmt.Errorf("marshal result: %w", err)
		}
		return []map[string]any{{"type": "text", "text": string(j)}}, nil
	}
}

func writeJSONRPCResult(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jsonrpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func writeJSONRPCError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jsonrpcResponse{
		JSONRPC: "2.0", ID: id, Error: &jsonrpcError{Code: code, Message: msg},
	})
}
