# Awake

`station awake` keeps a Mac from sleeping: every Mac's mode, a keep-awake for a while, or off. It is a section of Macs only; a VM follows its Mac. What a Mac can do is read from the capabilities it reports, never from its model: a battery, a lid, a thermal reading.

| Section      | Answers                                                                     |
| ------------ | --------------------------------------------------------------------------- |
| The modes    | Sleeps normally, Awake until, Never sleeps, Stopped by a guard, the verdict |
| Keep awake   | `station awake <for>`, what the engine holds                                |
| The guards   | What ends a keep-awake by itself, on which Mac                              |
| Never sleeps | The always-on Mac's policy                                                  |

## The modes

```sh
station awake [--json]                    # every Mac, this one first
station awake <for> [--lid] [--battery]   # 30m, 1h, 4h, until 18:00 (1 min to 24 h)
station awake off                         # end the keep-awake now; Never sleeps stays
```

Every Mac is listed, This Mac first, the others asked through their engines, so a Mac mini's mode is read and changed from the MacBook (`--on mac-mini`). Each is in one mode: `sleeps-normally`, `awake-until` (`Awake until 18:00`), `never-sleeps`, or `stopped-by-guard` (the guard that ended the last keep-awake, kept on the page for 24 hours), with its reading: lid, power source, battery, thermal. Each Mac also names what depends on it, what pauses while it sleeps: the servers it hosts (`hosts_servers`) and the work running there now (`pauses`), every running VM that is not a server; a stopped one is not listed. A Mac that does not answer is listed under `silent` with why, and as it last answered: `asleep` (it told it was going to sleep, or last answered on battery with its lid closed), `last_answer`, and the servers it then hosted (`hosts_servers`).

A client that draws Keep awake for… takes its terms from `awake.read`: its `Choices`, each a `Label` and either `For`, a length in nanoseconds, or `Until`, the moment it ends as RFC 3339, each a value `awake.keep` takes as it stands. They are `1 hr`, `4 hr` and `Until 18:00`, the next 18:00 of the Mac's own zone, tomorrow's once today's has passed; the menu bar's Until opens on that same hour. While a keep-awake holds the Mac, one choice is `Picked`: the length it was asked for, to the minute, else Until, which then says the hour it holds until (`Until 17:00`). Never sleeps picks none. Every term is a time: no keep-awake ends with a VM's work, which no VM can say the end of (a quiet VM is as often paused as done), and a Mac that sleeps mid-run costs more than one kept awake too long. What holds a Mac awake with no end is Never sleeps; `station awake off` ends any keep-awake.

### The verdict

`awake.read` says how the Mac stands awake in one `Verdict` ([the API](api.md)): the first case that matches, its `Sentence` the Mac's `Awake` words, capitalised. A client draws it as given and writes no sentence of its own.

| Case                                                 | Tone       | Sentence                                         | Line                                         |
| ---------------------------------------------------- | ---------- | ------------------------------------------------ | -------------------------------------------- |
| the engine reads (`Engine` is `pending`)             | no verdict |                                                  |                                              |
| the engine is stopped or not installed               | `plain`    | `Sleeps normally.`                               | `the engine is stopped`                      |
| `never-sleeps`, let go while the battery is low      | `warn`     | `Never sleeps, let go while the battery is low.` | `the always-on Mac policy`                   |
| `never-sleeps`                                       | `ok`       | `Never sleeps.`                                  | `the always-on Mac policy`                   |
| `awake-until`, on battery at 21 to 30 %              | `warn`     | `Awake until 18:00.`                             | `battery 25 %, it stops at 20 %`             |
| `awake-until`                                        | `ok`       | `Awake until 18:00, lid closed.`                 |                                              |
| `stopped-by-guard`                                   | `warn`     | `Awake stopped, battery 18 %.`                   |                                              |
| `sleeps-normally`, a guard would refuse a keep-awake | `warn`     | `Sleeps normally.`                               | `cannot be kept awake: Low Power Mode is on` |
| `sleeps-normally`                                    | `plain`    | `Sleeps normally.`                               |                                              |

The menu bar marks another Mac's card by the same tone: `ok` OK, `warn` Attention, `plain` Absent.

## Keep awake

The engine that already runs holds the session, in memory only: a `PreventUserIdleSystemSleep` assertion, and with `--lid` the clamshell sleep disabled (no root, no `pmset -a`, no file). Every end sets the clamshell back to normal, and so does every engine start and quit. A guard ends the session first: thermal pressure at trapping (or heavy for 5 minutes lid closed on battery), the battery at 20 % on battery, Low Power Mode, or the lid closed on battery without `--battery`; each end is logged in `engine.log` and kept on the page for 24 hours. With `--lid`, a lid close sleeps the display with `pmset displaysleepnow`, never a display assertion. A guard already past its threshold refuses the start (exit 1). Piped, one record per row: the session (`on`, its flags, until in RFC 3339, who asked), the guards, then the reading as `key<TAB>value`. Unrelated to the `restart-after-power-cut` setting, the admin pmset policy of an always-on Mac.

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

`never-sleeps` is a setting of the Always-on Mac group ([Settings](settings.md#the-packages)): applied, it writes `never_sleeps` to [`preferences.json`](preferences.md), and the engine's `awake` job holds the Mac awake for as long as it runs, its servers with it; no admin, applied from any Mac. `station awake off` ends a keep-awake only; Never sleeps stays until `station settings off never-sleeps`. Its [Mac readiness](machines.md#mac-readiness) lists it, and reads a Mac whose own sleep is off in System Settings (`pmset` sleep 0) as never sleeping too, set in System Settings, with the setting not applied.

## Related

- [Commands](commands.md#station-awake): the flags.
- [Activity](activity.md): battery and thermal in a machine's activity.
- [Machines](machines.md#mac-readiness): what an always-on Mac must have.
