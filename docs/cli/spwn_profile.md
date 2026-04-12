---
title: "spwn profile"
slug: "spwn-profile"
---

## spwn profile

Author and manage reusable profile templates (personality)

### Synopsis

Profiles are reusable personality templates — role, tone, purpose, behavior.
Each profile is a markdown file that agents inherit as their personality baseline.

Profiles live in ~/.spwn/profiles/ and attach to agents via
"spwn agent add <name> --profile <profile-name>".

A profile defines WHO the agent is (not WHAT it can do — that's tools and skills).

### Options

```
  -h, --help   help for profile
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn — create realities for things that can think
* [spwn profile edit](./spwn_profile_edit.md)	 - Open a profile template in $EDITOR
* [spwn profile install](./spwn_profile_install.md)	 - Install a profile template from the registry
* [spwn profile ls](./spwn_profile_ls.md)	 - List profile templates
* [spwn profile new](./spwn_profile_new.md)	 - Author a new profile template
* [spwn profile publish](./spwn_profile_publish.md)	 - Publish a profile template to the registry
* [spwn profile rm](./spwn_profile_rm.md)	 - Delete a profile template
* [spwn profile show](./spwn_profile_show.md)	 - Display a profile template

