import { expect, test } from '../../_fixtures/app.js';

test.describe('Sidebar', () => {
    test.beforeEach(async ({ page, app }) => {
        await page.goto('/');
        await app.waitForWorlds();
    });

    test('shows navigation items', async ({ page }) => {
        for (const name of ['Architect', 'Settings', 'Worlds', 'Agents', 'Tools']) {
            await expect(page.getByRole('button', { name, exact: true }).first()).toBeVisible();
        }
    });

    test('navigation changes page content', async ({ page, app }) => {
        await app.goToAgents();

        await page.getByRole('button', { name: 'Tools', exact: true }).first().click();
        await expect(page.getByRole('heading', { name: 'Tools' })).toBeVisible({ timeout: 5000 });

        await app.goToWorlds();
    });

    test('Docker version is visible', async ({ page }) => {
        await expect(page.getByRole('button', { name: /Docker status/ })).toBeVisible({
            timeout: 5000,
        });
    });
});
