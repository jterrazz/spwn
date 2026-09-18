import { expect, test } from '../../_fixtures/app.js';

test.describe('Command palette', () => {
    test.beforeEach(async ({ page, app }) => {
        await page.goto('/');
        await app.waitForClient();
    });

    test('opens with Cmd+K', async ({ page }) => {
        await page.keyboard.press('Meta+k');

        await expect(page.getByText(/Search for a command/iu)).toBeVisible({ timeout: 3000 });

        await page.keyboard.press('Escape');
        await expect(page.getByText(/Search for a command/iu)).not.toBeVisible();
    });
});
