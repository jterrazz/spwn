import { expect, test } from '../../_fixtures/app.js';

test.describe('Header', () => {
    test.beforeEach(async ({ page, app }) => {
        await page.goto('/');
        await app.waitForWorlds();
    });

    test('shows stats buttons', async ({ page }) => {
        // Each stat reads "<count> <noun>"; the noun alone also names a
        // sidebar entry, so the count is what tells the two apart.
        for (const noun of ['worlds', 'alive', 'sleeping']) {
            await expect(
                page.getByRole('button', { name: new RegExp(`^\\d+ ${noun}$`) }),
            ).toBeVisible({ timeout: 5000 });
        }
    });
});
