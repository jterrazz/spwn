# packages/auth

Credential resolution and container-side auth injection.

## Role

Resolves host-side AI-provider credentials (Anthropic, OpenAI, Google) from their canonical locations — env vars, `~/.claude/.credentials.json`, `~/.codex/auth.json`, `~/.config/google-genai/credentials` — into a uniform `Credential` value. Then syncs them into the bind-mountable `/credentials/` directory that every spwn container reads at startup, so the user never copies API keys into worlds by hand. Keeps host and container auth surfaces decoupled: runtimes only ever see `/credentials/`.

## Key types

- `Provider` — enum of supported providers (`anthropic`, `openai`, `google`).
- `Credential` — resolved value: provider, raw token/OAuth JSON, credential type, source location.
- `CredentialType` — how the credential was obtained (`api_key`, `oauth`, `cached`, …).
- `SyncCredentials()` — walk every provider, resolve, write the current state into `platform.CredentialsDir()`. Called before every docker exec; idempotent.
- `DockerEnvArgs` / `DockerEnvVars` — translate resolved credentials into `-e` flags for the Docker CLI and `Env` slices for the Docker API.

## Related

- **Imported by** — `apps/api`, `apps/cli`, `packages/architect`
- **Imports** — `packages/platform` (for `~/.spwn/credentials/` and provider-cache paths)
