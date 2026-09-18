import { expect, test } from '../../_fixtures/app.js';

test.describe('World lifecycle (requires Docker)', () => {
    test.beforeEach(async ({ api }) => {
        // The first spawn of a run builds the world's image, which
        // outlasts the default per-test minute.
        test.slow();
        await api.destroyAll();
        await api.installExample('matrix');
    });

    test('spawn → the world runs in the UI → destroy → it is stopped again', async ({
        api,
        app,
        page,
    }) => {
        const result = await api.spawnWorld('matrix', 'neo');
        const worldId = result.World.id;
        expect(worldId).toMatch(/^world-/u);

        await page.goto('/');
        await app.waitForClient();
        await app.selectFirstWorld();

        await expect(page.getByRole('main').getByText('matrix · running')).toBeVisible({
            timeout: 10_000,
        });

        await api.destroyWorld(worldId);

        // A declared world never leaves the list — destroying it puts
        // the container away and the world back to `stopped`.
        await expect
            .poll(
                async () => {
                    const worlds = await api.worlds();
                    return worlds.find((w) => w.name === 'matrix')?.status;
                },
                {
                    message: 'matrix status after destroy',
                    timeout: 15_000,
                },
            )
            .toBe('stopped');
    });

    test('multi-agent world shows all agents in sidebar', async ({ api, app, page }) => {
        await api.installExample('startup');
        // `devops` carries spwn:docker-cli, whose probe (`command -v
        // docker`) the mock base image cannot answer — two agents are
        // enough to prove a roster.
        await api.spawnWorld('startup', undefined, [
            { name: 'ceo', role: 'chief' },
            { name: 'analyst', role: 'worker' },
        ]);

        await page.goto('/');
        await app.waitForClient();
        // Worlds arrive in name order, so `startup` is the second one
        // the carousel walks onto.
        await app.selectFirstWorld();
        await app.selectNextWorld();

        const panel = page.getByRole('main');
        await expect(panel.getByText('startup ·')).toBeVisible({ timeout: 5000 });
        await expect(panel.getByText('ceo', { exact: true })).toBeVisible();
        await expect(panel.getByText('analyst', { exact: true })).toBeVisible();
    });

    test('world detail page loads', async ({ api, page }) => {
        const result = await api.spawnWorld('matrix', 'neo');
        const worldId = result.World.id;

        await page.goto(`/world/${worldId}`);

        await expect(page.getByText(/Neo|matrix|running|idle/iu).first()).toBeVisible({
            timeout: 10_000,
        });
    });
});
