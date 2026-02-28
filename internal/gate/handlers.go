package gate

import (
	"context"
	"fmt"
	"strings"
)

// NewEchoHandler returns a handler that echoes the interaction name and args.
func NewEchoHandler() Handler {
	return func(ctx context.Context, req *Request) (*Response, error) {
		out := fmt.Sprintf("echo: %s %s", req.Interaction, strings.Join(req.Args, " "))
		return &Response{
			ExitCode: 0,
			Stdout:   out,
		}, nil
	}
}

// NewErrorHandler returns a handler that always reports the MCP bridge is not connected.
func NewErrorHandler(source string) Handler {
	return func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			ExitCode: 1,
			Stderr:   fmt.Sprintf("MCP bridge not connected: %s", source),
		}, nil
	}
}
