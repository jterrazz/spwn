import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome } from '../../setup/helpers.js';
import { MindAssertion } from '../../setup/mind-assertion.js';
import { expectLine, expectTableHeader, expectTableRow } from '../../setup/output-helpers.js';
import { spwn } from '../../setup/spwn.specification.js';

describe('agent CRUD', () => {
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

    test('agent new creates agent with the 5-layer Mind', async () => {
        // WHEN - creating a new agent
        const result = await spwn('agent new').exec('agent new neo').run();

        // THEN - agent is created with structured status output
        expect(result.exitCode).toBe(0);
        expectLine(result.output, /→ Creating agent "neo"\.\.\./);
        expectLine(result.output, /✓ Created agent\s+neo/);
        expectLine(result.output, /✓ Created profile\s+profile\.md/);
        expectLine(result.output, /✓ Spawn with: spwn up --agent neo/);
    });

    test('init duplicate fails', async () => {
        // GIVEN - an agent already exists
        await spwn('first new').exec('agent new neo').run();

        // WHEN - creating the same agent again
        const result = await spwn('duplicate init').exec('agent new neo').run();

        // THEN - exits with error showing duplicate message
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /✗ Agent creation failed\s+agent "neo" already exists/);
    });

    test('list shows created agents', async () => {
        // GIVEN - two agents have been created
        await spwn('create neo').exec('agent new neo').run();
        await spwn('create trinity').exec('agent new trinity').run();

        // WHEN - listing agents
        const result = await spwn('list agents').exec('agent ls').run();

        // THEN - both agents appear in a table with correct columns
        expect(result.exitCode).toBe(0);
        expectTableHeader(result.output, ['NAME', 'WORLD', 'STATUS']);
        expectTableRow(result.output, ['neo', 'unattached']);
        expectTableRow(result.output, ['trinity', 'unattached']);
    });

    test('show prints agent details', async () => {
        // GIVEN - an agent exists
        await spwn('create for show').exec('agent new neo').run();

        // WHEN - inspecting the agent
        const result = await spwn('show agent').exec('agent show neo').run();

        // THEN - details include agent name, world status, and Mind layers
        expect(result.exitCode).toBe(0);
        expectLine(result.output, /Agent:\s+neo/);
        expectLine(result.output, /World:\s+unattached/);
        expectLine(result.output, /core\/\s+profile\.md/);
        expectLine(result.output, /skills\/\s+\(empty\)/);
        expectLine(result.output, /knowledge\/\s+\(empty\)/);
        expectLine(result.output, /playbooks\/\s+\(empty\)/);
        expectLine(result.output, /journal\/\s+\(empty\)/);
    });

    test('list on empty home returns no agents', async () => {
        // WHEN - listing agents with no agents created
        const result = await spwn('list empty').exec('agent ls').run();

        // THEN - exits successfully (no agents)
        expect(result.exitCode).toBe(0);
    });

    test('show on non-existent agent fails', async () => {
        // WHEN - showing an agent that does not exist
        const result = await spwn('show missing').exec('agent show nonexistent').run();

        // THEN - exits with error showing not found
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /agent "nonexistent" not found/);
    });

    test('delete removes agent', async () => {
        // GIVEN - an agent exists
        await spwn('create temp').exec('agent new temp').run();

        // WHEN - deleting the agent
        const result = await spwn('delete agent').exec('agent rm temp').run();

        // THEN - exits successfully with structured status
        expect(result.exitCode).toBe(0);
        expectLine(result.output, /→ Deleting agent "temp"\.\.\./);
        expectLine(result.output, /✓ Deleted agent\s+temp/);

        // AND - agent no longer appears in list
        const list = await spwn('list after delete').exec('agent ls').run();
        const tableLines = list.output.split('\n').filter((l) => l.includes('temp'));
        expect(tableLines.length).toBe(0);
    });

    test('delete non-existent agent fails', async () => {
        // WHEN - deleting an agent that does not exist
        const result = await spwn('delete missing').exec('agent rm nonexistent').run();

        // THEN - exits with error showing not found
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /✗ Delete failed\s+agent "nonexistent" not found/);
    });

    test('talk requires running world', async () => {
        // GIVEN - an agent exists but is not in any world
        await spwn('create neo for talk').exec('agent new neo').run();

        // WHEN - trying to talk to the agent
        const result = await spwn('talk without world').exec('agent talk neo hello').run();

        // THEN - exits with error about no active world
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /agent "neo" is not in any active world/);
    });

    test('list shows world column headers', async () => {
        // GIVEN - an agent has been created
        await spwn('create for list').exec('agent new atlas').run();

        // WHEN - listing agents
        const result = await spwn('list with world').exec('agent ls').run();

        // THEN - output includes table with world-related columns
        expect(result.exitCode).toBe(0);
        expectTableHeader(result.output, ['NAME', 'WORLD', 'STATUS']);
        expectTableRow(result.output, ['atlas', 'unattached']);
    });

    test('delete actually removes Mind directory from disk', async () => {
        // GIVEN - agent exists
        await spwn('create temp for disk check').exec('agent new temp').run();
        // Verify Mind directory exists
        new MindAssertion(home, 'temp').exists().hasLayer('core');

        // WHEN - deleting the agent
        const result = await spwn('delete temp disk').exec('agent rm temp').run();

        // THEN - Mind directory is gone from filesystem
        expect(result.exitCode).toBe(0);
        const agentDir = join(home, 'agents', 'temp');
        expect(existsSync(agentDir)).toBe(false);
    });

    test('cannot show agent after delete', async () => {
        // GIVEN - agent is created then deleted
        await spwn('create for show-delete').exec('agent new temp').run();
        await spwn('delete for show-delete').exec('agent rm temp').run();

        // WHEN - showing the deleted agent
        const result = await spwn('show after delete').exec('agent show temp').run();

        // THEN - exits with error showing not found
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /agent "temp" not found/);
    });
});
