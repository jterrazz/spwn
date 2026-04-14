---
title: "spwn check"
slug: "spwn-check"
---

## spwn check

Validate the project tree against spwn.yaml

### Synopsis

Walks up from the current directory looking for spwn.yaml, then runs
every validation rule against the project. Reports issues grouped by
severity. Exits non-zero when errors are found (or warnings, with
--strict).

```
spwn check [flags]
```

### Options

```
  -h, --help     help for check
      --strict   Exit non-zero on warnings, not just errors
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

