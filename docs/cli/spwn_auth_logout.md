---
title: "spwn auth logout"
slug: "spwn-auth-logout"
---

## spwn auth logout

Clear stored credentials for a provider

### Synopsis

Remove every stored credential for a provider — cache file,
macOS keychain entry, runtime-CLI auth files. Does NOT unset env vars
(the shell owns those); any active env vars are surfaced so you know
to unset them manually.

  spwn auth logout anthropic
  spwn auth logout openai
  spwn auth logout anthropic --method api_key   # spare keychain

```
spwn auth logout <provider> [flags]
```

### Options

```
  -h, --help            help for logout
      --method string   Scope logout to a single method (oauth | api_key)
```

### SEE ALSO

* [spwn auth](./spwn_auth.md)	 - Manage credentials — dashboard, login, use, logout, disable

