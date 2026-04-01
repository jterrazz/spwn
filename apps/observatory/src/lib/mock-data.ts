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

export interface DriftingAgent {
  name: string;
  layers: number;
}

export const MOCK_WORLDS: World[] = [
  {
    id: "w-titan-84721",
    config: "default",
    agent: "neo",
    agents: [{ name: "neo", tier: "citizen", status: "running" }],
    status: "running",
    created_at: new Date(Date.now() - 1000 * 60 * 12).toISOString(),
    workspace: "~/acme-api",
  },
  {
    id: "w-europa-39205",
    config: "default",
    agent: "morpheus",
    agents: [
      { name: "morpheus", tier: "governor", status: "running" },
      { name: "trinity", tier: "citizen", status: "idle" },
    ],
    status: "running",
    created_at: new Date(Date.now() - 1000 * 60 * 45).toISOString(),
    workspace: "~/signews",
  },
  {
    id: "w-ganymede-51003",
    config: "backend",
    agent: "atlas",
    agents: [{ name: "atlas", tier: "citizen", status: "idle" }],
    status: "idle",
    created_at: new Date(Date.now() - 1000 * 60 * 120).toISOString(),
    workspace: "~/infra",
  },
];

export const MOCK_DRIFTING: DriftingAgent[] = [
  { name: "iris", layers: 3 },
  { name: "sage", layers: 1 },
];

export const MOCK_ACTIVITY = [
  { time: "2m ago", event: "neo completed task", world: "Titan", type: "success" },
  { time: "12m ago", event: "morpheus delegated to trinity", world: "Europa", type: "info" },
  { time: "45m ago", event: "atlas session started", world: "Ganymede", type: "info" },
  { time: "1h ago", event: "trinity mind updated — 3 new playbooks", world: "Europa", type: "success" },
  { time: "2h ago", event: "neo reflected — patterns promoted", world: "Titan", type: "info" },
];
