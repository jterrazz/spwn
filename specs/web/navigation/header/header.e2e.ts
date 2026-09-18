import { expect, test } from '../../_fixtures/app.js';

test.describe('Header', () => {
    test.beforeEach(async ({ app, page }) => {
        await page.goto('/');
        await app.waitForWorlds();
    });

    test('shows stats buttons', async ({ page }) => {
        // Each stat reads "<count> <noun>"; the noun alone also names a
        // sidebar entry, so the count is what tells the two apart.
        await expect(page.getByRole('button', { name: /^\d+ worlds$/u })).toBeVisible({
            timeout: 5000,
        });
        await expect(page.getByRole('button', { name: /^\d+ alive$/u })).toBeVisible();
        await expect(page.getByRole('button', { name: /^\d+ sleeping$/u })).toBeVisible();
    });
});
