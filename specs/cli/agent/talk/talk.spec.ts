import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot read a container back: the simulated runtime’s receipt is read inside the running world

const simulated = cli.with({ env: { SPWN_BASE_IMAGE: 'spwn-test:latest' } });

describe('talk', () => {
    test('a talk runs the Claude Code runtime inside the world, which finds the mind, its prompt and the workspace', async () => {
        // Given - the project brought up on the simulator image, then talked to
        await using result = await simulated
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec(['up', 'agent talk neo "list files in /workspace"']);

        // Then - the talk's banner, and the receipt the simulated runtime wrote inside the container
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/claude-talk.txt'));
        const receipt = await result.container('neo').exec('cat /tmp/claude-mock.json');
        expect(JSON.parse(receipt.stdout.text)).toStrictEqual({
            claude_md_content: expect.stringMatching(
                /^# neo, worker in world "world-neo-[0-9a-f]{5}"\n\n## Identity\n/u,
            ) as unknown,
            claude_md_exists: true,
            exit_code: 0,
            mind_exists: true,
            pid: expect.any(Number) as unknown,
            resume: false,
            session_id: '',
            workspace_exists: true,
        });
    });

    test('a second talk to a codex world resumes the codex thread, with AGENTS.md and the native skill in sight', async () => {
        // Given - a codex project brought up, then talked to twice
        await using result = await simulated
            .fixture(directory('../../__fixtures__/codex-pilot/'))
            .exec([
                'up',
                'agent talk neo "Do you see your native skills?"',
                'agent talk neo "Continue in the same Codex thread."',
            ]);

        // Then - the reply, and the receipt of the second exec: same thread, resumed, the skill read whole
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('mock-codex reply\n');
        const receipt = await result.container('neo').exec('cat /tmp/codex-mock.json');
        expect(JSON.parse(receipt.stdout.text)).toStrictEqual({
            agents_md_content: expect.stringMatching(
                /^# neo, worker in world "world-neo-[0-9a-f]{5}"\n\n## Identity\n\nYour soul: who you are\. Re-read this when you feel lost\.\n\nCodex pilot fixture agent\.\n/u,
            ) as unknown,
            agents_md_exists: true,
            exit_code: 0,
            json_mode: true,
            mind_exists: true,
            pid: expect.any(Number) as unknown,
            prompt: 'Continue in the same Codex thread.',
            resume: true,
            skill_content:
                '---\nname: focus\ndescription: Use when a task needs a concrete inspection plan before editing.\n---\n\n# Focus Skill\n\nWhen a task is ambiguous, list the concrete files and commands you will\ninspect before editing.\n',
            skills_exists: true,
            subcommand: 'exec',
            thread_id: expect.stringMatching(/^th_mock_\d+$/u) as unknown,
            workspace_exists: true,
        });
    });

    test('a talk after the world was torn down and brought up again reaches the new container', async () => {
        // Given - the project up, down, up again, then talked to
        await using result = await simulated
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'up', 'agent talk neo hello']);

        // Then - the live container's runtime ran the talk
        expect(result.exitCode).toBe(0);
        const receipt = await result.container('neo').exec('cat /tmp/claude-mock.json');
        expect(JSON.parse(receipt.stdout.text)).toStrictEqual({
            claude_md_content: expect.stringMatching(
                /^# neo, worker in world "world-neo-[0-9a-f]{5}"\n/u,
            ) as unknown,
            claude_md_exists: true,
            exit_code: 0,
            mind_exists: true,
            pid: expect.any(Number) as unknown,
            resume: false,
            session_id: '',
            workspace_exists: true,
        });
    });
});
