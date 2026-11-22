# ADR-074: A far station answers its hello within 5 seconds, and ssh keeps the link alive

**Status:** Proposed
**Date:** 2026-11-22

## Context

Another Mac is reached by one `ssh <mac> station engine attach`, and the first frame of that connection is `hello`. A QA run joined two Mac homes and made the far side's ssh connect and then say nothing, as a frozen far station or a link that stops passing data after TCP is up does. `station overview --on beta`, `ports --on beta`, `apps --on beta` and `machines beta` never ended: each stayed until `timeout 45` killed it, one until a SIGKILL at 102 s. Its goroutine dump showed the wait in `api.(*Conn).Hello`, which waited on its answer or on its caller's context, and a command's context carries no deadline. `station machines` alone was bounded, by the 5 s each Mac is given there (`each.Budget`).

ssh's own `ConnectTimeout=5` bounds the TCP connect only. A far station that is alive but frozen keeps ssh healthy, so no ssh option alone can catch it.

## Decision

- **The hello is answered within 5 s or the connection ends** (`api.HelloBudget`, beside `api.ConnectBudget`). The budget runs from the hello's request, the far side's connect and start included. When it runs out, the connection is closed and its error is the connection's end, which over ssh is `remote.unreachable`: a read prints `✗ <mac> does not answer` and exits 0, an action `Error: <mac> does not answer` and exits 1, the words a Mac that does not answer already had. 5 s is the budget All machines gives each Mac, so a Mac is silent by one measure in both.
- **Only the hello is bounded.** A call's own answer is not: a long action may send no frame for minutes, and a bound there would cut it. A connection whose hello answered is a far station that spoke.
- **ssh keeps the link alive**: `-o ServerAliveInterval=2 -o ServerAliveCountMax=2`, so a link that stops passing data, a call's included, ends within about 6 s, as a Mac that does not answer.

## Consequences

- A command `--on` a silent Mac ends in about 7 s per connection, the hello's 5 s and the 2 s ssh is given to end once its stdin is closed. A read that words the Mac's last answer asks All machines once more, so it ends in about 14 s.
- A far station that answers its hello and then freezes mid-call still holds that call until its caller lets go. The engine's held streams have their own liveness (45 s without a frame).
- A slow far Mac whose ssh needs more than 5 s to connect, authenticate and start station is told as one that does not answer.
