import { expect, test } from '../fixtures/app.js';

test.describe('API health', () => {
    test('API is connected (not showing lock screen)', async ({ page }) => {
        await page.goto('/');
        await page.waitForTimeout(3000);

        await expect(page.getByText('Waiting for Docker')).not.toBeVisible({ timeout: 5000 });
        await expect(page.getByText('API OFFLINE')).not.toBeVisible();
        // The Worlds heading should be visible - confirms the app is past the lock screen
        await expect(page.getByRole('heading', { name: 'Worlds', level: 1 })).toBeVisible();
    });

    test('API version endpoint returns data', async ({ api }) => {
        const version = await api.get<{ current: string }>('/api/version');
        expect(version.current).toBeTruthy();
    });

    test('API templates endpoint returns 5 templates', async ({ api }) => {
        const data = await api.get<{ templates: Array<{ slug: string }> }>('/api/templates');
        expect(data.templates).toHaveLength(5);
        expect(data.templates[0].slug).toBe('startup');
    });

    test('API agents endpoint returns agents', async ({ api }) => {
        await api.installTemplate('matrix');
        const agents = await api.get<Array<{ name: string }>>('/api/agents');
        const names = agents.map((a) => a.name);
        expect(names).toContain('Neo');
    });

    test('Docker is detected as running', async ({ api }) => {
        const docker = await api.get<{ running: boolean }>('/api/system/docker');
        expect(docker.running).toBe(true);
    });
});
