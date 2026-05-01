import { expect, test } from '../../_fixtures/app.js';

test.describe('Agents list', () => {
    test.beforeEach(async ({ api }) => {
        await api.installExample('matrix');
        await api.installExample('startup');
    });

    test('shows agents list', async ({ page, app }) => {
        await page.goto('/');
        await app.waitForWorlds();
        await app.goToAgents();

        await expect(page.getByText('Neo')).toBeVisible({ timeout: 5000 });
    });

    test('clicking an agent navigates to their detail', async ({ page, app }) => {
        await page.goto('/');
        await app.waitForWorlds();
        await app.goToAgents();

        await page.getByText('Neo').first().click();

        await expect(page).toHaveURL(/agents/, { timeout: 5000 });
    });
});
