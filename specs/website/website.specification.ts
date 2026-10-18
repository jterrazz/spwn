import { specification } from '@jterrazz/test';
import { process } from '@jterrazz/test/services';
import { afterAll } from 'vitest';

// outside: the sealed HOME hides the per-user socket of OrbStack, Colima or Docker Desktop, so the operator's docker endpoint is read before HOME is replaced and handed over as SPWN_DOCKER_HOST
// invariant: the API serves a fresh project from a temporary directory, sealed from the operator's machine as the cli runner's ground is: its spwn home and HOME inside the project, the providers' tokens unset, no keychain read
const api = process({
    command: [
        'P="$(node website/__setup__/project.ts)" && cd "$P"',
        'unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN CLAUDE_CODE_OAUTH_TOKEN OPENAI_API_KEY',
        'export SPWN_DOCKER_HOST="$(docker context inspect --format \'{{.Endpoints.docker.Host}}\' 2>/dev/null)"',
        'HOME="$P/home" SPWN_HOME="$P/spwn-home" exec "$OLDPWD/../.artifacts/go/spwn" web --no-open --port "$PORT"',
    ].join(' && '),
    env: { SPWN_SKIP_AUTH_VALIDATION: '1', SPWN_SKIP_KEYCHAIN: '1' },
    ready: '/api/health',
});

export const { cleanup, website } = await specification.website({
    // outside: the page calls the API on its own origin, a port of its own, which the default policy would abort as a third party
    external: 'allow',
    server: (services) =>
        process({
            command: 'pnpm exec next dev --port "$PORT"',
            cwd: '../apps/web',
            env: { NEXT_PUBLIC_API_URL: services.api.url },
            ready: '/',
            timeout: 120_000,
        }),
    services: { api },
});

afterAll(cleanup);
