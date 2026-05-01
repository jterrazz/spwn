import { expect, test } from '../../_fixtures/app.js';

test.describe('Header', () => {
    test.beforeEach(async ({ page, app }) => {
        await page.goto('/');
        await app.waitForWorlds();
    });

    test('shows stats buttons', async ({ page }) => {
        await expect(page.locator('button:has-text("WORLDS")')).toBeVisible({ timeout: 5000 });
    });
});
