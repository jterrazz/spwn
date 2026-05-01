import { expect, test } from '../../_fixtures/app.js';

test.describe('World detail', () => {
    test('selecting a planet shows agent details', async ({ page, api }) => {
        await api.installExample('matrix');
        await api.spawnWorld('matrix', 'Neo');
        await page.goto('/');
        await expect(page.getByRole('button', { name: 'New World' })).toBeVisible({
            timeout: 10_000,
        });

        await page.keyboard.press('ArrowRight');

        await expect(page.getByText('Neo').first()).toBeVisible({ timeout: 5000 });
    });
});
