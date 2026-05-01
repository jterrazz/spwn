import { expect, test } from '../../_fixtures/app.js';

test.describe('Command palette', () => {
    test.beforeEach(async ({ page, app }) => {
        await page.goto('/');
        await app.waitForWorlds();
    });

    test('opens with Cmd+K', async ({ page }) => {
        await page.keyboard.press('Meta+k');

        await expect(page.getByText(/Search for a command/i)).toBeVisible({ timeout: 3000 });

        await page.keyboard.press('Escape');
        await expect(page.getByText(/Search for a command/i)).not.toBeVisible();
    });
});
