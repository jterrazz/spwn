import { test as base, expect, type Page } from '@playwright/test';
import { execSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const API_PORT = 9877;
const API_BASE = `http://localhost:${API_PORT}`;
const BIN = resolve(__dirname, '../../../bin/spwn');

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
            headers: body ? { 'Content-Type': 'application/json' } : {},
            body: body ? JSON.stringify(body) : undefined,
        });
        if (!res.ok) {
            const err = await res.json().catch(() => ({}));
            throw new Error(`POST ${path}: ${res.status} ${(err as any).error ?? ''}`);
        }
        return res.json() as T;
    }

    async delete(path: string): Promise<void> {
        const res = await fetch(`${this.baseUrl}${path}`, { method: 'DELETE' });
        if (!res.ok) {
            throw new Error(`DELETE ${path}: ${res.status}`);
        }
    }

    /** List running worlds */
    async worlds(): Promise<
        Array<{ id: string; status: string; agent: string; agents: Array<{ name: string }> }>
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

    /** Destroy all worlds (cleanup) */
    async destroyAll() {
        const worlds = await this.worlds();
        for (const w of worlds) {
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
    // eslint-disable-next-line no-empty-pattern -- Playwright requires destructured fixtures param
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

export { expect };
