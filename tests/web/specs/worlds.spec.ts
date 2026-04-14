import { expect, test } from '../fixtures/app.js';

test.describe('Worlds page', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/');
        await page.waitForTimeout(2000);
    });

    test('shows the Worlds heading', async ({ page }) => {
        await expect(page.getByRole('heading', { name: 'Worlds', level: 1 })).toBeVisible();
    });

    test('sidebar shows navigation items', async ({ page }) => {
        for (const name of ['Architect', 'Settings', 'Worlds', 'Agents', 'Tools']) {
            await expect(page.getByRole('button', { name, exact: true }).first()).toBeVisible();
        }
    });

    test('shows example gallery or planets depending on state', async ({ page }) => {
        const gallery = page.getByText('Give your agents');
        const template = page.getByText('Pick a template');
        const newWorld = page.getByRole('button', { name: 'New World' });
        await expect(gallery.or(template).or(newWorld)).toBeVisible({ timeout: 10_000 });
    });

    test('shows planets when worlds exist', async ({ page, api }) => {
        await api.installTemplate('matrix');
        await api.spawnWorld('matrix', 'Neo');
        await page.goto('/');
        await page.waitForTimeout(4000);

        await expect(page.getByRole('button', { name: 'New World' })).toBeVisible({
            timeout: 10_000,
        });
    });

    test('selecting a planet shows agent details', async ({ page, api }) => {
        await api.installTemplate('matrix');
        await api.spawnWorld('matrix', 'Neo');
        await page.goto('/');
        await page.waitForTimeout(4000);

        await page.keyboard.press('ArrowRight');
        await page.waitForTimeout(2000);

        // Should see agent name in the detail panel
        await expect(page.getByText('Neo').first()).toBeVisible({ timeout: 5000 });
    });
});
