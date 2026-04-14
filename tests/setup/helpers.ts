import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

/**
 * Create an isolated SPWN_HOME directory with the required subdirectories.
 * Each call creates a unique temporary directory under the system temp path.
 *
 * @returns Absolute path to the new SPWN_HOME directory
 */
export function createSpwnHome(): string {
    const dir = mkdtempSync(join(tmpdir(), 'spwn-test-'));
    mkdirSync(join(dir, 'worlds'), { recursive: true });
    mkdirSync(join(dir, 'agents'), { recursive: true });
    return dir;
}

/**
 * Create a minimal agent Mind with the standard 6-layer directory structure.
 * Writes a default identity file so the agent is immediately usable.
 *
 * @param spwnHome - Path to the SPWN_HOME directory
 * @param name - Agent name (used as directory name under agents/)
 */
export function createAgent(spwnHome: string, name: string): void {
    const agentDir = join(spwnHome, 'agents', name);
    // Current Mind layout: core, skills, knowledge, playbooks, journal
    // (matches foundation.MindLayers)
    const layers = ['core', 'skills', 'knowledge', 'playbooks', 'journal'];
    for (const layer of layers) {
        mkdirSync(join(agentDir, layer), { recursive: true });
    }
    writeFileSync(
        join(agentDir, 'core', 'profile.md'),
        `# ${name}\n\nYou are a test agent named ${name}.\n\n## Purpose\n\nTest automation.\n\n## Traits\n\n- Reliable\n- Systematic\n`,
    );
    writeFileSync(join(agentDir, 'agent.yaml'), `role: worker\n`);
}

/**
 * Run async tasks with bounded concurrency.
 * Executes up to `maxConcurrency` tasks at a time, waiting for a slot
 * to open before launching the next one.
 *
 * @param tasks - Array of async task factories
 * @param maxConcurrency - Maximum number of concurrent tasks
 */
export async function runConcurrently(
    tasks: (() => Promise<void>)[],
    maxConcurrency: number,
): Promise<void> {
    const executing = new Set<Promise<void>>();

    for (const task of tasks) {
        const p = task().then(() => {
            executing.delete(p);
        });
        executing.add(p);

        if (executing.size >= maxConcurrency) {
            await Promise.race(executing);
        }
    }

    await Promise.all(executing);
}
