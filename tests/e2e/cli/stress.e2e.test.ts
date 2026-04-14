import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome, runConcurrently } from '../../setup/helpers.js';
import { stripAnsi } from '../../setup/output-helpers.js';
import { spwn } from '../../setup/spwn.specification.js';

describe('CLI stress tests', () => {
    let home: string;
    let originalSpwnHome: string | undefined;

    beforeEach(() => {
        originalSpwnHome = process.env.SPWN_HOME;
        home = createSpwnHome();
        process.env.SPWN_HOME = home;
    });

    afterEach(() => {
        if (originalSpwnHome !== undefined) {
            process.env.SPWN_HOME = originalSpwnHome;
        } else {
            delete process.env.SPWN_HOME;
        }
    });

    test('create and destroy 5 agents rapidly with concurrency', async () => {
        // GIVEN - an initialized home
        await spwn('init').exec('init').run();

        const agentNames = ['alpha', 'bravo', 'charlie', 'delta', 'echo'];

        // WHEN - creating 5 agents concurrently (max 3 at a time)
        const createTasks = agentNames.map(
            (name) => () =>
                spwn(`create ${name}`)
                    .exec(`agent create ${name}`)
                    .run()
                    .then((result) => {
                        expect(result.exitCode).toBe(0);
                    }),
        );
        await runConcurrently(createTasks, 3);

        // THEN - all agents exist in list
        const listResult = await spwn('list all').exec('agent ls').run();
        expect(listResult.exitCode).toBe(0);
        const output = stripAnsi(listResult.output);
        for (const name of agentNames) {
            expect(output).toContain(name);
        }

        // WHEN - deleting all 5 agents concurrently (max 3 at a time)
        const deleteTasks = agentNames.map(
            (name) => () =>
                spwn(`rm ${name}`)
                    .exec(`agent rm ${name}`)
                    .run()
                    .then((result) => {
                        expect(result.exitCode).toBe(0);
                    }),
        );
        await runConcurrently(deleteTasks, 3);

        // THEN - no agents remain (except possibly 'default' created by init)
        const finalList = await spwn('final list').exec('agent ls').run();
        expect(finalList.exitCode).toBe(0);
        const finalOutput = stripAnsi(finalList.output);
        for (const name of agentNames) {
            expect(finalOutput).not.toContain(name);
        }
    });

    test('rapid sequential commands do not corrupt state', async () => {
        // GIVEN - an initialized home
        await spwn('init').exec('init').run();

        // WHEN - running many sequential create/delete cycles
        for (let i = 0; i < 5; i++) {
            const name = `rapid-${i}`;
            const createResult = await spwn(`create ${name}`).exec(`agent create ${name}`).run();
            expect(createResult.exitCode).toBe(0);

            const rmResult = await spwn(`rm ${name}`).exec(`agent rm ${name}`).run();
            expect(rmResult.exitCode).toBe(0);
        }

        // THEN - agent list is clean (no remnants from rapid cycles)
        const listResult = await spwn('final list').exec('agent ls').run();
        expect(listResult.exitCode).toBe(0);
        const output = stripAnsi(listResult.output);
        for (let i = 0; i < 5; i++) {
            expect(output).not.toContain(`rapid-${i}`);
        }
    });
});
