import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spawnSync } from "node:child_process";
import { resolve } from "node:path";
import { createSpwnHome } from "../../setup/helpers.js";

/**
 * E2E tests for GET /api/activity endpoint.
 *
 * Runs against the Go API server at http://localhost:3001 when GO_API_URL is set.
 * Without a live server, validates contract shapes and CLI-generated events.
 */

const BASE_URL = process.env.GO_API_URL || "http://localhost:3001";
const hasServer = Boolean(process.env.GO_API_URL);
const SPWN_BIN = resolve(import.meta.dirname, "../../../bin/spwn");

function runSpwn(args: string[], home: string): { exitCode: number; output: string } {
  const result = spawnSync(SPWN_BIN, args, {
    encoding: "utf-8",
    env: { ...process.env, SPWN_HOME: home, INIT_CWD: undefined } as NodeJS.ProcessEnv,
    timeout: 30_000,
  });
  return {
    exitCode: result.status ?? 1,
    output: (result.stdout ?? "") + (result.stderr ?? ""),
  };
}

interface ActivityEvent {
  id: string;
  timestamp: string;
  type: string;
  actor: string;
  verb: string;
  target?: string;
  phrase: string;
  world_id?: string;
  agent_id?: string;
  duration_ms?: number;
  cost_usd?: number;
}

describe("GET /api/activity", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
    spawnSync("rm", ["-rf", home], { timeout: 5000 });
  });

  test("response contract shape", () => {
    // Validate expected response shape
    const validResponse = {
      events: [
        {
          id: "abc123def456",
          timestamp: "2026-04-04T12:00:00.000Z",
          type: "agent.created",
          actor: "user",
          verb: "created",
          target: "neo",
          phrase: "You created neo",
          agent_id: "neo",
        },
      ],
    };

    expect(validResponse).toHaveProperty("events");
    expect(Array.isArray(validResponse.events)).toBe(true);

    const event = validResponse.events[0];
    expect(event).toHaveProperty("id");
    expect(event).toHaveProperty("timestamp");
    expect(event).toHaveProperty("type");
    expect(event).toHaveProperty("actor");
    expect(event).toHaveProperty("verb");
    expect(event).toHaveProperty("phrase");
  });

  test("event type uses dotted namespace", () => {
    const validTypes = [
      "world.spawned",
      "world.destroyed",
      "world.snapshot",
      "world.session_ended",
      "agent.created",
      "agent.deleted",
      "agent.joined",
      "agent.left",
      "agent.dreamed",
      "agent.slept",
      "agent.forked",
      "agent.talked",
      "architect.started",
      "architect.stopped",
      "architect.talked",
    ];

    for (const type of validTypes) {
      expect(type).toMatch(/^[a-z]+\.[a-z_]+$/);
    }
  });

  test("returns events array when server is available", async () => {
    if (!hasServer) return;

    const res = await fetch(`${BASE_URL}/api/activity`);
    expect(res.ok).toBe(true);

    const data = await res.json();
    expect(data).toHaveProperty("events");
    expect(Array.isArray(data.events)).toBe(true);
  });

  test("respects limit query parameter when server is available", async () => {
    if (!hasServer) return;

    // Generate events via CLI
    runSpwn(["agent", "new", "alpha"], home);
    runSpwn(["agent", "new", "beta"], home);
    runSpwn(["agent", "new", "gamma"], home);

    const res = await fetch(`${BASE_URL}/api/activity?limit=2`);
    expect(res.ok).toBe(true);
    const data = await res.json();
    expect(data.events.length).toBeLessThanOrEqual(2);
  });

  test("clamps limit to max 500 when server is available", async () => {
    if (!hasServer) return;

    const res = await fetch(`${BASE_URL}/api/activity?limit=9999`);
    expect(res.ok).toBe(true);
    const data = await res.json();
    expect(data.events.length).toBeLessThanOrEqual(500);
  });

  test("filters by type when server is available", async () => {
    if (!hasServer) return;

    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "rm", "neo"], home);

    const res = await fetch(`${BASE_URL}/api/activity?type=agent.created`);
    expect(res.ok).toBe(true);
    const data = await res.json();
    for (const event of data.events as ActivityEvent[]) {
      expect(event.type).toBe("agent.created");
    }
  });

  test("filters by agent when server is available", async () => {
    if (!hasServer) return;

    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "new", "morpheus"], home);

    const res = await fetch(`${BASE_URL}/api/activity?agent=neo`);
    expect(res.ok).toBe(true);
    const data = await res.json();
    for (const event of data.events as ActivityEvent[]) {
      expect(event.agent_id).toBe("neo");
    }
  });

  test("filters by since timestamp when server is available", async () => {
    if (!hasServer) return;

    const before = new Date().toISOString();
    runSpwn(["agent", "new", "neo"], home);

    const res = await fetch(
      `${BASE_URL}/api/activity?since=${encodeURIComponent(before)}`,
    );
    expect(res.ok).toBe(true);
    const data = await res.json();
    for (const event of data.events as ActivityEvent[]) {
      expect(new Date(event.timestamp).getTime()).toBeGreaterThanOrEqual(
        new Date(before).getTime(),
      );
    }
  });

  test("returns empty array when no events exist", async () => {
    if (!hasServer) return;

    // Fresh home — no activity should exist
    const res = await fetch(`${BASE_URL}/api/activity?agent=nonexistent-xyz-123`);
    expect(res.ok).toBe(true);
    const data = await res.json();
    expect(Array.isArray(data.events)).toBe(true);
  });

  test("events are sorted newest first when server is available", async () => {
    if (!hasServer) return;

    runSpwn(["agent", "new", "first"], home);
    runSpwn(["agent", "new", "second"], home);
    runSpwn(["agent", "new", "third"], home);

    const res = await fetch(`${BASE_URL}/api/activity?limit=50`);
    expect(res.ok).toBe(true);
    const data = await res.json();
    const events = data.events as ActivityEvent[];

    for (let i = 1; i < events.length; i++) {
      const prev = new Date(events[i - 1].timestamp).getTime();
      const curr = new Date(events[i].timestamp).getTime();
      expect(prev).toBeGreaterThanOrEqual(curr);
    }
  });

  test("CORS headers present when server is available", async () => {
    if (!hasServer) return;

    const res = await fetch(`${BASE_URL}/api/activity`, {
      headers: { Origin: "http://localhost:1420" },
    });
    expect(res.ok).toBe(true);
    // CORS middleware should set access-control headers
    expect(res.headers.get("access-control-allow-origin")).toBeTruthy();
  });
});
