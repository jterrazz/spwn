---
title: "spwn auth default"
slug: "spwn-auth-default"
---

## spwn auth default

Pick which provider spwn prefers when multiple are authenticated

### Synopsis

Set a soft preference for which provider's runtime spwn picks
when you're logged into more than one and no runtime is pinned at the
project or agent level.

This is the durable answer to the "multiple providers authenticated
and no runtime pinned" error — set it once and spwn will quietly
resolve ambiguity in that provider's favour. Does NOT disable the
other provider or override agent.yaml / spwn.yaml pins.

Example:
  spwn auth default anthropic        # prefer claude-code
  spwn auth default --clear          # revert to auto-resolve

```
spwn auth default [provider] [flags]
```

### Options

```
      --clear   Remove the default preference (revert to auto-resolve)
  -h, --help    help for default
```

### SEE ALSO

* [spwn auth](./spwn_auth.md)	 - Manage credentials — dashboard, login, use, logout, disable

