# ADR-107: Activity is sampled by who looks, kept in tiers of mean and max for 30 days, and streamed as points

**Status:** Proposed
**Date:** 2026-11-26

Supersedes in part [ADR-019](019-activity.md): its sentence on how often the engine samples and how long it keeps the history.

## Context

The owner asked to rework the timing, the refresh rate and the time window of Activity, "to feel alive, to be useful" (2026-11-26). Read from the tree of that day, the pipeline answered neither:

- **It does not move.** The engine samples a watched machine every 2 s, every 60 s otherwise, but `activity.read` sends one report and closes, and the window asks again only on an `engine.events` notice ([ADR-083](083-a-re-ask-is-quiet-only-for-an-engine-kept-view-never-by-a-timer.md)): an open Activity tab is the snapshot taken when it opened.
- **It keeps instants.** Two rings of whole samples, 180 × 2 s (6 min) and 1440 × 1 min (24 h), in memory only. A gauge's spike between two minute points is lost, and the window's memory, disk and network are drawn from the 24 h ring even on the 6 min window: 7 identical bars for 6 minutes.
- **It erases peaks.** The 24 h CPU bar is a 30 min mean, so a 3 min build at 100 % reads 10 %; the other measures keep one point in 20.
- **It re-sends everything.** Each answer carries the whole series, up to 580 KB at 24 h, and the menu bar's page is sent whole every 2 s.
- **It forgets.** Every engine restart, so every update, starts the 24 h again.

## Decision

### Sampling follows who looks

- A machine shown in a visible window, or in the menu bar's open Activity, is sampled every **1 s**, its busiest processes every 5 s. A VM stays at 2 s on the guest connection of today, which starts two guest processes per sample, until its samples ride the resident guest channel.
- Nobody looking: This Mac every 10 s (in-process, no spawn), a VM every 60 s until the guest channel, then 10 s. The idle engine's ceiling of process starts is unchanged.
- An act running on a machine (create, apply, start, the night upgrade) is sampled every 2 s for its duration and 30 s after, looked at or not, so the spike it causes has a shape the next morning.
- A window releases its watch 10 s after it is hidden or the tab is left.

### The history is four tiers of buckets, each mean and max

| Tier | Step   | Kept   |
| ---- | ------ | ------ |
| T0   | 1 s    | 10 min |
| T1   | 10 s   | 3 h    |
| T2   | 1 min  | 48 h   |
| T3   | 15 min | 30 d   |

- Each bucket keeps, for every measure (CPU share, memory used, swap used, disk read and write, network in and out), the **mean and the max** of what was measured in it, the samples it counts and the time its rates span. A rate is measured between two samples, from the kernel's counters, so its mean is exact at any step; a gauge is the mean of its readings. The max of every tier is the max of the samples, so a 1 s spike survives to the 30 days.
- Every sample rolls into all four tiers at once, in the slot of its own time: the clock of the machine that measured it, so a VM's slots follow its guest's clock. Each tier is a fixed ring of slots indexed by that time, so its size is bounded and a slot nothing was measured in is a gap, never a zero.
- A point a reader draws is one bucket, built from that bucket alone: its rates' means over the time they span, its gauges, its samples. A pause in the sampling, or a counter that went back (a reboot), is a gap at the bucket of the first sample after it, and the points before it stay.
- The 1 min tier alone also keeps its newest sample's load, free disk space and battery, for the 24 h series a script reads; no other tier's reader needs them.
- T2 and T3 survive restarts and updates, in fixed-record ring files written once per 15 min and at a clean stop: 30 days is the journal's horizon, so every event it keeps has its curves.
- Min is not kept: for load the floor is rarely the question.

### Points are streamed, not series

- A new watch, `activity.stream {machine, range}`, opens with the range's buckets from the coarsest tier that gives at least one bucket per 2 px (at most 400), then sends one point per sample and one event per journal line or notice of that machine. Its close releases the watch.
- `activity.read` stays one-shot for the CLI and the menu bar; it loses its whole series in favour of the stream's range.

### Windows and motion

- **Ranges:** 5 min (live, the default), 1 h, 24 h, 7 d, 30 d, each ending at now and moving; a range brushed on a chart is still until "Back to live".
- **The chart slides:** a canvas per card, drawn 1 s behind real time so its newest segment joins two known points; a band from mean to max under the mean line; a shared crosshair reads all four at one instant; the journal's lines and the notices are marks on the time axis.
- **Truthful motion:** a leading dot beats at 1 Hz only while samples arrive, and turns to a hollow ring with the last sample's time when they stop; the figure is a 3 s mean redrawn at 1 Hz, never tweened; a hidden window paints nothing; reduced motion draws once per point.
- **Neutral ink** for every curve. Blue only for the band of an act in progress, orange only when a finding asks for the owner; a high value is never coloured by itself. Disk and network sit on a fixed log-like scale, CPU and memory on their machine's limits.

### Refused

Sub-second sampling; tweened or rolling figures; motion at idle that no sample caused; an unbounded history; a React render per sample; a curve coloured by its value; a series re-sent per update.

## What the first lot ships

The decision lands in lots. The first ships the four tiers and nothing else of it:

- The history is the four tiers, each bucket the mean and max of every measure, the 1 min tier with its gauges; every number an answer writes is rounded to what it measures; `activity.read` asks its engine, through `engine.activity`, for one machine and only the part of its tiers it reads.
- The cadence is unchanged: 2 s while a surface watches the machine, 60 s otherwise.
- Nothing is persisted: an engine restart starts the history again from nothing.
- No stream: `activity.read` stays one-shot, its series served from the tiers.
- What a reader sees change against the two rings of whole samples kept before: a bar ends on the first boundary of its span at or after now, not at now; its mean is weighted by the time each sample spans; its `points` counts samples, not intervals; its busiest is the busiest its tier kept, a 1 s spike included; the 24 h graph is 97 points of 15 min, not 1440 of 1 min; a point is a bucket's rates, gauges and samples, no counter since boot; and the menu bar folds its charts with the window's fold, so both draw the same bars, but for a bar that holds a gap, whose samples the window's count includes and the menu bar's leaves out.

The cadence that follows who looks, the ring files and `activity.stream` are later lots.

## Consequences

- The numbers the owner may tune after the prototype (the default window, the cadences, the tiers' steps and horizons) are each one named constant.
- A machine's history costs about 0.74 MB of the engine's memory (737,280 B measured), 7.4 MB for ten machines: a bucket is 88 B, its memory and swap whole bytes, its shares and rates float32, which an answer rounds to what it measures, and each slot of the 1 min tier 24 B more of gauges; the files write about 1 KB per machine per 15 min.
- `activity.read`'s report keeps its whole series only until the stream lands; meanwhile it is served from the tiers and carries each measure's mean and max over the window asked. It asks the engine for one machine and only the buckets its window, its 24 h and its findings read, about 19 KB for a day of a full 30 days, whatever the fleet.
- The findings that must last (memory pressure held, swap growing, a server's CPU busy) are judged on the tiers' buckets.
- The cost of sampling at 1 s on a Mac is estimated, not measured; a measure on the acceptance Mac sets its budget.
