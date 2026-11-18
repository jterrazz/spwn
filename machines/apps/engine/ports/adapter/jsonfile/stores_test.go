package jsonfile_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract/contracttest"
)

func disk() jsonfile.Files { return local.New(nil, nil, nil, nil).FS }

func TestReservationStore(t *testing.T) {
	contracttest.ReservationStore(t, func(t *testing.T) contract.ReservationStore {
		return jsonfile.NewReservations(disk(), filepath.Join(t.TempDir(), "leases.json"))
	})
}

func TestDeclarationStore(t *testing.T) {
	contracttest.DeclarationStore(t, func(t *testing.T) contract.DeclarationStore {
		return jsonfile.NewDeclarations(disk(), filepath.Join(t.TempDir(), "public.json"))
	})
}

func TestReservationsReadAsTodaysLeases(t *testing.T) {
	file := filepath.Join(t.TempDir(), "leases.json")
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	if err := jsonfile.NewReservations(disk(), file).Put(context.Background(), ports.Reservation{VM: "api", GuestPort: 3000, MacPort: 40010, ReservedAt: at}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(file)
	want := "{\n  \"schema\": \"station.leases/v1\",\n  \"leases\": [\n    {\n      \"vm\": \"api\",\n      \"guest_port\": 3000,\n      \"host_port\": 40010,\n      \"reserved_at\": \"2026-11-01T09:00:00Z\"\n    }\n  ]\n}\n"
	if string(got) != want {
		t.Errorf("leases.json =\n%s\nwant\n%s", got, want)
	}
}

func TestAReservationSharedToNoAudienceIsUnreadable(t *testing.T) {
	file := filepath.Join(t.TempDir(), "leases.json")
	body := `{"schema": "station.leases/v1", "leases": [{"vm": "api", "guest_port": 3000, "host_port": 40010, "to": "everyone", "reserved_at": "2026-11-01T09:00:00Z"}]}`
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var unreadable *store.UnreadableError
	if _, err := jsonfile.NewReservations(disk(), file).List(context.Background()); !errors.As(err, &unreadable) {
		t.Errorf("list = %v; want the file unreadable for an audience outside the union", err)
	}
	ok := strings.Replace(body, "everyone", "public", 1)
	if err := os.WriteFile(file, []byte(ok), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := jsonfile.NewReservations(disk(), file).List(context.Background()); err != nil || len(got) != 1 || got[0].To != "public" {
		t.Errorf("list = %+v, %v; want the public share read", got, err)
	}
}
