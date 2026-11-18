package contracttest

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func uptimeRecord(day int) machines.Uptime {
	at := time.Date(2026, 11, day, 14, 2, 0, 0, time.UTC)
	return machines.Uptime{
		Since:     time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC),
		Days:      []machines.UptimeDay{{Date: at.Format("2006-01-02"), DownS: 420, Restarts: 1}},
		Incidents: []machines.Incident{{From: at, To: at.Add(7 * time.Minute), Cause: "powered off from inside"}},
	}
}

func UptimeStore(t *testing.T, open func(t *testing.T) contract.UptimeStore) {
	ctx := context.Background()

	t.Run("a server never marked has an empty record", func(t *testing.T) {
		s := open(t)
		got, err := s.Get(ctx, "infra")
		if err != nil || !reflect.DeepEqual(got, machines.Uptime{}) {
			t.Fatalf("Get = %+v, %v; want an empty record", got, err)
		}
	})

	t.Run("a record put is got back as it was put", func(t *testing.T) {
		s := open(t)
		want := uptimeRecord(5)
		if err := s.Put(ctx, "infra", want); err != nil {
			t.Fatalf("Put: %v", err)
		}
		got, err := s.Get(ctx, "infra")
		if err != nil || !got.Since.Equal(want.Since) || !reflect.DeepEqual(got.Days, want.Days) || len(got.Incidents) != 1 ||
			!got.Incidents[0].From.Equal(want.Incidents[0].From) || !got.Incidents[0].To.Equal(want.Incidents[0].To) || got.Incidents[0].Cause != want.Incidents[0].Cause {
			t.Fatalf("Get = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("putting a server again replaces its record and keeps the others", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, "infra", uptimeRecord(5)); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Put(ctx, "db", uptimeRecord(6)); err != nil {
			t.Fatalf("Put db: %v", err)
		}
		if err := s.Put(ctx, "infra", uptimeRecord(7)); err != nil {
			t.Fatalf("Put again: %v", err)
		}
		infra, _ := s.Get(ctx, "infra")
		db, _ := s.Get(ctx, "db")
		if len(infra.Days) != 1 || infra.Days[0].Date != "2026-11-07" || len(db.Days) != 1 || db.Days[0].Date != "2026-11-06" {
			t.Errorf("infra = %+v, db = %+v; want infra replaced, db kept", infra.Days, db.Days)
		}
	})

	t.Run("a deleted record is gone, and deleting it again changes nothing", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, "infra", uptimeRecord(5)); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Delete(ctx, "infra"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if got, err := s.Get(ctx, "infra"); err != nil || !got.Since.IsZero() {
			t.Errorf("Get after Delete = %+v, %v; want an empty record", got, err)
		}
		if err := s.Delete(ctx, "infra"); err != nil {
			t.Errorf("Delete again = %v; want nothing", err)
		}
	})
}
