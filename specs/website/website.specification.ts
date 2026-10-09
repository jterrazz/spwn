import { process, specification } from '@jterrazz/test';
import { afterAll } from 'vitest';

// invariant: the API serves a fresh project from a temporary directory, its spwn home inside it, so no run reads the operator's ~/.spwn
const api = process({
    command:
        'P="$(node website/__setup__/project.ts)" && cd "$P" && SPWN_HOME="$P/spwn-home" exec "$OLDPWD/../.artifacts/go/spwn" web --no-open --port "$PORT"',
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
