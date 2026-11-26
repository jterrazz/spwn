package devices_test

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/devices"
)

func TestAHeadPastTheLastGenerationIsOneSpwnNeverWrote(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	if _, err := devices.ParseHead(strconv.Itoa(devices.MaxGeneration) + " " + digest); err != nil {
		t.Fatalf("the last generation's head = %v; want it read", err)
	}
	for _, gen := range []int{devices.MaxGeneration + 1, math.MaxInt} {
		if _, err := devices.ParseHead(strconv.Itoa(gen) + " " + digest); err == nil {
			t.Fatalf("a head at generation %d was read; want it refused", gen)
		}
	}
}

func TestAChangeNeverWritesPastTheLastGeneration(t *testing.T) {
	noop := func(*devices.Book) error { return nil }
	for _, gen := range []int{devices.MaxGeneration - 1, devices.MaxGeneration, math.MaxInt} {
		next, err := devices.Next(devices.Book{}, devices.Head{Generation: gen}, noop)
		if err != nil || next.Generation != devices.MaxGeneration {
			t.Fatalf("a change past generation %d writes %d, %v; want %d", gen, next.Generation, err, devices.MaxGeneration)
		}
	}
}
