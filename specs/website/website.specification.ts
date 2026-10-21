import { specification } from '@jterrazz/test';
import { process } from '@jterrazz/test/services';
import { afterAll, inject } from 'vitest';

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
    port: inject('spwnApiPort'),
    ready: '/api/health',
});

// invariant: the UI is the run's one production build (`__setup__/ui.global-setup.ts`), and each file brings the API it reads on the address that build was given
export const { cleanup, website } = await specification.website({
    services: { api },
    url: inject('spwnUiUrl'),
});

afterAll(cleanup);
