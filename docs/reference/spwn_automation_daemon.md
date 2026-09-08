---
title: "spwn automation daemon"
slug: "spwn-automation-daemon"
---

## spwn automation daemon

Run the engine for the current project until interrupted

### Synopsis

Loads the project's automations from spwn.yaml, registers them
with the engine, and blocks. Triggers fire as configured (cron
expressions evaluated against the host clock; filesystem watches via
fsnotify). Each fire writes a receipt to .spwn/runs.jsonl.

Stop with Ctrl-C — the engine drains in-flight dispatches before
exiting.

Catch-up: on startup, every cron automation that fired before is
checked for missed slots. With catchup: collapse (the default), one
fire is dispatched on resume regardless of how many slots elapsed
during downtime, with the missed count exposed to the prompt
template via {{ .Missed }}. catchup: skip drops missed slots and
resumes at the next scheduled time.

```
spwn automation daemon [flags]
```

### Options

```
  -h, --help   help for daemon
```

### SEE ALSO

* [spwn automation](./spwn_automation.md)	 - Trigger-driven agent wakeups (cron + filesystem)

