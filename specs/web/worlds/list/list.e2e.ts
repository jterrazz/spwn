import { expect, test } from '../../_fixtures/app.js';

test.describe('Worlds list', () => {
    test.beforeEach(async ({ page, app }) => {
        await page.goto('/');
        await app.waitForWorlds();
    });

    test('shows the Worlds heading', async ({ page }) => {
        await expect(page.getByRole('heading', { name: 'Worlds', level: 1 })).toBeVisible();
    });

    test('shows example gallery or planets depending on state', async ({ page }) => {
        const gallery = page.getByText('Give your agents');
        const template = page.getByText('Pick a template');
        const newWorld = page.getByRole('button', { name: 'New World' });
        await expect(gallery.or(template).or(newWorld)).toBeVisible({ timeout: 10_000 });
    });

    test('shows planets when worlds exist', async ({ page, api }) => {
        await api.installExample('matrix');
        await api.spawnWorld('matrix', 'neo');
        await page.goto('/');

        await expect(page.getByRole('button', { name: 'New World' })).toBeVisible({
            timeout: 10_000,
        });
    });
});
