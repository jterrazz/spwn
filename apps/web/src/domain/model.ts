export interface Agent {
    name: string;
    role: string;
    status: string;
}

export interface WorldManifest {
    elements?: string[];
}

export interface Workspace {
    name: string;
    path: string;
    readonly?: boolean;
}

export interface World {
    id: string;
    name?: string;
    config: string;
    agent: string;
    agents: Agent[];
    runtime?: string;
    status: 'creating' | 'error' | 'idle' | 'running' | 'stopped';
    created_at: string;
    workspaces?: undefined | Workspace[];
    manifest?: WorldManifest;
}

/** Short human-readable summary of a world's workspace mounts, for display. Empty for ephemeral worlds (no host mounts). */
export function getWorkspaceSummary(world: Pick<World, 'workspaces'>): string {
    const ws = world.workspaces;
    if (!ws || ws.length === 0) {
        return 'ephemeral';
    }
    const [only] = ws;
    if (ws.length === 1 && only) {
        return only.path;
    }
    return `${ws.length} workspaces`;
}

/** Returns the user-facing name of a world: custom name if set, otherwise the capitalized middle segment of the world id (e.g. "w-titan-abc123" → "Titan"). */
export function getWorldName(world: Pick<World, 'id' | 'name'>): string {
    const name = world.name?.trim() ?? '';
    if (name !== '') {
        return name;
    }
    const [, middle] = world.id.split('-');
    return middle === undefined || middle === ''
        ? world.id
        : middle.charAt(0).toUpperCase() + middle.slice(1);
}

export interface Team {
    slug: string;
    name: string;
    color?: string;
    description?: string;
    members?: string[];
}

export interface AgentProfile {
    name: string;
    role: string;
    team?: string;
    engine: string;
    provider: string;
    purpose: string;
    profile: string;
    traits: string[];
    skills: string[];
    journal: { date: string; summary: string }[];
}

export const AVAILABLE_CONFIGS = [
    'default',
    'backend',
    'frontend',
    'fullstack',
    'devops',
    'minimal',
];

export const AVAILABLE_ROLES = ['chief', 'manager', 'worker'] as const;

export interface OrganizationRole {
    name: string;
    level: number;
    can_command?: string[] | undefined;
    reports_to?: string | undefined;
    max_per_world?: number | undefined;
    permissions?: string[] | undefined;
}

export interface Organization {
    slug: string;
    name: string;
    description?: string | undefined;
    roles: OrganizationRole[];
}
