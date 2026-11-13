//go:build darwin && cgo

package darwin

import (
	"context"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// value: IOPowerSources posts under com.apple.system, which only the system
// may post, so the wake is proved on a name of this test's own.
func TestAPostedNotificationWakesTheWatchAndCancelStopsIt(t *testing.T) {
	name := "com.jterrazz.station.test." + strconv.Itoa(time.Now().Nanosecond())
	ctx, cancel := context.WithCancel(context.Background())
	wake := notified(ctx, name)
	if err := exec.Command("/usr/bin/notifyutil", "-p", name).Run(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-wake:
	case <-time.After(2 * time.Second):
		t.Fatal("a post of the name never woke the watch")
	}
	cancel()
	time.Sleep(100 * time.Millisecond)
	if err := exec.Command("/usr/bin/notifyutil", "-p", name).Run(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-wake:
		t.Fatal("a post after cancel still woke the watch")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestALaptopsBatteryReadsItsHealthAndCycles(t *testing.T) {
	var p Power
	r := p.Read()
	if !r.HasBattery {
		t.Skip("this Mac has no battery")
	}
	if r.Health < 1 || r.Health > 100 || r.Cycles < 1 {
		t.Errorf("Read() health = %d, cycles = %d; want the AppleSmartBattery's, 1-100 %% and its cycle count", r.Health, r.Cycles)
	}
}
