//go:build world

package wire

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/world"
)

func leapingClock() worldClock {
	w := &world.World{ThisMac: "here", Now: time.Unix(1000, 0), Macs: map[string]*world.Mac{"here": {}}}
	return (&worldRun{w: w, macs: map[string]*worldMac{}}).clock()
}

func parkedOn(c worldClock, n int) {
	for {
		c.moved.mu.Lock()
		parked := c.moved.parked
		c.moved.mu.Unlock()
		if parked == n {
			return
		}
		runtime.Gosched()
	}
}

// value: work a woken waiter does between two waits, through another
// goroutine, so it takes many turns of the scheduler and no real time.
func hops(n int) {
	ping, pong := make(chan int), make(chan int)
	go func() {
		for v := range ping {
			pong <- v + 1
		}
	}()
	for i := range n {
		ping <- i
		<-pong
	}
	close(ping)
}

func TestWorldMacClockLeapGoesOnOnceTheWaitersItWokeAreBackOnTheClock(t *testing.T) {
	c := leapingClock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiting := context.WithValue(ctx, waits{}, true)
	rounds := make(chan int, 16)
	go func() {
		for n := 1; c.Sleep(waiting, time.Second) == nil; n++ {
			hops(2000)
			rounds <- n
		}
	}()
	far := make(chan struct{})
	go func() {
		_ = c.Sleep(waiting, 3*time.Second)
		hops(2000)
		close(far)
	}()
	parkedOn(c, 2)
	for leap := 1; leap <= 5; leap++ {
		_ = c.Sleep(ctx, time.Second)
		select {
		case n := <-rounds:
			if n != leap {
				t.Fatalf("leap %d saw the waiter's round %d", leap, n)
			}
		default:
			t.Fatalf("leap %d went on before the waiter it woke had done its round and waited again", leap)
		}
		if leap == 3 {
			select {
			case <-far:
			default:
				t.Fatal("the leap that ended a longer wait went on before its waiter was done")
			}
		}
	}
}

func TestWorldMacClockLeapLetsGoOfAWaiterThatLeavesTheClock(t *testing.T) {
	c := leapingClock()
	bound, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	waiting := context.WithValue(bound, waits{}, true)
	release := make(chan struct{})
	left := make(chan struct{})
	go func() {
		_ = c.Sleep(waiting, time.Second)
		close(left)
		<-release
	}()
	parkedOn(c, 1)
	_ = c.Sleep(bound, time.Second)
	if bound.Err() != nil {
		t.Fatal("a leap waited for a waiter that left the clock, until the bound")
	}
	<-left
	close(release)
	again := make(chan bool, 1)
	go func() {
		_ = c.Sleep(waiting, time.Second)
		again <- true
	}()
	parkedOn(c, 1)
	_ = c.Sleep(bound, time.Second)
	if bound.Err() != nil {
		t.Fatal("a turn kept by a waiter that left the clock held the next leap, until the bound")
	}
	select {
	case <-again:
	case <-bound.Done():
		t.Fatal("the next waiter was never woken")
	}
}

func TestWorldMacClockLeapLetsWhatAlreadyRunsTakeItsTurnFirst(t *testing.T) {
	c := leapingClock()
	began := c.Now()
	seen := make(chan time.Time, 1)
	go func() {
		hops(2000)
		seen <- c.Now()
	}()
	_ = c.Sleep(context.Background(), time.Second)
	select {
	case at := <-seen:
		if !at.Equal(began) {
			t.Fatalf("work that ran before the leap read the clock at %v, want %v", at, began)
		}
	default:
		t.Fatal("the clock leapt before the work that already ran had its turn")
	}
}

func TestWorldMacClockAwaitsGiveBackTheTurnTheWaiterKept(t *testing.T) {
	ready := make(chan struct{})
	close(ready)
	waits := map[string]func(c worldClock) error{
		"Await": func(c worldClock) error {
			return c.Await(context.Background(), ready, time.Second)
		},
		"AwaitWhile": func(c worldClock) error {
			return c.AwaitWhile(context.Background(), ready, nil, time.Second)
		},
	}
	for name, wait := range waits {
		t.Run(name, func(t *testing.T) {
			c := leapingClock()
			c.moved.mu.Lock()
			c.moved.out = 1
			c.moved.held[goroutine()] = true
			c.moved.mu.Unlock()
			if err := wait(c); err != nil {
				t.Fatalf("the wait failed: %v", err)
			}
			c.moved.mu.Lock()
			defer c.moved.mu.Unlock()
			if c.moved.out != 0 || len(c.moved.held) != 0 {
				t.Fatalf("after the wait, out is %d and %d turns are held, want the turn given back", c.moved.out, len(c.moved.held))
			}
		})
	}
}
