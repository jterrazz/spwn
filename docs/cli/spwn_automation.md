---
title: "spwn automation"
slug: "spwn-automation"
---

## spwn automation

Trigger-driven agent wakeups (cron + filesystem)

### Synopsis

Automations bind a trigger (cron expression or filesystem watch)
to one of your project's agents. The architect daemon fires them as
events arrive and writes a receipt for every dispatch.

Declare automations under spwn.yaml#worlds.<name>.automations. See
docs/automations.md for the full schema.

### Options

```
  -h, --help   help for automation
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn automation daemon](./spwn_automation_daemon.md)	 - Run the engine for the current project until interrupted
* [spwn automation logs](./spwn_automation_logs.md)	 - Tail .spwn/runs.jsonl receipts
* [spwn automation ls](./spwn_automation_ls.md)	 - List every automation declared in spwn.yaml
* [spwn automation status](./spwn_automation_status.md)	 - Engine state and last-fired snapshot

