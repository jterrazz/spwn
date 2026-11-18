# ADR-019: Activity is a section, its history the engine's memory

**Status:** Proposed
**Date:** 2026-11-06

Written 2026-11-06 from the owner's round 13 decision M1 and the code that carries it.

## Context

v2 printed a few counters in `station status` and read each of them on its own. The owner asked for monitoring of every machine, Macs and VMs: CPU, memory, swap, disk read and write, network in and out with their totals, load, thermal and battery on a Mac, free space, uptime, the busiest processes, with history and graphs in the TUI and the menu bar and plain values in the CLI.

## Decision

Activity is a section of every kind, the family `activity`, its read `activity.read`. Each fact has one owner: the counters that move are the `Meter` contract's (`macos` over sysctl and IOKit, `linux` over `/proc`), free space `DiskUsage.Volume`'s, battery and thermal `Power.Read`'s, the fixed size `Self`'s. A VM's activity is read by its guest; when the guest does not answer, from its process on its Mac. The engine's `sample` job keeps the history in memory only: a point every 2 s while a surface watches the machine (`engine.watch`), every 60 s otherwise, 24 h kept; the rates are the mean of the last 10 s. With the engine stopped a read takes two samples 1 s apart and says so. The TUI and the menu bar draw sparklines from the series; the CLI prints values, and `--json --history` the 24 h series.

## Consequences

- The history is lost at an engine restart; nothing is written to disk.
- The Overview's Activity line and its findings (disk almost full, memory pressure, swap growing, running hot, battery low) are the same read's.
- Dropped with their reasons (O2): the public IP lookup (an outbound call per read), the VPN list, the workspace scan of `~/Developer`, the gateway rows.
