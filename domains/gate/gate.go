// Package gate provides the public API for the gate domain.
// It wraps server and bridge operations for element bridging between host and container.
package gate

import (
	"github.com/jterrazz/spwn/domains/gate/internal/bridge"
	"github.com/jterrazz/spwn/domains/gate/internal/server"
)

// GateBridge represents an element bridged from the Host.
type GateBridge = bridge.GateBridge

// InvokeRequest is the wire format for element bridge calls.
type InvokeRequest = server.InvokeRequest

// InvokeHandler processes an element bridge invocation.
type InvokeHandler = server.InvokeHandler

// InvokeResult is the response from an element bridge call.
type InvokeResult = server.InvokeResult

// Server is an HTTP-over-TCP server for the host-side Gate.
type Server = server.Server

// NewServer creates a Gate server that listens on an ephemeral TCP port.
func NewServer(handler InvokeHandler) *Server {
	return server.NewServer(handler)
}

// StubHandler returns a handler that always reports "not implemented".
func StubHandler() InvokeHandler {
	return server.StubHandler()
}

// SetupBridges writes wrapper scripts for each gate bridge into gateDir/bin/.
func SetupBridges(gateDir string, bridges []GateBridge, gatePort int) error {
	return bridge.SetupBridges(gateDir, bridges, gatePort)
}

// GenerateWrapperScript returns a shell script that bridges an element invocation.
func GenerateWrapperScript(elementName string, gatePort int) string {
	return bridge.GenerateWrapperScript(elementName, gatePort)
}
