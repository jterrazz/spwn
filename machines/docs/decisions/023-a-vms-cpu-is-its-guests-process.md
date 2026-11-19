# ADR-023: A VM's CPU on its Mac is the process that holds its guest's disk

**Status:** Accepted
**Date:** 2026-11-08

## Context

Virtualization.framework runs each guest in an XPC service, `com.apple.Virtualization.VirtualMachine`, a child of launchd, not of `tart run`: the run's own pid reads near zero. SPEC B5.3 proposed finding that service by its responsible process (`responsibility_get_pid_responsible_for_pid`, the libsystem call Activity Monitor groups by), expected to be the run's pid. Measured on a Mac with a tart VM (4 cores) on 2026-11-08: the service's responsible process is the app that launched `tart`, the same one `tart` itself answers, never `tart`'s pid, so it names no VM when two run.

## Decision

The service is the one that holds the VM's disk image open (`vms/<vm>/disk.img`), as the tart adapter already finds it (kernel tables read with `proc_pidinfo`, no process started), kept per run. A Mac's `activity.sample` reads each running VM as that process (`vms`), its CPU a share of the whole Mac, and the live key `cpuh:<vm>` is that share, read on the VM's Mac. `cpu:<vm>` stays the guest's own counters, its share of its own cores.

## Consequences

- The same measure, with two busy loops in that VM: its guest counted 49.3 % of its 4 cores, the service 48.1 % (2.5 % apart), 19.2 % of the Mac's 10. The guest-folded fallback of SPEC B5.3 is not needed.
- A VM whose service has not opened its disk yet (booting) has no `cpuh`, never a zero.
