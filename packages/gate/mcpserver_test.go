package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeRPC(t *testing.T, body *bytes.Buffer) jsonrpcResponse {
	t.Helper()
	var resp jsonrpcResponse
	if err := json.Unmarshal(body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v\nbody=%s", err, body.String())
	}
	return resp
}

func rpcCall(t *testing.T, h http.Handler, method string, params any) jsonrpcResponse {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return decodeRPC(t, rec.Body)
}

func TestMCPServer_InitializeReturnsServerInfo(t *testing.T) {
	s := NewMCPServer("fake", "Fake", "1.2.3")
	resp := rpcCall(t, s.Handler(), "initialize", map[string]any{})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	res, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result not a map: %T", resp.Result)
	}
	info, ok := res["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("missing serverInfo: %+v", res)
	}
	if info["name"] != "fake" || info["version"] != "1.2.3" {
		t.Errorf("serverInfo wrong: %+v", info)
	}
}

func TestMCPServer_ToolsListEnumeratesAddedTools(t *testing.T) {
	s := NewMCPServer("fake", "Fake", "0.0.1")
	s.AddTool(&MCPTool{Name: "alpha", Description: "first"})
	s.AddTool(&MCPTool{Name: "beta", Description: "second"})

	resp := rpcCall(t, s.Handler(), "tools/list", nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	res := resp.Result.(map[string]any)
	tools, _ := res["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("got %d tools, want 2", len(tools))
	}
	first := tools[0].(map[string]any)
	if first["name"] != "alpha" || first["description"] != "first" {
		t.Errorf("first tool wrong: %+v", first)
	}
	// Default schema must be present even when not specified.
	if _, ok := first["inputSchema"]; !ok {
		t.Errorf("inputSchema missing from tool listing")
	}
}

func TestMCPServer_ToolsCallInvokesHandler(t *testing.T) {
	var sawArgs map[string]any
	s := NewMCPServer("fake", "Fake", "0.0.1")
	s.AddTool(&MCPTool{
		Name: "echo",
		Handler: func(_ context.Context, args map[string]any) (any, error) {
			sawArgs = args
			return "you said: " + args["msg"].(string), nil
		},
	})

	resp := rpcCall(t, s.Handler(), "tools/call", map[string]any{
		"name":      "echo",
		"arguments": map[string]any{"msg": "hello"},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if sawArgs["msg"] != "hello" {
		t.Errorf("handler saw msg=%v, want %q", sawArgs["msg"], "hello")
	}
	res := resp.Result.(map[string]any)
	content := res["content"].([]any)[0].(map[string]any)
	if content["text"] != "you said: hello" {
		t.Errorf("response content = %v, want %q", content["text"], "you said: hello")
	}
}

func TestMCPServer_ToolsCallSurfaceHandlerErrorAsIsError(t *testing.T) {
	s := NewMCPServer("fake", "Fake", "0.0.1")
	s.AddTool(&MCPTool{
		Name: "boom",
		Handler: func(_ context.Context, _ map[string]any) (any, error) {
			return nil, errors.New("everything is broken")
		},
	})

	resp := rpcCall(t, s.Handler(), "tools/call", map[string]any{"name": "boom"})
	if resp.Error != nil {
		t.Fatalf("expected NO JSON-RPC error (handler errors come back as isError content): %+v", resp.Error)
	}
	res := resp.Result.(map[string]any)
	if isErr, _ := res["isError"].(bool); !isErr {
		t.Errorf("result should have isError=true; got %+v", res)
	}
	content := res["content"].([]any)[0].(map[string]any)
	if !strings.Contains(content["text"].(string), "everything is broken") {
		t.Errorf("error content missing original message: %v", content)
	}
}

func TestMCPServer_ToolsCallUnknownToolReturnsRPCError(t *testing.T) {
	s := NewMCPServer("fake", "Fake", "0.0.1")
	resp := rpcCall(t, s.Handler(), "tools/call", map[string]any{"name": "missing"})
	if resp.Error == nil || resp.Error.Code != codeMethodNotFound {
		t.Errorf("expected method-not-found error, got %+v", resp.Error)
	}
}

func TestMCPServer_RejectsNonPOST(t *testing.T) {
	s := NewMCPServer("fake", "Fake", "0.0.1")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestMCPServer_RejectsBadJSONRPCVersion(t *testing.T) {
	s := NewMCPServer("fake", "Fake", "0.0.1")
	body := bytes.NewBufferString(`{"jsonrpc":"1.0","method":"initialize","id":1}`)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/", body))
	resp := decodeRPC(t, rec.Body)
	if resp.Error == nil || resp.Error.Code != codeInvalidRequest {
		t.Errorf("expected invalid-request error, got %+v", resp.Error)
	}
}

func TestMCPServer_NotificationsInitialized_NoBody(t *testing.T) {
	s := NewMCPServer("fake", "Fake", "0.0.1")
	body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/", body))
	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", rec.Code)
	}
}
