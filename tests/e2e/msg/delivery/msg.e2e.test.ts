import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn agent send` / `spwn agent inbox` under the docker() spec mode.
 *
 * Port of the legacy CLI-only `tests/e2e/messaging/msg.e2e.test.ts`. The
 * legacy tests ran against an in-process test context that no longer
 * matches how messaging works in project mode — messaging writes JSON
 * files into `/world/inbox/<agent>/` inside the running world container.
 *
 * Extensive coverage of the same surface already lives under
 * `tests/e2e/world/messaging/` (physics docs, type flag, JSON body
 * parsing, multi-message). This file focuses on the CLI-facing
 * behaviours the legacy msg test cared about:
 *   - send prints a confirmation line
 *   - inbox lists delivered messages with FROM/TYPE/STATUS columns
 *   - --from defaults to "user"
 *   - send/inbox on a non-existent agent fail cleanly (no panics)
 *
 * Augmented over the legacy test:
 *   - Reads the message JSON back out of the container to confirm
 *     the send actually wrote the file (not just printed a banner)
 */
describe('agent messaging (msg)', () => {
    test('send prints a confirmation and writes a JSON message file', async () => {
        await using result = await spec('msg send basic')
            .project('docker-pilot')
            .exec(['up', 'agent send neo --from morpheus "hello world"'])
            .run();

        expect(result.exitCode).toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toMatch(/[Ss]ent message/);
        expect(combined).toContain('morpheus');
        expect(combined).toContain('neo');

        // The message file really lives in the container inbox.
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        expect(neo.file('/world/inbox/neo').exists).toBe(true);

        const cat = await neo.exec('sh -c "cat /world/inbox/neo/*.json"');
        expect(cat.exitCode).toBe(0);
        expect(cat.stdout.text).toContain('"from": "morpheus"');
        expect(cat.stdout.text).toContain('"content": "hello world"');
    });

    test('send without --from defaults to "user"', async () => {
        await using result = await spec('msg send default from')
            .project('docker-pilot')
            .exec(['up', 'agent send neo "hi from default"'])
            .run();

        expect(result.exitCode).toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toMatch(/[Ss]ent message/);

        const cat = await result.container('neo').exec('sh -c "cat /world/inbox/neo/*.json"');
        expect(cat.exitCode).toBe(0);
        expect(cat.stdout.text).toContain('"from": "user"');
        expect(cat.stdout.text).toContain('"content": "hi from default"');
    });

    test('inbox shows the sender and body of a delivered message', async () => {
        await using result = await spec('msg inbox shows message')
            .project('docker-pilot')
            .exec(['up', 'agent send neo --from morpheus "inbox test"', 'agent inbox neo'])
            .run();

        expect(result.exitCode).toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toContain('morpheus');
        expect(combined).toContain('inbox test');
        // Table columns the legacy test matched against.
        expect(combined).toContain('FROM');
        expect(combined).toContain('TYPE');
        expect(combined).toContain('STATUS');
    });

    test('send to a non-existent agent fails cleanly', async () => {
        await using result = await spec('msg send missing')
            .project('docker-pilot')
            .exec(['up', 'agent send nonexistent --from morpheus "hello"'])
            .run();

        expect(result.exitCode).not.toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('ReferenceError');
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        expect(combined).toContain('nonexistent');
    });

    test('inbox on a non-existent agent fails cleanly', async () => {
        await using result = await spec('msg inbox missing')
            .project('docker-pilot')
            .exec(['up', 'agent inbox nonexistent'])
            .run();

        expect(result.exitCode).not.toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('ReferenceError');
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        expect(combined).toContain('nonexistent');
    });
});
