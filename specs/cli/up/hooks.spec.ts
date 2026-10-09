import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

const SESSION_HOOK = 'echo "session-start=$(date -u +%FT%TZ)" > /tmp/hook-pilot-session.log';
const BASH_HOOK = 'echo "bash $(date -u +%FT%TZ)" >> /tmp/hook-pilot-bash.log';

describe('hooks', () => {
    test('the subscribed hooks render into the agent settings, and each command runs inside the world', async () => {
        // Given - the hook-pilot project, whose neo subscribes to a session hook and a Bash hook, brought up
        await using result = await cli.fixture(directory('./__fixtures__/hook-pilot/')).exec('up');

        // Then - the rendered settings, whole, and what running both commands inside the container leaves
        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        const settings: unknown = JSON.parse(neo.file('/agents/neo/.claude/settings.json').content);
        const ran = await neo.exec(
            `sh -c '${SESSION_HOOK} && ${BASH_HOOK} && cat /tmp/hook-pilot-session.log /tmp/hook-pilot-bash.log'`,
        );
        expect({ ran: ran.stdout.text, settings }).toStrictEqual({
            ran: expect.stringMatching(
                /^session-start=\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\nbash \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\n$/u,
            ) as unknown,
            settings: {
                hooks: {
                    PreToolUse: [
                        { hooks: [{ command: BASH_HOOK, type: 'command' }], matcher: 'Bash' },
                    ],
                    SessionStart: [
                        { hooks: [{ command: SESSION_HOOK, type: 'command' }], matcher: '*' },
                    ],
                },
                skipDangerousModePermissionPrompt: true,
            },
        });
    });
});
