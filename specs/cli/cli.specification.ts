import { specification } from '@jterrazz/test';
import { resolve } from 'node:path';
import { afterAll } from 'vitest';

// checker-disable-next-line named-path-exists -- the product is the Go binary `make build` compiles into the artefact root, which git ignores; the suite's turbo task builds it first
const SPWN_BIN = resolve(import.meta.dirname, '../../.artifacts/go/spwn');

export const { cli, cleanup } = await specification.cli(SPWN_BIN, {
    commands: [
        'agent',
        'architect',
        'auth',
        'automation',
        'build',
        'check',
        'cookie-sync',
        'create',
        'daemon',
        'default',
        'deploy',
        'destroy',
        'disable',
        'down',
        'dream',
        'edit',
        'enable',
        'enter',
        'export',
        'fork',
        'gate',
        'import',
        'init',
        'inspect',
        'install',
        'knowledge',
        'list',
        'login',
        'logout',
        'logs',
        'ls',
        'new',
        'providers',
        'rename',
        'restart',
        'restore',
        'rm',
        'save',
        'show',
        'skill',
        'sleep',
        'snap',
        'start',
        'status',
        'stop',
        'talk',
        'uninstall',
        'up',
        'upgrade',
        'use',
        'world',
    ],
    // invariant: no run reads or writes the operator's spwn home, keychain or provider tokens; live validation, which calls the providers' APIs from every spawn, is skipped too, and a spec that needs it clears the variable
    defaults: {
        ANTHROPIC_API_KEY: null,
        ANTHROPIC_AUTH_TOKEN: null,
        CLAUDE_CODE_OAUTH_TOKEN: null,
        OPENAI_API_KEY: null,
        SPWN_HOME: '$WORKDIR/spwn-home',
        SPWN_SKIP_AUTH_VALIDATION: '1',
        SPWN_SKIP_KEYCHAIN: '1',
    },
    docker: {
        envVar: 'SPWN_TEST_LABEL',
        nameLabel: 'sh.spwn.world.config',
        testRunLabel: 'sh.spwn.test.run',
    },
    env: {
        isolated: { SPWN_HOME: '$WORKDIR/spwn-home' },
        sealed: {
            HOME: '$WORKDIR/empty-home',
            SPWN_HOME: '$WORKDIR/spwn-home',
            SPWN_SKIP_KEYCHAIN: '1',
        },
    },
});

afterAll(cleanup);
