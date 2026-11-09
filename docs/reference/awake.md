# Awake

`station awake` keeps a Mac from sleeping: every Mac's mode, a keep-awake for a while, or off. It is a section of Macs only; a VM follows its Mac. What a Mac can do is read from the capabilities it reports, never from its model: a battery, a lid, a thermal reading.

| Section      | Answers                                                        |
| ------------ | -------------------------------------------------------------- |
| The modes    | Sleeps normally, Awake until, Never sleeps, Stopped by a guard |
| Keep awake   | `station awake <for>`, what the engine holds                   |
| The guards   | What ends a keep-awake by itself, on which Mac                 |
| Never sleeps | The always-on Mac's policy                                     |

## The modes

```sh
station awake [--json]                    # every Mac, this one first
station awake <for> [--lid] [--battery]   # 30m, 1h, 4h, until 18:00 (1 min to 24 h)
station awake --until-done <vm>           # until the agent working in that VM is done
station awake off                         # end the keep-awake now; Never sleeps stays
```

Every Mac is listed, This Mac first, the others asked through their engines, so a Mac mini's mode is read and changed from the MacBook (`--on mac-mini`). Each is in one mode: `sleeps-normally`, `awake-until` (`Awake until 18:00`), `never-sleeps`, or `stopped-by-guard` (the guard that ended the last keep-awake, kept on the page for 24 hours), with its reading: lid, power source, battery, thermal. Each Mac also names what depends on it, what pauses while it sleeps: the servers it hosts (`hosts_servers`) and the work running there now (`pauses`), a running workspace whose agent has not finished; a stopped workspace, or one whose agent finished, is not listed. A Mac that does not answer is listed under `silent` with why, and as it last answered: `asleep` (it told it was going to sleep, or last answered on battery with its lid closed), `last_answer`, and the servers it then hosted (`hosts_servers`).

## Keep awake

The engine that already runs holds the session, in memory only: a `PreventUserIdleSystemSleep` assertion, and with `--lid` the clamshell sleep disabled (no root, no `pmset -a`, no file). Every end sets the clamshell back to normal, and so does every engine start and quit. A guard ends the session first: thermal pressure at trapping (or heavy for 5 minutes lid closed on battery), the battery at 20 % on battery, Low Power Mode, or the lid closed on battery without `--battery`; each end is logged in `engine.log` and kept on the page for 24 hours. With `--lid`, a lid close sleeps the display with `pmset displaysleepnow`, never a display assertion. A guard already past its threshold refuses the start (exit 1). With `--until-done <vm>` the session follows a run: the engine's `awake` job reads the agents report it keeps of that VM, and the session ends (`awake_ended`, "linux-1 is done") the second its agents leave `working`, or after 24 h at the latest; a VM no agent works in is refused. A switch flipped later keeps the run it follows. Piped, one record per row: the session (`on`, its flags, until in RFC 3339, who asked), the guards, then the reading as `key<TAB>value`. Unrelated to the `restart-after-power-cut` setting, the admin pmset policy of an always-on Mac.

## The guards

A guard exists only on a Mac whose capabilities carry the fact it reads (`awake.Active(caps)`), in this order of precedence:

| Guard       | Needs               | Ends the keep-awake when                                                                       |
| ----------- | ------------------- | ---------------------------------------------------------------------------------------------- |
| `thermal`   | a thermal reading   | thermal pressure reaches trapping, or stays heavy for 5 minutes with the lid closed on battery |
| `battery`   | a battery           | the battery is at 20 % on battery (a warning from 30 %)                                        |
| `low_power` | any Mac             | Low Power Mode turns on                                                                        |
| `lid`       | a lid and a battery | the lid closes on battery without `--battery`                                                  |

So a Mac mini, which reports no battery and no lid, has no battery or lid guard; a MacBook has all four. Each end is an `awake_ended` event: logged in `engine.log`, told by the menu bar.

## Never sleeps

`never-sleeps` is a setting of the Always-on Mac group ([Settings](settings.md#the-items)): applied, it writes `never_sleeps` to [`preferences.json`](preferences.md), and the engine's `awake` job holds the Mac awake for as long as it runs, its servers with it; no admin, applied from any Mac. `station awake off` ends a keep-awake only; Never sleeps stays until `station settings off never-sleeps`. A Mac that hosts servers lists it in its [Mac readiness](machines.md#mac-readiness).

## Related

- [Commands](commands.md#station-awake): the flags.
- [Activity](activity.md): battery and thermal in a machine's activity.
- [Machines](machines.md#mac-readiness): what an always-on Mac must have.
