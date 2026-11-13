package apps_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestAnAppPresentButNotHeldByItsMethodIsNotManagedNeverOK(t *testing.T) {
	appStore := apps.Report{App: catalog.App{Name: "dia"}, Method: catalog.MethodCask, State: catalog.StateOK, Detail: "app-store"}
	if got := apps.StateOf(appStore); got != apps.NotManaged {
		t.Errorf("state %s", got)
	}
	held := appStore
	held.Managed = true
	if got := apps.StateOf(held); got != apps.OK {
		t.Errorf("held by its method: state %s", got)
	}
}
