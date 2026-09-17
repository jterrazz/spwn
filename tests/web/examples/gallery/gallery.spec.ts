import { expect, test } from '../../_fixtures/app.js';

/**
 * The bundled gallery reaches a user two ways, and only one of them
 * is reachable here. `spwn web` inside a PROJECT answers `/api/worlds`
 * with the worlds the manifest declares, so the landing page always
 * has planets to draw and never falls back to the install-a-template
 * cards — that empty state belongs to a home with no project, which
 * this harness (a project, by construction) cannot produce. What the
 * gallery is FOR is proven here instead: the catalog each card would
 * describe, and the install that a click performs.
 */
test.describe('Example gallery', () => {
    test('describes every bundled example', async ({ api }) => {
        const startup = await api.get<{
            agents: string[];
            command: string;
            name: string;
            worlds: string[];
        }>('/api/examples/startup');

        expect(startup.name).toBe('Startup');
        expect(startup.command).toBe('spwn up startup');
        expect(startup.agents).toEqual(['analyst', 'ceo', 'devops']);
        expect(startup.worlds).toEqual(['startup']);
    });

    test('offers one card per bundled example', async ({ api }) => {
        const gallery = await api.get<{ examples: Array<{ name: string; slug: string }> }>(
            '/api/examples',
        );

        expect(gallery.examples.map((example) => example.slug)).toContain('matrix');
        for (const example of gallery.examples) {
            expect(example.name).not.toBe('');
        }
    });

    test('installing an example lands its agents in the home', async ({ api }) => {
        await api.installExample('startup');

        const agents = await api.get<Array<{ name: string }>>('/api/agents');

        expect(agents.map((agent) => agent.name)).toEqual(
            expect.arrayContaining(['analyst', 'ceo', 'devops']),
        );
    });

    test('the declared worlds of the project are the landing page', async ({ app, page }) => {
        await page.goto('/');

        await expect(page.getByRole('heading', { level: 1, name: 'Worlds' })).toBeVisible();
        await expect(app.planet('matrix')).toBeVisible({ timeout: 10_000 });
        await expect(app.planet('startup')).toBeVisible();
    });
});
