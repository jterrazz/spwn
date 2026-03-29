// Package gate provides the public API for the gate domain.
// It wraps server and bridge operations for element bridging between host and container.
package gate

import (
	"spwn.sh/core/gate/internal/bridge"
	"spwn.sh/core/gate/internal/server"
)

// Bridge represents an element bridged from the Host.
type Bridge = bridge.GateBridge

// InvokeRequest is the wire format for element bridge calls.
type InvokeRequest = server.InvokeRequest

// InvokeHandler processes an element bridge invocation.
type InvokeHandler = server.InvokeHandler

// InvokeResult is the response from an element bridge call.
type InvokeResult = server.InvokeResult

// Server is an HTTP-over-TCP server for the host-side Gate.
type Server = server.Server

// NewServer returns a Gate HTTP server that listens on an ephemeral TCP port
// and dispatches element bridge invocations to the given handler.
func NewServer(handler InvokeHandler) *Server {
	return server.NewServer(handler)
}

// StubHandler returns an InvokeHandler that responds to every invocation with
// a "not implemented" error. Useful for testing and placeholder wiring.
func StubHandler() InvokeHandler {
	return server.StubHandler()
}

// ExecHandler returns an InvokeHandler that executes the configured bridge
// source command on the host for each matching element invocation.
func ExecHandler(bridges []Bridge) InvokeHandler {
	return server.ExecHandler(bridges)
}

// SetupBridges generates executable wrapper scripts for each gate bridge and
// writes them into gateDir/bin/, where the container-side gate can discover them.
func SetupBridges(gateDir string, bridges []Bridge, gatePort int) error {
	return bridge.SetupBridges(gateDir, bridges, gatePort)
}

// GenerateWrapperScript returns the shell script contents for a single element
// bridge wrapper that forwards invocations to the host Gate server at gatePort.
func GenerateWrapperScript(elementName string, gatePort int) string {
	return bridge.GenerateWrapperScript(elementName, gatePort)
}
