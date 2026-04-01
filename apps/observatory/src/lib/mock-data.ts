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

export const MOCK_WORLDS: World[] = [
  {
    id: "w-titan-84721",
    config: "default",
    agent: "neo",
    agents: [{ name: "neo", tier: "citizen", status: "running" }],
    status: "running",
    created_at: "2026-04-01T16:00:00Z",
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
    created_at: "2026-04-01T15:30:00Z",
    workspace: "~/signews",
  },
  {
    id: "w-ganymede-51003",
    config: "backend",
    agent: "atlas",
    agents: [{ name: "atlas", tier: "citizen", status: "idle" }],
    status: "idle",
    created_at: "2026-04-01T14:00:00Z",
    workspace: "~/infra",
  },
];

export const MOCK_LIMBO: LimboAgent[] = [
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

export const MOCK_PROFILES: Record<string, AgentProfile> = {
  neo: {
    name: "neo",
    tier: "citizen",
    engine: "claude-code",
    provider: "anthropic",
    purpose: "Full-stack developer specializing in API design and implementation",
    persona: "Focused, methodical, prefers clean architecture. Communicates concisely.",
    traits: ["detail-oriented", "async-first", "test-driven", "pragmatic"],
    skills: ["typescript", "node.js", "api-design", "testing", "code-review", "git"],
    playbooks: ["delegate-subtask", "review-pr", "debug-error", "refactor-module"],
    knowledge: ["project-structure.md", "api-patterns.md", "auth-flow.md", "db-schema.md"],
    journal: [
      { date: "2026-04-01", summary: "Completed API endpoints for user management. Refactored auth middleware." },
      { date: "2026-03-31", summary: "Set up project structure. Implemented base models and migrations." },
      { date: "2026-03-30", summary: "Initial world setup. Read through AGENT.md and workspace layout." },
    ],
    bonds: [
      { agent: "morpheus", relationship: "receives guidance from" },
      { agent: "trinity", relationship: "collaborates with" },
    ],
  },
  morpheus: {
    name: "morpheus",
    tier: "governor",
    engine: "claude-code",
    provider: "anthropic",
    purpose: "Project architect and team coordinator. Oversees design decisions and delegates work.",
    persona: "Strategic thinker, clear communicator. Balances speed with quality.",
    traits: ["leadership", "strategic", "mentoring", "systems-thinking"],
    skills: ["architecture", "project-management", "code-review", "delegation", "planning"],
    playbooks: ["plan-sprint", "delegate-task", "review-architecture", "resolve-conflict"],
    knowledge: ["system-design.md", "team-workflows.md", "project-roadmap.md"],
    journal: [
      { date: "2026-04-01", summary: "Delegated frontend tasks to trinity. Reviewed neo's API design." },
      { date: "2026-03-31", summary: "Set up project architecture. Defined module boundaries." },
    ],
    bonds: [
      { agent: "neo", relationship: "mentors" },
      { agent: "trinity", relationship: "coordinates with" },
    ],
  },
  trinity: {
    name: "trinity",
    tier: "citizen",
    engine: "claude-code",
    provider: "anthropic",
    purpose: "Frontend developer specializing in UI/UX implementation",
    persona: "Creative, efficient, pixel-perfect. Focuses on user experience.",
    traits: ["creative", "efficient", "visual-thinker", "accessibility-focused"],
    skills: ["react", "tailwind", "framer-motion", "accessibility", "responsive-design"],
    playbooks: ["build-component", "style-page", "add-animation", "fix-layout"],
    knowledge: ["design-system.md", "component-patterns.md", "accessibility-guide.md"],
    journal: [
      { date: "2026-04-01", summary: "Built landing page components. Added motion transitions." },
    ],
    bonds: [
      { agent: "morpheus", relationship: "reports to" },
      { agent: "neo", relationship: "integrates with" },
    ],
  },
  atlas: {
    name: "atlas",
    tier: "citizen",
    engine: "claude-code",
    provider: "anthropic",
    purpose: "Infrastructure and DevOps specialist. Manages deployments and monitoring.",
    persona: "Reliable, thorough, security-conscious. Documents everything.",
    traits: ["reliable", "security-minded", "thorough", "automation-focused"],
    skills: ["docker", "kubernetes", "terraform", "monitoring", "ci-cd", "networking"],
    playbooks: ["deploy-service", "setup-monitoring", "incident-response", "scale-infra"],
    knowledge: ["infra-topology.md", "deployment-guide.md", "security-policies.md"],
    journal: [
      { date: "2026-04-01", summary: "Configured monitoring dashboards. Set up alerting rules." },
      { date: "2026-03-31", summary: "Initial infrastructure setup. Docker compose for local dev." },
    ],
    bonds: [],
  },
};

export const MOCK_MESSAGES: AgentMessage[] = [
  {
    id: "msg-1",
    from: "morpheus",
    to: "neo",
    content: "The API design looks solid. Please add rate limiting to the auth endpoints before merging.",
    timestamp: "2026-04-01T16:30:00Z",
    read: true,
    channel: "w-europa-39205",
  },
  {
    id: "msg-2",
    from: "neo",
    to: "morpheus",
    content: "Rate limiting added. Using a sliding window approach with Redis. PR is ready for review.",
    timestamp: "2026-04-01T16:45:00Z",
    read: true,
    channel: "w-europa-39205",
  },
  {
    id: "msg-3",
    from: "morpheus",
    to: "trinity",
    content: "The API endpoints are ready. You can start integrating the user management UI. Neo will be available for questions.",
    timestamp: "2026-04-01T17:00:00Z",
    read: false,
    channel: "w-europa-39205",
  },
  {
    id: "msg-4",
    from: "trinity",
    to: "morpheus",
    content: "Got it. Starting with the user list view. Will need the pagination params from neo.",
    timestamp: "2026-04-01T17:05:00Z",
    read: false,
    channel: "w-europa-39205",
  },
  {
    id: "msg-5",
    from: "architect",
    to: "neo",
    content: "World snapshot completed. Your session state has been preserved.",
    timestamp: "2026-04-01T15:00:00Z",
    read: true,
    channel: "system",
  },
];

export const MOCK_SNAPSHOTS: Snapshot[] = [
  {
    id: "snap-titan-001",
    worldId: "w-titan-84721",
    name: "pre-refactor",
    created_at: "2026-04-01T15:00:00Z",
    size: "24 MB",
    agents: 1,
  },
  {
    id: "snap-titan-002",
    worldId: "w-titan-84721",
    name: "api-complete",
    created_at: "2026-04-01T17:30:00Z",
    size: "31 MB",
    agents: 1,
  },
  {
    id: "snap-europa-001",
    worldId: "w-europa-39205",
    name: "initial-setup",
    created_at: "2026-04-01T15:30:00Z",
    size: "18 MB",
    agents: 2,
  },
  {
    id: "snap-europa-002",
    worldId: "w-europa-39205",
    name: "sprint-1-done",
    created_at: "2026-04-01T18:00:00Z",
    size: "42 MB",
    agents: 2,
  },
];

export const MOCK_LOGS: LogEntry[] = [
  { timestamp: "2026-04-01T17:30:12Z", level: "info", source: "architect", message: "Health check passed for w-titan-84721" },
  { timestamp: "2026-04-01T17:29:45Z", level: "info", source: "neo", message: "Task completed: implement-auth-endpoints" },
  { timestamp: "2026-04-01T17:28:30Z", level: "debug", source: "neo", message: "Running test suite: 24 passed, 0 failed" },
  { timestamp: "2026-04-01T17:25:00Z", level: "info", source: "architect", message: "Snapshot created: api-complete" },
  { timestamp: "2026-04-01T17:20:15Z", level: "warn", source: "neo", message: "Memory usage at 78% — consider reflecting" },
  { timestamp: "2026-04-01T17:15:00Z", level: "info", source: "neo", message: "Started task: implement-auth-endpoints" },
  { timestamp: "2026-04-01T17:10:00Z", level: "info", source: "architect", message: "World w-titan-84721 status: running" },
  { timestamp: "2026-04-01T17:05:22Z", level: "error", source: "neo", message: "Build failed: missing dependency @types/bcrypt" },
  { timestamp: "2026-04-01T17:04:00Z", level: "info", source: "neo", message: "Installing dependencies..." },
  { timestamp: "2026-04-01T17:00:00Z", level: "info", source: "architect", message: "Session started for neo in w-titan-84721" },
];

// Available configs for the Create World dialog
export const AVAILABLE_CONFIGS = ["default", "backend", "frontend", "fullstack", "devops", "minimal"];

// Available tiers for the Create Agent dialog
export const AVAILABLE_TIERS = ["governor", "citizen", "npc"] as const;
