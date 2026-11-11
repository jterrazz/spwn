# Activity

`station activity` is what a machine is doing now, the Activity section of every kind: Macs, workspaces and servers alike ([ADR-019](../decisions/019-activity.md)).

| Section               | Answers                                           |
| --------------------- | ------------------------------------------------- |
| What it shows         | The measures, the busiest processes, the findings |
| Where each comes from | The one owner of each fact                        |
| The history           | What the engine keeps, how often, for how long    |
| Graphs                | What the menu bar draws                           |

## What it shows

```sh
station activity [--all-machines] [--json [--history]]
```

What a machine is doing now (SPEC §6): CPU of the whole machine and its load, memory used of total with its pressure and swap, free space, disk read and write and network in and out with what the kernel counted since boot, thermal and battery on a Mac, how long it has been up, and the five busiest processes by CPU and by memory (a VM's Virtualization process named by its VM, on every reading of its Mac). A Mac's sample also carries each VM it runs as that process, its CPU a share of the whole Mac ([ADR-023](../decisions/023-a-vms-cpu-is-its-guests-process.md)). The rates are the mean of the last 10 s of the history the engine keeps while a surface watches the machine (its 2 s points; a VM's is its Mac's engine's); with the engine stopped, or unwatched (its points a minute apart), two samples read now, 1 s apart, said `rates over 1 s`, its 24 h series and what must last (a CPU busy for 10 min) still the engine's. Another machine's read starts with `On <machine>:`. A VM whose guest does not answer shows its process on its Mac, `up 12d 0h (since its run started) · from mac-mini (its guest does not answer)`, and no disk or network values; a stopped VM says `○ api is stopped` and exits 0; a Meter that does not answer in 2 s turns every value to `? · no answer in 2 s` and the read still exits 0; one that answers but counted nothing shows `?`, never zeros. What it raises in Needs you (disk almost full, memory pressure critical, swap growing, running hot, battery low, a busy server) heads the page with its fix. `--all-machines` is one row per machine, a Mac that does not answer its own row, and gathers every machine's findings into one Needs you: a finding of any machine but This Mac names it, `jterrazz-mac-mini: disk almost full: 3 % free`, and its fix carries `--on` that machine (in the JSON, each of the top-level `findings` names its `machine`); with `--on` it is refused, exit 2. `--json` prints `station.activity.read/v1`; `--history` adds the engine's 24 h series in 60 s points and needs `--json` (exit 2 without).

## Where each comes from

Each fact has one owner, and Activity reads it there:

| Fact                                                                                                            | Owner                                                                                             |
| --------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| CPU and load, memory used and pressure, swap, disk read and write, network in and out, boot time, the processes | `activity/contract.Meter`: `macos` (sysctl, IOKit block devices, proc_pidinfo), `linux` (`/proc`) |
| free space                                                                                                      | `disk/contract.DiskUsage`'s volume                                                                |
| battery and thermal                                                                                             | `awake/contract.Power`'s reading                                                                  |
| the fixed size (memory, CPUs, model)                                                                            | `core/contract.Self`                                                                              |
| a VM                                                                                                            | its guest's own `activity.read`; when it does not answer, its hypervisor process on its Mac       |

A measure the Meter could not count is `?`, never a zero. A read calls no third-party server (the rule `no-outbound-literals`).

## The history

The engine's `sample` job keeps it, in memory only, for This Mac and every VM it runs: a point every 2 s while a surface watches the machine (the menu bar's, `station activity` itself through `engine.watch`), every 60 s otherwise, 24 hours kept. The rates `station activity` prints are the mean of the last 10 s of it; a CPU busy for 10 minutes, swap growing, and what else must last to be a finding are read from it. An engine restart starts it again from nothing. `--json --history` prints the 24 h series in 60 s points.

## Graphs

The menu bar's machine page draws a machine's Activity over 6 min, an hour or a day: the CPU as a line over 6 min and as bars over the hour and the day, the network as sent over received mirrored, the memory now as a bar; a pointer over a chart reads the value of that point and when it was. The window picked is remembered per machine, an hour until one is picked. What its page carries is the [API](api.md)'s. The CLI prints values only. How they are drawn is the [Design system](../concepts/design-system.md#graphs)'s.

## Related

- [Commands](commands.md#station-activity): the flags.
- [Awake](awake.md): the reading battery and thermal come from.
- [Disk](disk.md): free space, and what takes it.
