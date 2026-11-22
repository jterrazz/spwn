# ADR-068: In the fake world no result waits on real time

**Status:** Proposed
**Date:** 2026-11-21

Amends [ADR-020](020-the-fake-world.md): how the world's clock leaps.

## Context

The spec build's clock leaps the world's time instead of waiting. Each leap first spent a moment of real time, `runtime.Gosched()` then `time.Sleep(min(d/50, 5ms))`, so that what the leap should see had run. Measured on 2026-11-21: with the pause removed, 10 stories turn red, among them the load-curve story (`usecases/activity/read/a-load-curve-moves-with-the-world-clock`, 37 red runs in two days). The pause decided their result, and a loaded gate stretched what it allowed. In the load-curve story, the race was whether the feed goroutine a leap woke reached its dial of the other Mac before the command's close began that Mac's engine itself: the close took 3 more ticks, and the second read's window moved. The pause was also 54 % of a two-macs command.

## Decision

A leap is a handshake with what runs, in three steps. First it waits until nothing but leapers can run, so what already runs takes its turn. Then it moves the time and wakes the waiters parked on the clock, counted. Last, it goes on once each of them is back and nothing but leapers can run again. A waiter is back when it parks on the clock again or, its wait over, when its goroutine next calls the clock (`Sleep`, and every other wait the clock offers). "Nothing but leapers can run" is read from the scheduler's own counts (`runtime/metrics`, running and runnable goroutines), twice with a turn of the scheduler between. A waiter still out after 4 such readings in a row waits on something other than the clock, or has ended (a budget's timer, a finished backoff), and is let go. No real pause decides what a leap sees: past 8 busy readings in a row, a leap sleeps between two readings, 20 µs doubling to 1 ms, out of the count of leapers, so what it waits for gets the CPU it would spin on; only the readings end the wait.

## Consequences

- What a leap sees no longer depends on the load: the 10 stories pass with the pause gone, and a leap that does not wait turns them red again.
- A goroutine woken by a leap that waits on real time, or on a real file descriptor, is let go as idle, as it was let go after 5 ms before: the world still waits on real time in `greetBound`, the world lock's wait, a held call's look at the world file, and the 1 ms of a wait while surfaces are let go. Those are not this decision's.
- A leap that only spun read the scheduler's counts, which lock the scheduler, for as long as what it waited for ran: on a loaded VM those reads were a quarter of a command's CPU. Ten world advances of the history-rings story beside 48 busy processes (load about 60, 2026-11-22) took 17.6 to 22.2 s and 9.0 to 9.5 s of CPU with the spin, 8.1 to 9.2 s and 4.1 to 4.3 s with the pause, the world written alike.
- A leap that would wait for ever (a goroutine that spins without the clock) holds the story to its 30 s bound, with its goroutine dump, instead of passing as a flake.
- The spec build alone reads the scheduler's counts; the release build's clock is untouched.
- Three of the window's recordings move with it: the `overview.read` with `"Elsewhere": true` of the worlds `stories`, `stories-file` and `working-hard` now reads the CPU and the network rate beside the memory.
