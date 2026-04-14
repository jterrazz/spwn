import { afterEach, describe, expect, test } from 'vitest';

import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../../setup/spwn.specification.js';

/**
 * Docker-backed system skills / AGENTS.md injection tests.
 *
 * These spawn a real world container via `spwn world --agent …` and
 * inspect the container filesystem, so they need a running Docker
 * daemon. They remain on the legacy `createTestContext` helpers
 * because the `@jterrazz/test` spec runner has no container-
 * inspection surface today.
 */

describe('system skills infrastructure', () => {
    let ctx: TestContext;

    afterEach(() => {
        ctx?.cleanup();
    });

    test('system AGENTS.md is injected into world containers', () => {
        // GIVEN - an initialized SPWN_HOME with agent
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - spawning a world with a single agent
        const spawn = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawn.output)!;
        expect(id).toBeTruthy();

        // THEN - AGENTS.md exists inside container at /world/AGENTS.md
        ctx.world(id).toHaveFile('/world/AGENTS.md');

        // AND - contains expected Agent Operating Manual content
        const content = ctx.world(id).readFile('/world/AGENTS.md');
        expect(content).toBeTruthy();
        expect(content.length).toBeGreaterThan(100);
    });

    test('system skills are injected into world containers', () => {
        // GIVEN - an initialized SPWN_HOME
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - spawning a world
        const spawn = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawn.output)!;
        expect(id).toBeTruthy();

        // THEN - /world/skills/ directory exists inside container
        ctx.world(id).toHaveDirectory('/world/skills');

        // AND - key system skill files exist
        const skillsExist = ctx.world(id).fileExists('/world/skills');
        expect(skillsExist).toBe(true);
    });

    test('AGENT.md is generated per world with agent name', () => {
        // GIVEN - an initialized SPWN_HOME with agent
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - spawning a world
        const spawn = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawn.output)!;
        expect(id).toBeTruthy();

        // THEN - AGENTS.md exists and references the agent name in roster
        ctx.world(id).toHaveFile('/world/AGENTS.md');
        ctx.world(id).toHaveFile('/world/roster.md');
        const roster = ctx.world(id).readFile('/world/roster.md');
        expect(roster).toContain('neo');
    });

    test('agent can read system skills directory', () => {
        // GIVEN - an initialized SPWN_HOME with agent
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - spawning a world
        const spawn = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawn.output)!;
        expect(id).toBeTruthy();

        // THEN - the skills directory is accessible
        const universe = ctx.world(id);
        universe.toHaveDirectory('/world/skills');
    });
});
