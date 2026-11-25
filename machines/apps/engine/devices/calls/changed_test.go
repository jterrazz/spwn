package calls_test

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/devices/calls"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestAnAddAndAForgetTellTheEngineAtOnceAndARefusalTellsNobody(t *testing.T) {
	told := 0
	d := calls.Deps{
		Store: fake.StoreOf(&world.World{}, "mac-mini"), Sealer: calls.Sealer{Key: newKey(t), Mac: "mac-mini"},
		Presence: func(context.Context, string) devices.Answer { return devices.Given },
		Now:      func() time.Time { return time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC) },
		Changed:  func() { told++ },
	}
	ctx := t.Context()
	if _, err := calls.Add.Run(ctx, d, calls.AddIn{Name: "laptop", Key: peerKey}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := calls.Add.Run(ctx, d, calls.AddIn{Name: "laptop", Key: peerKey}, nil); err == nil {
		t.Fatal("a second add of the same device: want devices.duplicate")
	}
	if _, err := calls.Forget.Run(ctx, d, calls.ForgetIn{Name: "laptop", Confirm: "laptop"}, nil); err != nil {
		t.Fatal(err)
	}
	if told != 2 {
		t.Fatalf("the engine was told %d times; want once for the add and once for the forget", told)
	}
}
