---
title: "spwn auth login"
slug: "spwn-auth-login"
---

## spwn auth login

Set up credentials for a provider

### Synopsis

Register credentials for a provider. The simplest path is an
API key:

  spwn auth login anthropic --api-key sk-ant-...

For OAuth-backed subscription access (Claude.ai / ChatGPT Plus via codex),
run the upstream CLI login first, then re-run this command — spwn will
detect the new credential and record it:

  claude login   # then: spwn auth login anthropic
  codex login    # then: spwn auth login openai

```
spwn auth login <provider> [flags]
```

### Options

```
      --api-key string   Save an API key for this provider
  -h, --help             help for login
      --oauth            Print OAuth login instructions for this provider
```

### SEE ALSO

* [spwn auth](./spwn_auth.md)	 - Manage credentials — status, login, use, logout, disable

