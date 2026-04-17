/**
 * Server-side functions to read spwn data from the filesystem.
 * These run in Node.js (RSC or API routes) - NOT in the browser.
 *
 * Data sources:
 *   ~/.spwn/state.json       - world list (array of World objects)
 *   ~/.spwn/agents/{name}/   - agent mind directories
 */

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

import type { AgentProfile, LimboAgent, World } from './types';

// ── Paths ──

function spwnHome(): string {
    return process.env.SPWN_HOME || path.join(os.homedir(), '.spwn');
}

function statePath(): string {
    return path.join(spwnHome(), 'state.json');
}

function agentsDir(): string {
    return path.join(spwnHome(), 'agents');
}

// ── Raw types matching Go structs ──

interface RawAgentRecord {
    name: string;
    agent_id: string;
    role?: string;
    status: string;
}

interface RawWorld {
    id: string;
    config: string;
    agent?: string;
    agent_id?: string;
    backend?: string;
    container_id?: string;
    workspaces?: { name: string; path: string; readonly?: boolean }[];
    workspace?: string; // Legacy
    mind_path?: string;
    gate_dir?: string;
    status: string;
    created_at: string;
    agents?: RawAgentRecord[];
}

// ── World reading ──

function rawToWorld(raw: RawWorld): World {
    const agents = (raw.agents ?? []).map((a) => ({
        name: a.name,
        role: a.role || 'worker',
        status: a.status,
    }));

    // If no agents array but single agent field, create one
    if (agents.length === 0 && raw.agent) {
        agents.push({
            name: raw.agent,
            role: 'worker',
            status: raw.status,
        });
    }

    // Prefer the new `workspaces` array; migrate legacy `workspace` string.
    let workspaces = raw.workspaces;
    if ((!workspaces || workspaces.length === 0) && raw.workspace) {
        workspaces = [{ name: 'default', path: raw.workspace }];
    }

    return {
        id: raw.id,
        config: raw.config || 'default',
        agent: raw.agent || (agents[0]?.name ?? ''),
        agents,
        status: raw.status as World['status'],
        created_at: raw.created_at,
        workspaces,
    };
}

/**
 * Read worlds from ~/.spwn/state.json.
 * Returns an empty array if the file doesn't exist.
 */
export async function getWorlds(): Promise<World[]> {
    try {
        const data = await fs.promises.readFile(statePath(), 'utf8');
        if (!data.trim()) {
            return [];
        }
        const raw: RawWorld[] = JSON.parse(data);
        return raw.filter((w) => w.status !== 'destroyed').map(rawToWorld);
    } catch {
        return [];
    }
}

/**
 * Get a single world by ID.
 */
export async function getWorld(id: string): Promise<null | World> {
    const worlds = await getWorlds();
    return worlds.find((w) => w.id === id) ?? null;
}

// ── Agent reading ──

interface AgentMindInfo {
    name: string;
    path: string;
    layers: Record<string, string[]>;
}

const MIND_LAYERS = ['identity', 'skills', 'playbooks', 'journal'];

async function readDirSafe(dir: string): Promise<string[]> {
    try {
        const entries = await fs.promises.readdir(dir);
        return entries.filter((e) => !e.startsWith('.'));
    } catch {
        return [];
    }
}

async function inspectAgent(name: string): Promise<AgentMindInfo | null> {
    const agentPath = path.join(agentsDir(), name);
    try {
        const stat = await fs.promises.stat(agentPath);
        if (!stat.isDirectory()) {
            return null;
        }
    } catch {
        return null;
    }

    const layers: Record<string, string[]> = {};
    for (const layer of MIND_LAYERS) {
        layers[layer] = await readDirSafe(path.join(agentPath, layer));
    }

    return { name, path: agentPath, layers };
}

/**
 * List all agents from ~/.spwn/agents/.
 */
export async function getAgents(): Promise<AgentMindInfo[]> {
    try {
        const entries = await fs.promises.readdir(agentsDir(), { withFileTypes: true });
        const agents: AgentMindInfo[] = [];
        for (const entry of entries) {
            if (!entry.isDirectory() || entry.name.startsWith('.')) {
                continue;
            }
            const info = await inspectAgent(entry.name);
            if (info) {
                agents.push(info);
            }
        }
        return agents;
    } catch {
        return [];
    }
}

/**
 * Get limbo agents - agents that exist in ~/.spwn/agents/ but are not
 * currently assigned to any active world.
 */
export async function getLimboAgents(worlds: World[]): Promise<LimboAgent[]> {
    const agents = await getAgents();
    if (agents.length === 0) {
        return [];
    }

    const activeAgentNames = new Set<string>();
    for (const w of worlds) {
        for (const a of w.agents) {
            activeAgentNames.add(a.name);
        }
    }

    const limbo = agents
        .filter((a) => !activeAgentNames.has(a.name))
        .map((a) => ({
            name: a.name,
        }));

    return limbo;
}

/**
 * Get full agent profile by reading their mind directory.
 * Returns null if the agent directory doesn't exist.
 */
export async function getAgentProfile(name: string): Promise<AgentProfile | null> {
    const info = await inspectAgent(name);
    if (!info) {
        return null;
    }

    // Read core identity files for purpose/profile
    let purpose = '';
    let profileText = '';
    const coreFiles = info.layers['core'] ?? [];
    for (const file of coreFiles) {
        if (file.endsWith('.md')) {
            try {
                const content = await fs.promises.readFile(
                    path.join(info.path, 'core', file),
                    'utf8',
                );
                // Extract purpose from content
                const purposeMatch = content.match(
                    /## (?:Purpose|Your Identity)\n([\s\S]*?)(?:\n##|$)/,
                );
                if (purposeMatch) {
                    purpose = purposeMatch[1].trim().slice(0, 200);
                }
                // Use first paragraph as profile text
                const lines = content.split('\n').filter((l) => l.trim() && !l.startsWith('#'));
                if (lines.length > 0) {
                    profileText = lines[0].trim();
                }
            } catch {
                // Ignore read errors
            }
        }
    }

    // Read journal entries
    const journalFiles = (info.layers['journal'] ?? []).filter((f) => f.endsWith('.md'));
    const journal: { date: string; summary: string }[] = [];
    for (const file of journalFiles.slice(-10).toReversed()) {
        try {
            const content = await fs.promises.readFile(
                path.join(info.path, 'journal', file),
                'utf8',
            );
            // Extract date from filename (e.g., 2026-04-01_w-titan.md)
            const dateMatch = file.match(/^(\d{4}-\d{2}-\d{2})/);
            const date = dateMatch ? dateMatch[1] : file.replace(/\.md$/, '');
            // Use first non-header line as summary
            const summaryLine = content
                .split('\n')
                .find((l) => l.trim() && !l.startsWith('#') && !l.startsWith('---'));
            journal.push({
                date,
                summary: summaryLine?.trim() || '(empty entry)',
            });
        } catch {
            // Skip unreadable files
        }
    }

    // Build skills list from skills/ directory
    const skills = (info.layers['skills'] ?? []).map((f) => f.replace(/\.md$/, ''));

    return {
        name,
        role: 'worker' as const,
        engine: 'claude-code',
        provider: 'anthropic',
        purpose: purpose || '',
        profile: profileText || '',
        traits: [],
        skills,
        journal,
    };
}

/**
 * Get the mind file tree for an agent (used in the "Files" tab).
 */
export async function getAgentMindTree(name: string): Promise<Record<string, string[]>> {
    const info = await inspectAgent(name);
    if (!info) {
        return {};
    }
    return info.layers;
}
