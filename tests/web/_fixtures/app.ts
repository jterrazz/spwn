import { test as base, expect, type Page } from '@playwright/test';
import { execSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const currentDir = dirname(fileURLToPath(import.meta.url));
const API_PORT = 9877;
const API_BASE = `http://localhost:${API_PORT}`;
const BIN = resolve(currentDir, '../../../.artifacts/go/spwn');

/**
 * API helper - calls the Go API directly (faster than UI for setup).
 */
class SpwnAPI {
    private baseUrl: string;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl;
    }

    async get<T = unknown>(path: string): Promise<T> {
        const res = await fetch(`${this.baseUrl}${path}`);
        if (!res.ok) {
            throw new Error(`GET ${path}: ${res.status}`);
        }
        return res.json() as T;
    }

    async post<T = unknown>(path: string, body?: unknown): Promise<T> {
        const res = await fetch(`${this.baseUrl}${path}`, {
            method: 'POST',
            // A bodyless POST carries neither a body nor a content type: under
            // `exactOptionalPropertyTypes` an explicit `undefined` is not the
            // same as an absent key.
            ...(body === undefined
                ? {}
                : {
                      body: JSON.stringify(body),
                      headers: { 'Content-Type': 'application/json' },
                  }),
        });
        if (!res.ok) {
            const err: unknown = await res.json().catch(() => ({}));
            const detail =
                typeof err === 'object' && err !== null && 'error' in err ? String(err.error) : '';
            throw new Error(`POST ${path}: ${res.status} ${detail}`);
        }
        return res.json() as T;
    }

    async delete(path: string): Promise<void> {
        const res = await fetch(`${this.baseUrl}${path}`, { method: 'DELETE' });
        if (!res.ok) {
            throw new Error(`DELETE ${path}: ${res.status}`);
        }
    }

    /**
     * List the project's worlds. In a project (which the web e2e
     * harness always is) the API answers with the worlds the manifest
     * DECLARES — known by `name`, carrying their agents as plain
     * names, and `stopped` until a container backs them. Only a world
     * that runs has an `id`.
     */
    async worlds(): Promise<
        Array<{
            agents: Array<string | { name: string }>;
            id?: string;
            name?: string;
            status: string;
        }>
    > {
        return this.get('/api/worlds');
    }

    /** Install a bundled example */
    async installExample(slug: string) {
        return this.post(`/api/examples/${slug}/install`);
    }

    /** Spawn a world */
    async spawnWorld(
        config: string,
        agent?: string,
        agents?: Array<{ name: string; role: string }>,
    ) {
        const body: Record<string, unknown> = { config };
        if (agent) {
            body.agent = agent;
        }
        if (agents) {
            body.agents = agents;
        }
        return this.post<{ World: { id: string } }>('/api/worlds', body);
    }

    /** Destroy a world */
    async destroyWorld(id: string) {
        return this.delete(`/api/worlds/${id}`);
    }

    /** Destroy every world that runs — a declared one has nothing to destroy. */
    async destroyAll() {
        const worlds = await this.worlds();
        for (const w of worlds) {
            if (w.id === undefined) {
                continue;
            }
            try {
                await this.destroyWorld(w.id);
            } catch {
                /* Ignore */
            }
        }
    }

    /** Run a CLI command via the binary */
    cli(args: string): string {
        return execSync(`${BIN} ${args}`, {
            encoding: 'utf8',
            timeout: 30_000,
        });
    }
}

/**
 * Page helpers for common UI interactions.
 */
class SpwnPage {
    private page: Page;

    constructor(page: Page) {
        this.page = page;
    }

    /** Navigate to the Worlds page */
    async goToWorlds() {
        await this.page.getByRole('button', { name: 'Worlds' }).click();
        await expect(this.page.getByRole('heading', { name: 'Worlds', level: 1 })).toBeVisible();
    }

    /** Navigate to the Agents page */
    async goToAgents() {
        await this.page.getByRole('button', { name: 'Agents' }).click();
        await expect(this.page.getByRole('heading', { name: 'Agents' })).toBeVisible();
    }

    /** Navigate to Settings */
    async goToSettings() {
        await this.page.getByRole('button', { name: 'Settings' }).click();
        await expect(this.page.getByRole('heading', { name: /Settings|Providers/ })).toBeVisible();
    }

    /** Click a planet/world in the carousel by name */
    async selectWorld(name: string) {
        await this.page.getByRole('button', { name }).click();
        await expect(this.page.getByText(name).first()).toBeVisible();
    }

    /**
     * Wait until the client has taken over the page. The Docker pill
     * reads `checking` in the server-rendered markup and only carries
     * a version once the browser's own fetch has come back — the first
     * moment an effect-bound handler, like the ⌘K listener, is
     * attached.
     */
    async waitForClient() {
        await expect(this.page.getByRole('button', { name: /Docker status: v/ })).toBeVisible({
            timeout: 15_000,
        });
    }

    /**
     * The planet a world is drawn as. The sidebar carries a world
     * switcher of the same name, so the carousel is reached through
     * the main region.
     */
    planet(name: string) {
        return this.page.getByRole('main').getByRole('button', { exact: true, name });
    }

    /**
     * Select the first world of the carousel. Arrow keys are the
     * selection gesture the page binds to the window; the worlds
     * arrive in name order, so walking them is deterministic.
     */
    async selectFirstWorld() {
        await this.page.keyboard.press('ArrowRight');
    }

    /** Walk the carousel one world further. */
    async selectNextWorld() {
        await this.page.keyboard.press('ArrowRight');
    }

    /** Wait for worlds to load (not skeleton) */
    async waitForWorlds() {
        const enterWorld = this.page.getByRole('link', { name: /Enter World/i });
        const pickTemplate = this.page.getByText('Pick a template');
        const newWorld = this.page.getByRole('button', { name: 'New World' });
        await expect(enterWorld.or(pickTemplate).or(newWorld)).toBeVisible({ timeout: 15_000 });
    }

    /** Click "Enter World" button in the selection panel */
    async enterWorld() {
        await this.page.getByRole('link', { name: /Enter World/i }).click();
        await expect(this.page).toHaveURL(/world\//);
    }

    /** Press Escape to deselect */
    async deselect() {
        await this.page.keyboard.press('Escape');
        await expect(this.page.getByRole('link', { name: /Enter World/i })).not.toBeVisible();
    }

    /** Open command palette */
    async openCommandPalette() {
        await this.page.keyboard.press('Meta+k');
        await expect(this.page.getByText(/Search for a command/i)).toBeVisible();
    }
}

/**
 * Extended test fixture with API + page helpers.
 */
export const test = base.extend<{
    api: SpwnAPI;
    app: SpwnPage;
}>({
    // oxlint-disable-next-line no-empty-pattern -- Playwright only injects a fixture when the worker function destructures its first argument
    api: async ({}, use) => {
        const api = new SpwnAPI(API_BASE);
        await use(api);
        // Cleanup: destroy all worlds created during the test
        await api.destroyAll().catch(() => {});
    },

    app: async ({ page }, use) => {
        const app = new SpwnPage(page);
        await use(app);
    },
});

export { expect } from '@playwright/test';
