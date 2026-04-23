---
title: "spwn auth disable"
slug: "spwn-auth-disable"
---

## spwn auth disable

Tell spwn not to use this provider, even if creds exist

### Synopsis

Opt a provider out without touching credentials. Useful when
you want spwn to ignore (say) codex OAuth on your machine but leave
the codex CLI working.

```
spwn auth disable <provider> [flags]
```

### Options

```
  -h, --help   help for disable
```

### SEE ALSO

* [spwn auth](./spwn_auth.md)	 - Manage credentials — status, login, use, logout, disable

