# ADR-061: The history is kept by the Mac that holds the machine

**Status:** Proposed
**Date:** 2026-11-19

## Context

The window shows what was done on every machine: who made or deleted a VM, what last night's upgrade moved, a gesture the owner just made. The engine kept no such history: a VM record holds its creator and is gone with the VM, the ledger holds the last night alone, notices are live only, and `engine.log` is free text.

Three places to keep a line were weighed. On the machine that runs the call, a VM's history would be lost with the VM, the very moment the owner asks who deleted it, and a stopped VM could not be read. In one store on This Mac, the lines of another Mac's own gestures and nights would never reach it, and two Macs would disagree. Every call already has one Router on the Mac that holds its target, and every fleet-wide read already fans out to each Mac (`apps.matrix`, the images).

## Decision

- **A line is kept by the Mac that holds the machine it concerns**: the Mac itself, or the Mac of a VM, in `{state}/station/journal.jsonl` (`station.journal/v1`), thirty days, the server strip's horizon. A VM keeps none.
- **Two calls of a `journal` family read it.** `journal.kept`, at the target, answers one Mac's raw lines; `journal.read`, in this process, asks each Mac and answers the lines by day, a night's merged across machines and summed, narrowed to a Mac, a machine or a subject, paged by day.
- **A Mac that does not answer, or whose station keeps no journal yet, is an `Unheard` row**, never a refusal of the read.
- **The model is `core/journal`**, pure, so the call kernel may name a line when each gesture comes to declare the lines it keeps; the store has one writer, `journal/calls`.

## Consequences

- A deleted VM's history outlives it, and a stopped VM's reads like a running one's.
- A Mac that is off has no history to read until it answers again; the read says which Mac was not heard.
- The read path lands before any call writes a line: until each gesture declares its lines, a journal holds only what a world seeds. Recording, and who did a gesture, are the next decision's to carry.
- Thirty days per Mac bound the file, and a read that asks more days is clamped to thirty.
