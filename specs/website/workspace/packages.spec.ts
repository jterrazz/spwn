import { button, content, dialog, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const DIFFERS = '/?world=machinefile-differs&machine=jterrazz-macbook-pro&tab=Packages';

describe('packages', () => {
    test('W8 a difference from the file sets what it says beside what the machine has, and what Apply will do', async () => {
        // Given - the Packages tab of This Mac over machinefile-differs, which lacks ripgrep and fd
        const result = await website.visit(DIFFERS, async (visitor) => {
            // When - the tab is read
            // Then - a verdict counts the differences, then one table sets each line
            await visitor.see(heading('2 differences from', { exact: false }));
            const differs = region('Differs from its file');
            await visitor.see(within(differs, heading('Differs from its file 2')));
            await visitor.see(within(differs, content('ripgrep')));
            await visitor.see(within(differs, button('Plan')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'W8 waits for a recording of an extra app, which asks Adopt, Remove or Leave, Leave by default',
    );

    test('W9 Plan lists what would change, and says nothing has changed yet', async () => {
        // Given - the Packages tab of This Mac over machinefile-differs
        const result = await website.visit(DIFFERS, async (visitor) => {
            // When - Plan is pressed
            await visitor.click(within(region('Differs from its file'), button('Plan')));

            // Then - counts, then the list by group, and the line that nothing has changed
            await visitor.see(heading('Plan for jterrazz-macbook-pro'));
            await visitor.see(content('Nothing has changed yet.', { exact: false }));
            const install = region('To install');
            await visitor.see(within(install, content('ripgrep')));
            await visitor.see(within(install, heading('To install 2')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo("W9 waits for the overlay's diff on machinefile.plan, under what is adopted");

    test('W12 Apply on a reviewed plan reports it done', async () => {
        // Given - the plan of This Mac over machinefile-differs, two apps to install
        const result = await website.visit(DIFFERS, async (visitor) => {
            await visitor.click(within(region('Differs from its file'), button('Plan')));
            await visitor.click(button('Apply'));

            // When - the dialog confirms Apply
            await visitor.click(
                within(dialog('Apply the plan of jterrazz-macbook-pro?'), button('Apply')),
            );

            // Then - the plan reads Done
            await visitor.see(content('Done'));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'W12 waits for a recording of each step turning to done in order, of the tab then reading Matches, and of Changes recording what was installed',
    );

    test('W13 a plan without a choice removes nothing', async () => {
        // Given - the plan of This Mac over machinefile-differs
        const result = await website.visit(DIFFERS, async (visitor) => {
            // When - Plan is pressed with nothing chosen
            await visitor.click(within(region('Differs from its file'), button('Plan')));

            // Then - nothing is to be removed, which is never the default
            await visitor.see(content('never the default', { exact: false }));
            await visitor.gone(region('To remove'));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('W13 waits for a recording of an app no file lists');

    test('W14 the apps are grouped by layer, each with its count', async () => {
        // Given - the Packages tab of This Mac over machinefile-differs
        const result = await website.visit(DIFFERS, async (visitor) => {
            // When - Apps is read
            const apps = region('Apps');
            await visitor.see(apps);

            // Then - each layer is a group, in the order they apply, with its apps and versions
            const base = within(apps, region('jterrazz-base'));
            const own = within(apps, region('jterrazz-macbook-pro'));
            await visitor.see(within(base, content('2 apps')));
            await visitor.see(within(base, content('homebrew')));
            await visitor.see(within(own, content('4 apps')));
            await visitor.see(within(own, content('ripgrep')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'W14 window differs: a layer shows its count as a word and no square per app, hollow where one is missing, and a line carries no icon and no rule in plain words',
    );

    test.todo("W15 waits for a call capturing a machine's Packages");

    test('W16 Upgrade now answers Done in the Upgrades section', async () => {
        // Given - the Packages tab of This Mac over machinefile-differs
        const result = await website.visit(DIFFERS, async (visitor) => {
            // When - Upgrade now is pressed
            await visitor.click(within(region('Upgrades'), button('Upgrade now')));

            // Then - the gesture reads Done
            await visitor.see(content('Done'));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'W16 waits for the day and the count of the last run on machinefile.drift, and a Changes line',
    );

    test.todo(
        'W17 waits for whether a machine upgrades by itself, when, and its last run, on machinefile.drift',
    );
});
