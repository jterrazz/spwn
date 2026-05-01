import { expect, test } from '../../_fixtures/app.js';

test.describe('API health', () => {
    test('API is connected (not showing lock screen)', async ({ page }) => {
        await page.goto('/');

        await expect(page.getByText('Waiting for Docker')).not.toBeVisible({ timeout: 5000 });
        await expect(page.getByText('API OFFLINE')).not.toBeVisible();
        // The Worlds heading should be visible - confirms the app is past the lock screen
        await expect(page.getByRole('heading', { name: 'Worlds', level: 1 })).toBeVisible();
    });

    test('API version endpoint returns data', async ({ api }) => {
        const version = await api.get<{ current: string }>('/api/version');
        expect(version.current).toBeTruthy();
    });

    test('API examples endpoint returns the gallery', async ({ api }) => {
        const data = await api.get<{ examples: Array<{ slug: string }> }>('/api/examples');
        expect(data.examples.length).toBeGreaterThanOrEqual(2);
        expect(data.examples[0].slug).toBe('startup');
        expect(data.examples.map((e) => e.slug)).toContain('matrix');
    });

    test('API agents endpoint returns agents', async ({ api }) => {
        await api.installExample('matrix');
        const agents = await api.get<Array<{ name: string }>>('/api/agents');
        const names = agents.map((a) => a.name);
        expect(names).toContain('Neo');
    });

    test('Docker is detected as running', async ({ api }) => {
        const docker = await api.get<{ running: boolean }>('/api/system/docker');
        expect(docker.running).toBe(true);
    });
});
