package api

import (
	"net/http"
	"testing"

	"spwn.sh/packages/architect"
	"spwn.sh/packages/world/runtimestate"
)

func TestCreateWorldAnswers400ForAWorkspaceNameThatIsNotASlug(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	t.Setenv("SPWN_SKIP_AUTH_VALIDATION", "1")
	store, err := runtimestate.NewStoreWith(noContainersBackend{}, t.TempDir())
	if err != nil {
		t.Fatalf("runtimestate: %v", err)
	}
	srv := New(store, architect.New(noContainersBackend{}, store), "127.0.0.1:0")

	w := doJSON(t, srv.Handler(), "POST", "/api/worlds", map[string]any{
		"workspaces": []map[string]any{{"name": "my ws", "path": t.TempDir()}},
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400: %s", w.Code, w.Body.String())
	}
}
