package agent

import (
	"strings"
	"testing"

	"spwn.sh/core/universe"
)

// These tests pin the routing rules for "talk to agent X in world Y". The
// original bug: five worlds all had agent "qa"; the routing returned the
// first matching world regardless of which world the user was talking from.
// The observatory handler was then execing `spwn agent talk qa ...` without
// a world pin, so messages sent from "The Test" ended up in "Matrix" or
// another world's container.

func running(_ string) bool  { return true }
func stopped(_ string) bool  { return false }
func onlyA(id string) bool   { return id == "cA" }

func makeWorld(id, container, agent string) universe.World {
	return universe.World{ID: id, ContainerID: container, Agent: agent}
}

func TestRouteAgentToWorld_PinnedWinsWhenMultipleShareAgentName(t *testing.T) {
	// THIS is the regression case. Five worlds, all with agent "qa".
	// Without a pin, `findAgentContainer` would return the first match
	// ("matrix") regardless of which world the user meant. With a pin to
	// "the-test", it must return "the-test".
	worlds := []universe.World{
		makeWorld("w-matrix", "cMatrix", "qa"),
		makeWorld("w-terra", "cTerra", "qa"),
		makeWorld("w-eris", "cEris", "qa"),
		makeWorld("w-the-test", "cTheTest", "qa"),
		makeWorld("w-callisto", "cCallisto", "qa"),
	}

	cid, wid, err := routeAgentToWorld(worlds, "qa", "w-the-test", running)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wid != "w-the-test" {
		t.Errorf("expected worldID=w-the-test, got %q", wid)
	}
	if cid != "cTheTest" {
		t.Errorf("expected containerID=cTheTest, got %q", cid)
	}
}

func TestRouteAgentToWorld_UnpinnedReturnsFirstRunning(t *testing.T) {
	// Legacy "unpinned" behavior is still supported for CLI use from outside
	// the UI. Takes the first running world that has the agent.
	worlds := []universe.World{
		makeWorld("w-first", "cA", "qa"),
		makeWorld("w-second", "cB", "qa"),
	}
	cid, wid, err := routeAgentToWorld(worlds, "qa", "", onlyA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wid != "w-first" || cid != "cA" {
		t.Errorf("expected (cA, w-first), got (%q, %q)", cid, wid)
	}
}

func TestRouteAgentToWorld_UnpinnedSkipsStoppedContainers(t *testing.T) {
	worlds := []universe.World{
		makeWorld("w-stopped", "cStopped", "qa"), // skipped — not running
		makeWorld("w-running", "cA", "qa"),
	}
	cid, wid, err := routeAgentToWorld(worlds, "qa", "", onlyA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wid != "w-running" || cid != "cA" {
		t.Errorf("expected to skip stopped world, got (%q, %q)", cid, wid)
	}
}

func TestRouteAgentToWorld_PinnedWorldNotFound(t *testing.T) {
	worlds := []universe.World{makeWorld("w-a", "cA", "qa")}
	_, _, err := routeAgentToWorld(worlds, "qa", "w-missing", running)
	if err == nil || !strings.Contains(err.Error(), "w-missing") {
		t.Errorf("expected 'w-missing' not-found error, got: %v", err)
	}
}

func TestRouteAgentToWorld_PinnedWorldNotRunning(t *testing.T) {
	worlds := []universe.World{makeWorld("w-a", "cA", "qa")}
	_, _, err := routeAgentToWorld(worlds, "qa", "w-a", stopped)
	if err == nil || !strings.Contains(err.Error(), "is not running") {
		t.Errorf("expected 'not running' error, got: %v", err)
	}
}

func TestRouteAgentToWorld_PinnedWorldLacksAgent(t *testing.T) {
	// Pinning a world that doesn't have the requested agent must fail loudly
	// rather than silently falling back to another world.
	worlds := []universe.World{
		makeWorld("w-a", "cA", "neo"),
		makeWorld("w-b", "cB", "qa"),
	}
	_, _, err := routeAgentToWorld(worlds, "qa", "w-a", running)
	if err == nil || !strings.Contains(err.Error(), "does not contain agent") {
		t.Errorf("expected 'does not contain agent' error, got: %v", err)
	}
}

func TestRouteAgentToWorld_FindsAgentInAgentsSlice(t *testing.T) {
	// Multi-agent worlds list citizens in the Agents slice, not just the
	// primary Agent field. Routing must check both.
	worlds := []universe.World{
		{
			ID:          "w-multi",
			ContainerID: "cMulti",
			Agent:       "governor",
			Agents: []universe.AgentRecord{
				{Name: "governor", Tier: "governor"},
				{Name: "qa", Tier: "citizen"},
			},
		},
	}
	cid, wid, err := routeAgentToWorld(worlds, "qa", "w-multi", running)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wid != "w-multi" || cid != "cMulti" {
		t.Errorf("expected to find qa in agents slice, got (%q, %q)", cid, wid)
	}
}

func TestRouteAgentToWorld_AgentNotAnywhere(t *testing.T) {
	worlds := []universe.World{makeWorld("w-a", "cA", "neo")}
	_, _, err := routeAgentToWorld(worlds, "qa", "", running)
	if err == nil || !strings.Contains(err.Error(), "not in any active world") {
		t.Errorf("expected 'not in any active world' error, got: %v", err)
	}
}
