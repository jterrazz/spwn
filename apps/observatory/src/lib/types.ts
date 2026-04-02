export interface Agent {
  name: string;
  tier: string;
  status: string;
}

export interface World {
  id: string;
  config: string;
  agent: string;
  agents: Agent[];
  status: "running" | "idle" | "stopped" | "creating";
  created_at: string;
  workspace: string;
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
