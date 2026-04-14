import { describe, expect, test } from 'vitest';

import { dockerSpec } from '../../../setup/cli.specification.js';

/**
 * Agent messaging under the docker() spec mode.
 *
 * `spwn agent send <to> [msg]` and `spwn agent inbox <to>` operate on a
 * running world: spwn finds the world containing the target agent via
 * docker labels, then writes/reads JSON message files under
 * `/world/inbox/<agent>/<id>.json` inside the container.
 *
 * Legacy semantics preserved:
 *   - send succeeds and prints "Sent message <from> → <to>"
 *   - inbox lists delivered messages (FROM/TYPE/STATUS columns)
 *   - --type and --from flags round-trip into the inbox view
 *   - send without --from defaults to "user"
 *   - send/inbox on non-existent agents fail with a friendly error
 *
 * Augmented over the legacy test:
 *   - Asserts the `/world/inbox/<agent>` directory and per-message JSON
 *     files exist directly inside the container via `.file(path)`
 *   - Reads a message file with `neo.exec('cat ...')` and parses JSON to
 *     verify sender/content/type persisted as expected
 *   - Verifies physics.md documents /world/inbox via in-container read
 */
describe('agent messaging', () => {
    test('send writes a message into the agent inbox inside the container', async () => {
        await using result = await dockerSpec('agent send basic')
            .project('docker-pilot')
            .exec(['up', 'agent send neo --from morpheus "implement webhooks"'])
            .run();

        expect(result.exitCode).toBe(0);
        // With a multi-command chain, only the last command's streams are
        // Captured, so we assert on the final "Sent message" confirmation.
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toMatch(/Sent message\s+morpheus → neo/);

        // And the JSON message file is present inside the container.
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        expect(neo.file('/world/inbox/neo').exists).toBe(true);

        const ls = await neo.exec('ls /world/inbox/neo');
        expect(ls.exitCode).toBe(0);
        ls.stdout.toContain('.json');

        // And the file body carries the right metadata.
        const find = await neo.exec('sh -c "cat /world/inbox/neo/*.json"');
        expect(find.exitCode).toBe(0);
        const body = find.stdout.text;
        expect(body).toContain('"from": "morpheus"');
        expect(body).toContain('"to": "neo"');
        expect(body).toContain('"content": "implement webhooks"');
        expect(body).toContain('"type": "task"');
    });

    test('inbox shows a delivered message with sender and content', async () => {
        await using result = await dockerSpec('agent inbox delivered')
            .project('docker-pilot')
            .exec(['up', 'agent send neo --from morpheus "implement webhooks"', 'agent inbox neo'])
            .run();

        expect(result.exitCode).toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toContain('morpheus');
        expect(combined).toContain('implement webhooks');
        // Table columns the legacy test asserted on.
        expect(combined).toContain('FROM');
        expect(combined).toContain('TYPE');
        expect(combined).toContain('STATUS');
    });

    test('inbox is empty before any send', async () => {
        await using result = await dockerSpec('agent inbox empty')
            .project('docker-pilot')
            .exec(['up', 'agent inbox neo'])
            .run();

        expect(result.exitCode).toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toContain('No messages');
    });

    test('multiple messages to the same agent all appear in the inbox', async () => {
        await using result = await dockerSpec('agent inbox multi')
            .project('docker-pilot')
            .exec([
                'up',
                'agent send neo --from morpheus "first message"',
                'agent send neo --from morpheus "second message"',
                'agent send neo --from morpheus "third message"',
                'agent inbox neo',
            ])
            .run();

        expect(result.exitCode).toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toContain('first message');
        expect(combined).toContain('second message');
        expect(combined).toContain('third message');

        // And three message files live in the container inbox.
        const ls = await result.container('neo').exec('sh -c "ls /world/inbox/neo/*.json | wc -l"');
        expect(ls.exitCode).toBe(0);
        expect(ls.stdout.text.trim()).toBe('3');
    });

    test('--type flag sets the message type and --from defaults to user', async () => {
        await using result = await dockerSpec('agent send flags')
            .project('docker-pilot')
            .exec(['up', 'agent send neo --type question "what is the matrix?"'])
            .run();

        expect(result.exitCode).toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        // Default --from is "user" (omitted on the send line above).
        expect(combined).toMatch(/Sent message\s+user → neo/);

        // And the persisted JSON has both pieces of metadata.
        const cat = await result.container('neo').exec('sh -c "cat /world/inbox/neo/*.json"');
        expect(cat.exitCode).toBe(0);
        expect(cat.stdout.text).toContain('"from": "user"');
        expect(cat.stdout.text).toContain('"type": "question"');
        expect(cat.stdout.text).toContain('"content": "what is the matrix?"');
    });

    test('send to a non-existent agent fails cleanly', async () => {
        await using result = await dockerSpec('agent send missing')
            .project('docker-pilot')
            .exec(['up', 'agent send nonexistent --from morpheus "hello"'])
            .run();

        // The up step succeeds; the send step fails — combined exit code non-zero.
        expect(result.exitCode).not.toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        expect(combined).toContain('nonexistent');
    });

    test('inbox on a non-existent agent fails cleanly', async () => {
        await using result = await dockerSpec('agent inbox missing')
            .project('docker-pilot')
            .exec(['up', 'agent inbox nonexistent'])
            .run();

        expect(result.exitCode).not.toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        expect(combined).toContain('nonexistent');
    });

    test('physics.md documents the /world/inbox communication channel', async () => {
        await using result = await dockerSpec('physics documents inbox')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const physics = result.container('neo').file('/world/physics.md').content;
        expect(physics).toContain('Communication');
        expect(physics).toContain('/world/inbox');
    });
});
