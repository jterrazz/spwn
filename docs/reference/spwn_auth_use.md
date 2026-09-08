---
title: "spwn auth use"
slug: "spwn-auth-use"
---

## spwn auth use

Pick which credential method spwn should prefer

### Synopsis

Flip the active method for a provider. Run without a method
to clear the preference (back to auto-select).

Example:
  spwn auth use anthropic oauth
  spwn auth use openai api_key

```
spwn auth use <provider> <method> [flags]
```

### Options

```
  -h, --help   help for use
```

### SEE ALSO

* [spwn auth](./spwn_auth.md)	 - Manage credentials — dashboard, login, use, logout, disable

