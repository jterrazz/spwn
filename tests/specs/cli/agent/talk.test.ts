import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn agent talk <name> [msg]` under docker-aware mode.
 *
 * Talk opens a conversation with a running agent by shelling into the
 * agent's container and exec'ing the runtime binary (claude-code). The
 * happy-path tests pin `SPWN_BASE_IMAGE=spwn-test:latest` so the
 * container uses the mock `/usr/local/bin/claude` shipped in the test
 * image — otherwise the tests would need real Anthropic credentials.
 *
 * Every result binds with `await using` so the spawned container is
 * force-removed at scope exit (rule B5).
 */
describe('agent talk', () => {
    test('talk to an unattached agent fails cleanly', async () => {
        // Given - an orphan agent on disk with no active world, talked to
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['agent create orphan', 'agent talk orphan hello']);

        // Then - exit 1 with a friendly no-world error and no panic (scalpel: case-insensitive message probe)
        expect(result.exitCode).toBe(1);
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/not in any active world|no active world/i);
        expect(result.stderr).not.toContain('panic');
    });

    test('talk to a non-existent agent hints at agent create', async () => {
        // Given - talk against an agent that does not exist
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('agent talk does-not-exist hello');

        // Then - exit 1 with the friendly create hint, no raw Go wrapper noise
        expect(result.exitCode).toBe(1);
        expect(result.stderr).not.toContain('exit status 1');
        expect(result.stderr).not.toContain('panic');
        expect(result.stderr).toMatch('talk-missing-agent.txt');
    });

    test("talk against a live world exec's the runtime inside the container", async () => {
        // Given - a world brought up, then talked to with the mock runtime image
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'agent talk neo "list files in /workspace"']);

        // Then - talk shelled into the running container and the mock observed the bind mounts
        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        const cat = await neo.exec('cat /tmp/claude-mock.json');
        expect(cat.exitCode).toBe(0);
        const receipt = JSON.parse(cat.stdout.text) as {
            claude_md_exists: boolean;
            mind_exists: boolean;
        };
        expect(receipt.mind_exists).toBe(true);
        expect(receipt.claude_md_exists).toBe(true);
    });

    test('talk against a codex world sees AGENTS.md, native skills, and resumes the thread', async () => {
        // Given - a codex world brought up, then talked to twice to resume the thread
        await using result = await cli
            .fixture('$FIXTURES/codex-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec([
                'up',
                'agent talk neo "Do you see your native skills?"',
                'agent talk neo "Continue in the same Codex thread."',
            ]);

        // Then - the codex mock receipt confirms the exec, resume, and workspace/skill view
        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        const receiptRaw = await neo.exec('cat /tmp/codex-mock.json');
        expect(receiptRaw.exitCode).toBe(0);
        const receipt = JSON.parse(receiptRaw.stdout.text) as {
            agents_md_content: string;
            agents_md_exists: boolean;
            json_mode: boolean;
            prompt: string;
            resume: boolean;
            skills_exists: boolean;
            skill_content: string;
            subcommand: string;
            thread_id: string;
            workspace_exists: boolean;
        };
        expect(receipt.subcommand).toBe('exec');
        expect(receipt.json_mode).toBe(true);
        expect(receipt.resume).toBe(true);
        expect(receipt.prompt).toBe('Continue in the same Codex thread.');
        expect(receipt.thread_id).toMatch(/^th_mock_/);
        expect(receipt.agents_md_exists).toBe(true);
        expect(receipt.agents_md_content).toContain('skill/focus');
        expect(receipt.skills_exists).toBe(true);
        expect(receipt.skill_content).toContain('Focus Skill');
        expect(receipt.workspace_exists).toBe(true);
        const prompt = await neo.exec('grep -q "Codex pilot prompt" /agents/neo/AGENTS.md');
        expect(prompt.exitCode).toBe(0);
        const skill = await neo.exec('test -f /agents/neo/.agents/skills/focus/SKILL.md');
        expect(skill.exitCode).toBe(0);
    });

    test('talk can be invoked multiple times on the same world', async () => {
        // Given - a world brought up, then talked to twice back-to-back
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'agent talk neo hello', 'agent talk neo "hello again"']);

        // Then - the world survives both talks and the latest mock receipt is present
        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        expect(neo.file('/tmp/claude-mock.json').exists).toBe(true);
    });

    test('agent ls --json shows neo attached to the running world', async () => {
        // Given - a world brought up, then listed as JSON
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'agent ls --json']);

        // Then - neo is reported running under a project-mode roster (scalpel: structural probe over dynamic runtime status)
        expect(result.exitCode).toBe(0);
        const report = result.json.value as {
            agents: Array<{ name: string; status: string; world?: string }>;
            mode: string;
        };
        expect(report.mode).toBe('project');
        const neo = report.agents.find((a) => a.name === 'neo');
        expect(neo).toBeDefined();
        expect(neo?.status).toMatch(/running/);
    });

    test('world list --json surfaces the running world with its agents', async () => {
        // Given - a world brought up, then listed as JSON
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'world list --json']);

        // Then - one running project world attaches neo
        expect(result.exitCode).toBe(0);
        const list = result.json.value as {
            mode: string;
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(list.mode).toBe('project');
        expect(list.worlds).toHaveLength(1);
        expect(list.worlds[0].agents).toContain('neo');
        expect(list.worlds[0].status).toBe('running');
    });

    test('after down, agent ls shows neo as unattached', async () => {
        // Given - a world brought up then torn down, then listed as JSON
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'down', 'agent ls --json']);

        // Then - neo is no longer reported running (scalpel: structural probe over dynamic status)
        expect(result.exitCode).toBe(0);
        const report = result.json.value as {
            agents: Array<{ name: string; status: string }>;
        };
        const neo = report.agents.find((a) => a.name === 'neo');
        expect(neo).toBeDefined();
        expect(neo?.status).not.toMatch(/running/);
    });

    test('talk routes to the live world after a previous world was destroyed', async () => {
        // Given - a full up/down cycle followed by a fresh second up, then talked to
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'down', 'up', 'agent talk neo hello']);

        // Then - talk reaches the currently live container, not the torn-down one
        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        const cat = await neo.exec('cat /tmp/claude-mock.json');
        expect(cat.exitCode).toBe(0);
        const receipt = JSON.parse(cat.stdout.text) as {
            mind_exists: boolean;
        };
        expect(receipt.mind_exists).toBe(true);
    });

    test('agent inspect prints layer details when attached to a world', async () => {
        // Given - a world brought up, then the agent inspected
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'agent inspect neo']);

        // Then - the Mind tree renders on stderr (scalpel: regex over the mind-tree render)
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/Agent:\s+neo/);
        expect(stderr).toMatch(/playbooks\//);
    });
});
