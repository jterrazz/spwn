package bridge

import "testing"

func TestCheckCapability_EmptyAllowsAll(t *testing.T) {
	b := GateBridge{As: "tool", Capabilities: nil}
	if err := CheckCapability(b, "anything"); err != nil {
		t.Fatalf("expected nil error for empty capabilities, got: %v", err)
	}
}

func TestCheckCapability_AllowedOperation(t *testing.T) {
	b := GateBridge{As: "tool", Capabilities: []string{"read", "write"}}
	if err := CheckCapability(b, "read"); err != nil {
		t.Fatalf("expected nil error for allowed operation, got: %v", err)
	}
	if err := CheckCapability(b, "write"); err != nil {
		t.Fatalf("expected nil error for allowed operation, got: %v", err)
	}
}

func TestCheckCapability_DeniedOperation(t *testing.T) {
	b := GateBridge{As: "tool", Capabilities: []string{"read"}}
	if err := CheckCapability(b, "write"); err == nil {
		t.Fatal("expected error for denied operation, got nil")
	}
}

func TestCheckCapability_EmptySliceAllowsAll(t *testing.T) {
	b := GateBridge{As: "tool", Capabilities: []string{}}
	if err := CheckCapability(b, "anything"); err != nil {
		t.Fatalf("expected nil error for empty slice capabilities, got: %v", err)
	}
}
