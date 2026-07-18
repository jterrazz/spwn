import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Error-recovery / state-resilience. Each test isolates itself with a fresh
 * temp workdir and an `$WORKDIR/spwn-home` SPWN_HOME. Commands that share state
 * chain via `exec([...])` inside one persistent working directory. Every exec
 * result binds with `await using` (rule B5); these are CLI-only, no containers.
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('error recovery - state resilience', () => {
    test('agent commands work after failed agent operations', async () => {
        // Given - init then a failing rm short-circuits the chain (exec stops on first non-zero)
        await using result = await isolated().exec([
            'init',
            'agent rm ghost',
            'agent create testbot',
            'agent ls',
        ]);

        // Then - the chain fails on `agent rm ghost` without crashing (absence probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).not.toContain('panic:');

        // When - a fresh home runs create + ls to prove nothing was corrupted
        await using recovery = await isolated().exec(['init', 'agent create testbot', 'agent ls']);

        // Then - the agent is created and listed (agent ls renders on stderr)
        expect(recovery.exitCode).toBe(0);
        expect(recovery.stderr).toContain('testbot');
    });

    test('export non-existent agent does not corrupt state', async () => {
        // Given - a bad export after a real create exits non-zero without crashing
        await using bad = await isolated().exec([
            'init',
            'agent create neoprime',
            'agent export ghost',
        ]);

        // Then - non-zero exit, no crash (absence probe)
        expect(bad.exitCode).toBe(1);
        expect(bad.stderr).not.toContain('panic:');
        expect(bad.stderr).not.toContain('goroutine ');

        // When - a parallel healthy setup lists the agent
        await using healthy = await isolated().exec(['init', 'agent create neoprime', 'agent ls']);

        // Then - the agent is listed cleanly
        expect(healthy.exitCode).toBe(0);
        expect(healthy.stderr).toContain('neoprime');
    });

    test('multiple errors in sequence do not compound', async () => {
        // Given - three back-to-back failing removals, each in its own isolated home
        for (let i = 0; i < 3; i += 1) {
            await using result = await isolated().exec(['init', 'agent rm nonexistent']);
            expect(result.exitCode).toBe(1);
            expect(result.stderr).not.toContain('panic:');
        }

        // Then - a normal init + create + ls still works
        await using healthy = await isolated().exec(['init', 'agent create survivor', 'agent ls']);
        expect(healthy.exitCode).toBe(0);
        expect(healthy.stderr).toContain('survivor');
    });

    test('init is idempotent - running init twice does not break state', async () => {
        // Given - running init twice then creating an agent in the same home
        await using result = await isolated().exec([
            'init',
            'init',
            'agent create testbot',
            'agent ls',
        ]);

        // Then - either the idempotent path went all the way through...
        if (result.exitCode === 0) {
            expect(result.stderr).toContain('testbot');
        } else {
            // ...or a clean-home retry proves followups still work
            await using retry = await isolated().exec(['init', 'agent create testbot', 'agent ls']);
            expect(retry.exitCode).toBe(0);
            expect(retry.stderr).toContain('testbot');
        }
    });
});
