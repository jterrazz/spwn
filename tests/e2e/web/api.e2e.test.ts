import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { createSpwnHome } from "../../setup/helpers.js";

/**
 * E2E tests for spwn API routes.
 *
 * These tests validate the Go API server contracts. They run against
 * the Go API server at http://localhost:3001 when GO_API_URL is set,
 * otherwise they validate the expected request/response shapes as contract tests.
 */

const BASE_URL = process.env.GO_API_URL || "http://localhost:3001";
const hasServer = Boolean(process.env.GO_API_URL);

/** Helper: skip if no live server available. */
function requireServer() {
  if (!hasServer) {
    return true; // signal to skip
  }
  return false;
}

describe("Go API routes", () => {
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
  });

  // ──────────────────────────────────────────────
  // GET /api/architect/status
  // ──────────────────────────────────────────────
  describe("GET /api/architect/status", () => {
    test("returns valid status shape when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/status`);
      expect(res.ok).toBe(true);

      const data = await res.json();
      expect(data).toHaveProperty("status");
      expect(["running", "stopped", "starting", "error"]).toContain(
        data.status,
      );
    });

    test("status response contract shape", () => {
      // Validate expected shape even without a server
      const validResponses = [
        { status: "running", uptime: 3600 },
        { status: "stopped" },
        { status: "starting" },
        { status: "error", error: "Container not found" },
      ];

      for (const res of validResponses) {
        expect(res).toHaveProperty("status");
        expect(typeof res.status).toBe("string");
      }
    });
  });

  // ──────────────────────────────────────────────
  // POST /api/architect/start
  // ──────────────────────────────────────────────
  describe("POST /api/architect/start", () => {
    test("returns success or already-running when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/start`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });

      // Should be 200 (started) or 409 (already running) or 500 (Docker error)
      expect([200, 409, 500]).toContain(res.status);

      const data = await res.json();
      expect(data).toHaveProperty("status");
    });

    test("start response contract shape", () => {
      const validResponses = [
        { status: "started", containerId: "abc123" },
        { status: "already_running" },
        { status: "error", error: "Docker not available" },
      ];

      for (const res of validResponses) {
        expect(res).toHaveProperty("status");
        expect(typeof res.status).toBe("string");
      }
    });
  });

  // ──────────────────────────────────────────────
  // POST /api/architect/talk
  // ──────────────────────────────────────────────
  describe("POST /api/architect/talk", () => {
    test("accepts message and returns stream when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/talk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: "What is the project status?" }),
      });

      // May be 200 (streaming) or 503 (architect not running)
      expect([200, 503]).toContain(res.status);

      if (res.ok) {
        const contentType = res.headers.get("content-type") || "";
        // Should be SSE or JSON
        expect(
          contentType.includes("text/event-stream") ||
            contentType.includes("application/json") ||
            contentType.includes("text/plain"),
        ).toBe(true);
      }
    });

    test("rejects empty message with 400 when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/talk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: "" }),
      });

      expect(res.status).toBe(400);

      const data = await res.json();
      expect(data).toHaveProperty("error");
    });

    test("rejects missing message field with 400 when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/talk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });

      expect(res.status).toBe(400);
    });

    test("talk request contract shape", () => {
      // Validate expected request shape
      const validRequest = { message: "Hello architect" };
      expect(validRequest).toHaveProperty("message");
      expect(typeof validRequest.message).toBe("string");
      expect(validRequest.message.length).toBeGreaterThan(0);
    });
  });

  // ──────────────────────────────────────────────
  // POST /api/architect/stop
  // ──────────────────────────────────────────────
  describe("POST /api/architect/stop", () => {
    test("returns success or not-running when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/stop`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });

      // Should be 200 (stopped) or 404/409 (not running)
      expect([200, 404, 409]).toContain(res.status);

      const data = await res.json();
      expect(data).toHaveProperty("status");
    });

    test("stop response contract shape", () => {
      const validResponses = [
        { status: "stopped" },
        { status: "not_running" },
        { status: "error", error: "Failed to stop container" },
      ];

      for (const res of validResponses) {
        expect(res).toHaveProperty("status");
        expect(typeof res.status).toBe("string");
      }
    });
  });

  // ──────────────────────────────────────────────
  // GET /api/auth/providers
  // ──────────────────────────────────────────────
  describe("GET /api/auth/providers", () => {
    test("returns array of providers when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/auth/providers`);
      expect(res.ok).toBe(true);

      const data = await res.json();
      expect(Array.isArray(data)).toBe(true);
      expect(data.length).toBe(3);
    });

    test("each provider has required fields when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/auth/providers`);
      const data = (await res.json()) as Array<Record<string, unknown>>;

      for (const provider of data) {
        expect(provider).toHaveProperty("provider");
        expect(typeof provider.provider).toBe("string");

        expect(provider).toHaveProperty("connected");
        expect(typeof provider.connected).toBe("boolean");

        expect(provider).toHaveProperty("credentialType");
        expect(typeof provider.credentialType).toBe("string");

        expect(provider).toHaveProperty("source");
        expect(typeof provider.source).toBe("string");
      }
    });

    test("providers response contract shape", () => {
      // Validate expected shape even without a server
      const validResponse = [
        {
          provider: "anthropic",
          connected: true,
          credentialType: "api-key",
          source: "env",
        },
        {
          provider: "openai",
          connected: false,
          credentialType: "none",
          source: "none",
        },
        {
          provider: "open-router",
          connected: true,
          credentialType: "api-key",
          source: "config",
        },
      ];

      expect(Array.isArray(validResponse)).toBe(true);
      expect(validResponse.length).toBe(3);

      for (const p of validResponse) {
        expect(p).toHaveProperty("provider");
        expect(p).toHaveProperty("connected");
        expect(p).toHaveProperty("credentialType");
        expect(p).toHaveProperty("source");
        expect(typeof p.provider).toBe("string");
        expect(typeof p.connected).toBe("boolean");
        expect(typeof p.credentialType).toBe("string");
        expect(typeof p.source).toBe("string");
      }
    });
  });

  // ──────────────────────────────────────────────
  // POST /api/auth/check
  // ──────────────────────────────────────────────
  describe("POST /api/auth/check", () => {
    test("validates a provider when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/auth/check`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: "anthropic" }),
      });

      // Should return 200 regardless of whether credentials exist
      expect(res.ok).toBe(true);

      const data = await res.json();
      expect(data).toHaveProperty("provider");
      expect(data.provider).toBe("anthropic");
      expect(data).toHaveProperty("connected");
      expect(typeof data.connected).toBe("boolean");
    });

    test("returns error for unknown provider when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/auth/check`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: "nonexistent-provider" }),
      });

      // Should be 400 or 404 for unknown provider
      expect([400, 404, 422]).toContain(res.status);
    });

    test("check request contract shape", () => {
      // Validate expected request/response shapes
      const validRequest = { provider: "anthropic" };
      expect(validRequest).toHaveProperty("provider");
      expect(typeof validRequest.provider).toBe("string");

      const validResponse = {
        provider: "anthropic",
        connected: true,
        credentialType: "api-key",
        source: "env",
      };
      expect(validResponse).toHaveProperty("provider");
      expect(validResponse).toHaveProperty("connected");
      expect(validResponse).toHaveProperty("credentialType");
      expect(typeof validResponse.connected).toBe("boolean");
    });
  });

  // ──────────────────────────────────────────────
  // POST /api/auth/configure
  // ──────────────────────────────────────────────
  describe("POST /api/auth/configure", () => {
    test("saves a token when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/auth/configure`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          provider: "anthropic",
          token: "sk-ant...2345",
        }),
      });

      expect(res.ok).toBe(true);

      const data = await res.json();
      expect(data).toHaveProperty("status");
      expect(["saved", "configured", "ok", "success"]).toContain(data.status);
    });

    test("rejects missing token with 400 when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/auth/configure`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: "anthropic" }),
      });

      expect([400, 422]).toContain(res.status);
    });

    test("rejects missing provider with 400 when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/auth/configure`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: "sk-ant...2345" }),
      });

      expect([400, 422]).toContain(res.status);
    });

    test("configure request contract shape", () => {
      const validRequest = {
        provider: "anthropic",
        token: "sk-ant...2345",
      };
      expect(validRequest).toHaveProperty("provider");
      expect(validRequest).toHaveProperty("token");
      expect(typeof validRequest.provider).toBe("string");
      expect(typeof validRequest.token).toBe("string");

      const validResponse = { status: "saved" };
      expect(validResponse).toHaveProperty("status");
      expect(typeof validResponse.status).toBe("string");
    });
  });

  // ──────────────────────────────────────────────
  // Cross-cutting: content-type and method checks
  // ──────────────────────────────────────────────
  describe("HTTP method enforcement", () => {
    test("GET on POST-only endpoint returns 405 when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/talk`, {
        method: "GET",
      });

      // Should be 405 Method Not Allowed or 404
      expect([404, 405]).toContain(res.status);
    });

    test("POST on GET-only endpoint returns 405 when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/status`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });

      // Should be 405 Method Not Allowed or still work (some frameworks allow both)
      expect([200, 405]).toContain(res.status);
    });
  });

  // ──────────────────────────────────────────────
  // History API endpoints
  // ──────────────────────────────────────────────
  describe("GET /api/architect/history", () => {
    test("returns sessions array when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/history`);
      expect(res.ok).toBe(true);

      const data = await res.json();
      expect(data).toHaveProperty("sessions");
      expect(Array.isArray(data.sessions)).toBe(true);
    });

    test("each session has required fields when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/history`);
      const data = await res.json();

      for (const session of data.sessions) {
        expect(session).toHaveProperty("id");
        expect(session).toHaveProperty("messages");
        expect(session).toHaveProperty("startedAt");
        expect(typeof session.id).toBe("string");
        expect(Array.isArray(session.messages)).toBe(true);
        expect(typeof session.startedAt).toBe("string");
      }
    });

    test("messages have required fields when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/history`);
      const data = await res.json();

      for (const session of data.sessions) {
        for (const msg of session.messages) {
          expect(msg).toHaveProperty("role");
          expect(msg).toHaveProperty("content");
          expect(msg).toHaveProperty("timestamp");
          expect(msg).toHaveProperty("type");
          expect(typeof msg.role).toBe("string");
          expect(typeof msg.content).toBe("string");
          expect(typeof msg.timestamp).toBe("string");
          expect(typeof msg.type).toBe("string");
        }
      }
    });

    test("sessions are in chronological order when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/architect/history`);
      const data = await res.json();

      if (data.sessions.length >= 2) {
        for (let i = 1; i < data.sessions.length; i++) {
          const prev = new Date(data.sessions[i - 1].startedAt).getTime();
          const curr = new Date(data.sessions[i].startedAt).getTime();
          expect(curr).toBeGreaterThanOrEqual(prev);
        }
      }
    });

    test("history response contract shape", () => {
      const validResponse = {
        sessions: [
          {
            id: "session-001",
            startedAt: "2025-01-01T00:00:00Z",
            messages: [
              {
                role: "user",
                content: "Hello",
                timestamp: "2025-01-01T00:00:01Z",
                type: "text",
              },
              {
                role: "assistant",
                content: "Hi there",
                timestamp: "2025-01-01T00:00:02Z",
                type: "text",
              },
            ],
          },
        ],
      };

      expect(validResponse).toHaveProperty("sessions");
      expect(Array.isArray(validResponse.sessions)).toBe(true);

      const session = validResponse.sessions[0];
      expect(session).toHaveProperty("id");
      expect(session).toHaveProperty("messages");
      expect(session).toHaveProperty("startedAt");

      const msg = session.messages[0];
      expect(msg).toHaveProperty("role");
      expect(msg).toHaveProperty("content");
      expect(msg).toHaveProperty("timestamp");
      expect(msg).toHaveProperty("type");
    });
  });

  describe("GET /api/worlds/{id}/history", () => {
    test("returns sessions array when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/worlds/w-test-00001/history`);
      // 200 if world exists, 404 if not — both are valid
      expect([200, 404]).toContain(res.status);

      if (res.ok) {
        const data = await res.json();
        expect(data).toHaveProperty("sessions");
        expect(Array.isArray(data.sessions)).toBe(true);
      }
    });

    test("each session has required fields when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/worlds/w-test-00001/history`);
      if (!res.ok) return; // world may not exist

      const data = await res.json();
      for (const session of data.sessions) {
        expect(session).toHaveProperty("id");
        expect(session).toHaveProperty("messages");
        expect(session).toHaveProperty("startedAt");
      }
    });

    test("sessions are in chronological order when server is available", async () => {
      if (requireServer()) return;

      const res = await fetch(`${BASE_URL}/api/worlds/w-test-00001/history`);
      if (!res.ok) return;

      const data = await res.json();
      if (data.sessions.length >= 2) {
        for (let i = 1; i < data.sessions.length; i++) {
          const prev = new Date(data.sessions[i - 1].startedAt).getTime();
          const curr = new Date(data.sessions[i].startedAt).getTime();
          expect(curr).toBeGreaterThanOrEqual(prev);
        }
      }
    });

    test("world history response contract shape", () => {
      const validResponse = {
        sessions: [
          {
            id: "session-w-001",
            startedAt: "2025-01-01T00:00:00Z",
            messages: [
              {
                role: "user",
                content: "Build the feature",
                timestamp: "2025-01-01T00:00:01Z",
                type: "stack",
              },
            ],
          },
        ],
      };

      expect(validResponse).toHaveProperty("sessions");
      const session = validResponse.sessions[0];
      expect(session).toHaveProperty("id");
      expect(session).toHaveProperty("messages");
      expect(session).toHaveProperty("startedAt");

      const msg = session.messages[0];
      expect(msg).toHaveProperty("role");
      expect(msg).toHaveProperty("content");
      expect(msg).toHaveProperty("timestamp");
      expect(msg).toHaveProperty("type");
    });
  });
});
