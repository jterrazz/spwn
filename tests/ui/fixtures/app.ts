import { test as base, expect, type Page } from "@playwright/test";
import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";

/**
 * Test config written by global-setup.
 */
interface TestConfig {
  apiPort: number;
  spwnHome: string;
  bin: string;
}

function loadConfig(): TestConfig {
  const configPath = process.env.SPWN_TEST_CONFIG;
  if (!configPath) {
    throw new Error("SPWN_TEST_CONFIG env var not set — global-setup did not run?");
  }
  return JSON.parse(readFileSync(configPath, "utf-8"));
}

/**
 * API helper — calls the Go API directly (faster than UI for setup).
 */
class SpwnAPI {
  constructor(private baseUrl: string, private config: TestConfig) {}

  async get<T = unknown>(path: string): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`);
    if (!res.ok) throw new Error(`GET ${path}: ${res.status}`);
    return res.json() as T;
  }

  async post<T = unknown>(path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: body ? { "Content-Type": "application/json" } : {},
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(`POST ${path}: ${res.status} ${(err as any).error ?? ""}`);
    }
    return res.json() as T;
  }

  async delete(path: string): Promise<void> {
    const res = await fetch(`${this.baseUrl}${path}`, { method: "DELETE" });
    if (!res.ok) throw new Error(`DELETE ${path}: ${res.status}`);
  }

  /** List running worlds */
  async worlds(): Promise<Array<{ id: string; status: string; agent: string; agents: Array<{ name: string }> }>> {
    return this.get("/api/worlds");
  }

  /** Install a bundled example */
  async installExample(slug: string) {
    return this.post(`/api/examples/${slug}/install`);
  }

  /** Spawn a world */
  async spawnWorld(config: string, agent?: string, agents?: Array<{ name: string; role: string }>) {
    const body: Record<string, unknown> = { config };
    if (agent) body.agent = agent;
    if (agents) body.agents = agents;
    return this.post<{ Universe: { id: string } }>("/api/worlds", body);
  }

  /** Destroy a world */
  async destroyWorld(id: string) {
    return this.delete(`/api/worlds/${id}`);
  }

  /** Destroy all worlds (cleanup) */
  async destroyAll() {
    const worlds = await this.worlds();
    for (const w of worlds) {
      try { await this.destroyWorld(w.id); } catch { /* ignore */ }
    }
  }

  /** Run a CLI command via the binary */
  cli(args: string): string {
    return execSync(`${this.config.bin} ${args}`, {
      env: { ...process.env, SPWN_HOME: this.config.spwnHome },
      encoding: "utf-8",
      timeout: 30_000,
    });
  }
}

/**
 * Page helpers for common UI interactions.
 */
class SpwnPage {
  constructor(private page: Page) {}

  /** Navigate to the Worlds page */
  async goToWorlds() {
    await this.page.getByRole("button", { name: "Worlds" }).click();
    await this.page.waitForTimeout(500);
  }

  /** Navigate to the Agents page */
  async goToAgents() {
    await this.page.getByRole("button", { name: "Agents" }).click();
    await this.page.waitForTimeout(500);
  }

  /** Navigate to Settings */
  async goToSettings() {
    await this.page.getByRole("button", { name: "Settings" }).click();
    await this.page.waitForTimeout(500);
  }

  /** Click a planet/world in the carousel by name */
  async selectWorld(name: string) {
    // The planet label in the carousel
    await this.page.getByRole("button", { name }).click();
    await this.page.waitForTimeout(1000);
  }

  /** Wait for worlds to load (not skeleton) */
  async waitForWorlds() {
    // Wait until either planets appear or the empty gallery shows
    await this.page.waitForSelector(
      '[role="button"]:has-text("Enter World"), :has-text("Pick a template")',
      { timeout: 15_000 },
    ).catch(() => {});
    await this.page.waitForTimeout(500);
  }

  /** Click "Enter World" button in the selection panel */
  async enterWorld() {
    await this.page.getByRole("link", { name: /Enter World/i }).click();
    await this.page.waitForTimeout(1000);
  }

  /** Press Escape to deselect */
  async deselect() {
    await this.page.keyboard.press("Escape");
    await this.page.waitForTimeout(500);
  }

  /** Open command palette */
  async openCommandPalette() {
    await this.page.keyboard.press("Meta+k");
    await this.page.waitForTimeout(300);
  }
}

/**
 * Extended test fixture with API + page helpers.
 */
export const test = base.extend<{
  api: SpwnAPI;
  app: SpwnPage;
}>({
  api: async ({}, use) => {
    const config = loadConfig();
    const api = new SpwnAPI(`http://localhost:${config.apiPort}`, config);
    await use(api);
    // Cleanup: destroy all worlds created during the test
    await api.destroyAll().catch(() => {});
  },

  app: async ({ page }, use) => {
    const app = new SpwnPage(page);
    await use(app);
  },
});

export { expect };
