export interface Agent {
  name: string;
  tier: string;
  status: string;
}

export interface WorldManifest {
  cpu?: string;
  memory?: string;
  timeout?: string;
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
  status: "running" | "idle" | "stopped" | "creating" | "error";
  created_at: string;
  workspaces?: Workspace[];
  manifest?: WorldManifest;
}

/**
 * Short human-readable summary of a world's workspace mounts, for display.
 * Empty for ephemeral worlds (no host mounts).
 */
export function getWorkspaceSummary(world: Pick<World, "workspaces">): string {
  const ws = world.workspaces;
  if (!ws || ws.length === 0) return "ephemeral";
  if (ws.length === 1) return ws[0].path;
  return `${ws.length} workspaces`;
}

/**
 * Returns the user-facing name of a world: custom name if set, otherwise the
 * capitalized middle segment of the world id (e.g. "w-titan-abc123" → "Titan").
 */
export function getWorldName(world: Pick<World, "id" | "name">): string {
  if (world.name && world.name.trim()) return world.name.trim();
  const parts = world.id.split("-");
  return parts.length >= 2 ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1) : world.id;
}

export interface LimboAgent {
  name: string;
  layers: number;
}

export interface AgentProfile {
  name: string;
  tier: "governor" | "citizen" | "npc";
  engine: string;
  provider: string;
  purpose: string;
  persona: string;
  traits: string[];
  skills: string[];
  playbooks: string[];
  knowledge: string[];
  journal: { date: string; summary: string }[];
  bonds: { agent: string; relationship: string }[];
}

export interface AgentMessage {
  id: string;
  from: string;
  to: string;
  content: string;
  timestamp: string;
  read: boolean;
  channel: string;
}

export interface Snapshot {
  id: string;
  worldId: string;
  name: string;
  created_at: string;
  size: string;
  agents: number;
}

export interface LogEntry {
  timestamp: string;
  level: "info" | "warn" | "error" | "debug";
  source: string;
  message: string;
}

// Available configs for the Create World dialog
export const AVAILABLE_CONFIGS = ["default", "backend", "frontend", "fullstack", "devops", "minimal"];

// Available tiers for the Create Agent dialog
export const AVAILABLE_TIERS = ["governor", "citizen", "npc"] as const;
