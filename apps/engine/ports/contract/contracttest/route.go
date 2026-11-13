package contracttest

import (
	"context"
	"regexp"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

type RouteScene struct {
	// value: a route under test, with a machine where its evidence is
	// there for the declaration and one where it is not.
	Route        contract.PublicRoute
	Declared     ports.Declaration
	Seen, Unseen ports.Machine
}

func PublicRoute(t *testing.T, open func(t *testing.T) RouteScene) {
	t.Run("the route is named by one lowercase word for --via", func(t *testing.T) {
		if name := open(t).Route.Name(); !regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`).MatchString(name) {
			t.Fatalf("Name = %q, want a lowercase word", name)
		}
	})
	t.Run("a port whose evidence is there is seen", func(t *testing.T) {
		s := open(t)
		got, err := s.Route.Verify(context.Background(), s.Declared, s.Seen)
		if err != nil || !got.Seen {
			t.Fatalf("Verify = %+v, %v, want seen", got, err)
		}
	})
	t.Run("a port whose evidence is not there is not seen, says what is missing, and is no error", func(t *testing.T) {
		s := open(t)
		got, err := s.Route.Verify(context.Background(), s.Declared, s.Unseen)
		if err != nil || got.Seen || got.Missing == "" {
			t.Fatalf("Verify = %+v, %v, want not seen with what is missing", got, err)
		}
	})
}
