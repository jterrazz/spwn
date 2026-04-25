package world

import (
	"sort"
	"strings"
	"testing"

	"spwn.sh/packages/world"
)

// mergeDepPolicy unions per-tool policies across the agents in a
// world. Strategy: deny is the union (any agent denying wins),
// allow is the intersection (must be allowed by every agent that
// opined). Empty policy on either side passes the other through.
//
// These rules matter because a multi-agent world ships ONE image
// — picking "tightest possible" means a watcher with deny:[post-*]
// can't be loosened just because a marketer (in the same world)
// allows everything.

func TestMergeDepPolicy_DenyUnion(t *testing.T) {
	a := world.DepPolicy{Deny: []string{"post-tweet"}}
	b := world.DepPolicy{Deny: []string{"reply-tweet", "post-tweet"}}
	got := mergeDepPolicy(a, b)
	sort.Strings(got.Deny)
	want := []string{"post-tweet", "reply-tweet"}
	if strings.Join(got.Deny, ",") != strings.Join(want, ",") {
		t.Errorf("deny = %v, want %v", got.Deny, want)
	}
}

func TestMergeDepPolicy_AllowIntersection(t *testing.T) {
	a := world.DepPolicy{Allow: []string{"a", "b", "c"}}
	b := world.DepPolicy{Allow: []string{"b", "c", "d"}}
	got := mergeDepPolicy(a, b)
	sort.Strings(got.Allow)
	want := []string{"b", "c"}
	if strings.Join(got.Allow, ",") != strings.Join(want, ",") {
		t.Errorf("allow = %v, want %v", got.Allow, want)
	}
}

func TestMergeDepPolicy_EmptyPassesOtherThrough(t *testing.T) {
	a := world.DepPolicy{}
	b := world.DepPolicy{Deny: []string{"x"}}
	got := mergeDepPolicy(a, b)
	if len(got.Deny) != 1 || got.Deny[0] != "x" {
		t.Errorf("empty + deny: got %v, want [x]", got.Deny)
	}
	got2 := mergeDepPolicy(b, a)
	if len(got2.Deny) != 1 || got2.Deny[0] != "x" {
		t.Errorf("deny + empty: got %v, want [x]", got2.Deny)
	}
}

func TestMergeDepPolicy_OneSideAllowOtherDeny(t *testing.T) {
	// Realistic conflict: marketer allows post-tweet, watcher denies
	// it. The merged policy should keep the deny (strictness wins).
	a := world.DepPolicy{Allow: []string{"post-tweet", "fetch-home"}}
	b := world.DepPolicy{Deny: []string{"post-tweet"}}
	got := mergeDepPolicy(a, b)
	if len(got.Deny) != 1 || got.Deny[0] != "post-tweet" {
		t.Errorf("deny lost when merged with allow: %v", got.Deny)
	}
	// No intersection (b has no Allow), so allow stays empty too.
	if len(got.Allow) != 0 {
		t.Errorf("allow should be empty when only one side declares it: %v", got.Allow)
	}
}
