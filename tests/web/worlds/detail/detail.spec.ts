import { expect, test } from '../../_fixtures/app.js';

test.describe('World detail', () => {
    test('selecting a planet shows agent details', async ({ app, page }) => {
        await page.goto('/');
        await app.waitForClient();

        await app.selectFirstWorld();

        const panel = page.getByRole('main');
        await expect(panel.getByText('matrix ·')).toBeVisible({ timeout: 5000 });
        await expect(panel.getByText('neo', { exact: true })).toBeVisible();
    });
});
