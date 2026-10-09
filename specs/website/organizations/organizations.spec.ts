import { button, content, field } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('organizations', () => {
    test('the built-in organization is drawn as a chart and cannot be deleted', async () => {
        // Given - a host with only the built-in organization
        const result = await website.visit('/organizations', async (visitor) => {
            await visitor.see(content('Built-in three-tier organization'));
        });

        // Then - its three levels are on the chart, with no button to delete it
        expect(result.content.text).toContain('chief');
        expect(result.content.text).toContain('reports to manager');
        expect(result.content.text).not.toContain('Delete default');
    });

    test('an organization declared through the form joins the page, then is deleted', async () => {
        // Given - the built-in organization alone
        const result = await website.visit('/organizations', async (visitor) => {
            await visitor.see(content('Built-in three-tier organization'));

            // When - the visitor declares a crew of one captain, then deletes it
            await visitor.click(button('New Organization'));
            await visitor.fill(field('Name'), 'Crew');
            await visitor.fill(field('Role 1 name'), 'captain');
            await visitor.click(button('Create'));
            await visitor.see(content('captain'));
            await visitor.click(button('Delete Crew'));
            await visitor.click(button('Delete'));
            await visitor.gone(button('Delete Crew'));
        });

        // Then - only the built-in organization remains
        expect(result.content.text).not.toContain('captain');
    });
});
