import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

const home = cli.with({ env: { SPWN_HOME: '$WORKDIR/spwn-home' } });

describe('activity', () => {
    test('a created agent is one agent.created event', async () => {
        // Given - one agent created in the run's own spwn home
        await using result = await home.exec('agent create neo');

        // Then - the activity log holds that one event, whole
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/activity.jsonl').grep('"type"')).toMatch(
            `
                {"id":"{{hex}}","timestamp":"{{iso8601}}","type":"agent.created","actor":"user","verb":"created","target":"neo","phrase":"You created neo","agent_id":"neo"}
            `,
        );
    });

    test('a removed agent appends an agent.deleted event', async () => {
        // Given - an agent created, then removed
        await using result = await home.exec(['agent create neo', 'agent rm neo']);

        // Then - the log holds the creation, then the deletion
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/activity.jsonl').grep('"type"')).toMatch(
            `
                {"id":"{{hex}}","timestamp":"{{iso8601}}","type":"agent.created","actor":"user","verb":"created","target":"neo","phrase":"You created neo","agent_id":"neo"}
                {"id":"{{hex}}","timestamp":"{{iso8601}}","type":"agent.deleted","actor":"user","verb":"deleted","target":"neo","phrase":"neo was deleted","agent_id":"neo"}
            `,
        );
    });

    test('a fork appends an agent.forked event naming its source', async () => {
        // Given - an agent created, then forked
        await using result = await home.exec(['agent create neo', 'agent fork neo trinity']);

        // Then - the log holds the creation, then the fork with its source in the metadata
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/activity.jsonl').grep('"type"')).toMatch(
            `
                {"id":"{{hex}}","timestamp":"{{iso8601}}","type":"agent.created","actor":"user","verb":"created","target":"neo","phrase":"You created neo","agent_id":"neo"}
                {"id":"{{hex}}","timestamp":"{{iso8601}}","type":"agent.forked","actor":"user","verb":"forked","target":"trinity","phrase":"trinity forked from neo","agent_id":"trinity","metadata":{"layers":["playbooks","journal"],"source":"neo"}}
            `,
        );
    });

    test('a slept agent appends an agent.slept event the agent itself acts', async () => {
        // Given - an agent created, then put to sleep
        await using result = await home.exec(['agent create neo', 'agent sleep neo']);

        // Then - the log holds the creation, then the sleep actored by neo
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/activity.jsonl').grep('"type"')).toMatch(
            `
                {"id":"{{hex}}","timestamp":"{{iso8601}}","type":"agent.created","actor":"user","verb":"created","target":"neo","phrase":"You created neo","agent_id":"neo"}
                {"id":"{{hex}}","timestamp":"{{iso8601}}","type":"agent.slept","actor":"neo","verb":"slept","target":"neo","phrase":"neo slept","agent_id":"neo","metadata":{"archived_playbooks":0,"pruned_sessions":0}}
            `,
        );
    });
});
