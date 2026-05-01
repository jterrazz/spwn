import { expect, test } from '../../_fixtures/app.js';

test.describe('World lifecycle (requires Docker)', () => {
    test.beforeEach(async ({ api }) => {
        await api.destroyAll();
        await api.installExample('matrix');
    });

    test('spawn → appears in UI → destroy → disappears', async ({ page, api }) => {
        const result = await api.spawnWorld('matrix', 'Neo');
        const worldId = result.World.id;
        expect(worldId).toMatch(/^world-/);

        await page.goto('/');

        await expect(page.getByRole('button', { name: 'New World' })).toBeVisible({
            timeout: 10_000,
        });

        await api.destroyWorld(worldId);

        await expect
            .poll(
                async () => {
                    const worlds = await api.worlds();
                    return worlds.some((w) => w.id === worldId);
                },
                {
                    timeout: 15_000,
                    message: `world ${worldId} destroyed`,
                },
            )
            .toBe(false);
    });

    test('multi-agent world shows all agents in sidebar', async ({ page, api }) => {
        await api.installExample('startup');
        await api.spawnWorld('startup', undefined, [
            { name: 'ceo', role: 'chief' },
            { name: 'devops', role: 'worker' },
            { name: 'analyst', role: 'worker' },
        ]);

        await page.goto('/');

        await page.getByText('World').first().click();

        await expect(page.getByText('ceo')).toBeVisible({ timeout: 5000 });
        await expect(page.getByText('devops')).toBeVisible();
        await expect(page.getByText('analyst')).toBeVisible();
    });

    test('world detail page loads', async ({ page, api }) => {
        const result = await api.spawnWorld('matrix', 'Neo');
        const worldId = result.World.id;

        await page.goto(`/world/${worldId}`);

        await expect(page.getByText(/Neo|matrix|running|idle/i).first()).toBeVisible({
            timeout: 10_000,
        });
    });
});
