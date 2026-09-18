import { expect, test } from '../../_fixtures/app.js';

test.describe('Agent detail', () => {
    test.beforeEach(async ({ api }) => {
        await api.installExample('matrix');
        await api.installExample('startup');
    });

    test('agent detail page loads directly', async ({ page }) => {
        await page.goto('/agents/Neo');

        await expect(page.getByText('Neo').first()).toBeVisible({ timeout: 5000 });
    });
});
