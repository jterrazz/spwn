import {
    banner,
    button,
    component,
    content,
    dialog,
    field,
    file,
    heading,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../composition/compose';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import type { Pace } from '../../../../infrastructure/outbound/engine-replay/replay';
import { EngineProvider } from '../../../dependencies/dependencies';
import type { Route } from '../../../navigation/route';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../../screens/registry';

const recordingFor = await recordingsOf('create-held', 'create-failing', 'create-refused');

type Visitor = Parameters<NonNullable<Parameters<typeof component.render>[1]>>[0];

const HOME: Route = { page: 'home' };
const HELD: Pace = { gestureFrame: 0, held: 'machines.create-workspace' };

const guide = dialog('New machine linux-2');

function windowOver(world: string, pace?: Pace) {
    const dependencies = composeReplay({ pace, recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={HOME} pages={PAGES} />
        </EngineProvider>
    );
}

async function launched(visitor: Visitor) {
    await visitor.click(within(banner(), button('New machine')));
    await visitor.click(within(region('On'), button('macbook-pro')));
    const sheet = dialog('New machine on macbook-pro');
    await visitor.fill(within(sheet, field('What is it for?')), 'Lot shell');
    await visitor.fill(within(sheet, field('Name')), 'linux-2');
    await visitor.click(within(sheet, button('ephemeral')));
    await visitor.click(within(sheet, button('Launch')));
    await visitor.gone(sheet);
}

describe('guide-sheet', () => {
    test('launching hands the sheet to the Guide of the act the engine began, drawn as it serves it', async () => {
        // Given - the window over a world where linux-2's create is held at its boot
        const result = await component.render(windowOver('create-held', HELD), async (visitor) => {
            // When - linux-2 is launched
            await launched(visitor);

            // Then - its Guide is open on the step the engine reports
            await visitor.see(within(guide, heading('Boot now Network next')));
        });
        await expect(result.tree).toMatch(file('./__expected__/launched-working.aria.yaml'));
        expect(result.errors.text).toBe('');
    });

    test('hiding closes the Guide, the act goes on, and Show progress on its machine reopens the same act', async () => {
        // Given - linux-2's Guide open over a create held at its boot
        const result = await component.render(windowOver('create-held', HELD), async (visitor) => {
            await launched(visitor);
            await visitor.see(guide);

            // When - Hide is pressed
            await visitor.click(within(guide, button('Hide')));

            // Then - the Guide is gone and linux-2's page offers it again
            await visitor.gone(guide);
            await visitor.see(heading('linux-2 creating'));

            // When - Show progress is pressed
            await visitor.click(button('Show progress'));

            // Then - the same act is open, at the same step
            await visitor.see(within(guide, heading('Boot now Network next')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a create made reads ready, and its action opens the machine', async () => {
        // Given - linux-2 launched over a world whose create runs to its end
        const result = await component.render(windowOver('create-held'), async (visitor) => {
            await launched(visitor);

            // When - the engine has made it, and its action is pressed
            await visitor.see(
                within(guide, heading('linux-2 is ready. Open it in your terminal.')),
            );
            await visitor.click(within(guide, button('Open linux-2')));

            // Then - the Guide is gone and the machine's page is open
            await visitor.gone(guide);
            await visitor.see(heading('linux-2 running'));
        });
        expect(result.errors.text).toBe('');
    });

    test('a create made says its headline once to assistive tech', async () => {
        // Given - linux-2 launched over a world whose create runs to its end
        const result = await component.render(windowOver('create-held'), async (visitor) => {
            // When - it is launched
            await launched(visitor);
            await visitor.see(
                within(guide, heading('linux-2 is ready. Open it in your terminal.')),
            );
        });

        // Then - the tree holds the Guide with its live region
        await expect(result.tree).toMatch(file('./__expected__/launched-done.aria.yaml'));
    });

    test('a create that failed at its boot reads failed there with what was kept and its two repairs', async () => {
        // Given - linux-2 launched over a world where the hypervisor refuses its boot
        const result = await component.render(windowOver('create-failing'), async (visitor) => {
            // When - it is launched
            await launched(visitor);
            await visitor.see(within(guide, heading('Boot failed')));
        });

        // Then - the tree holds the Guide with the engine's cause, its kept disk, Try again and Delete
        await expect(result.tree).toMatch(file('./__expected__/launched-failed.aria.yaml'));
    });

    test('pressing Try again runs the call the engine named at the machine it named, and leaves for its page', async () => {
        // Given - the failed Guide of linux-2
        const result = await component.render(windowOver('create-failing'), async (visitor) => {
            await launched(visitor);
            await visitor.see(within(guide, heading('Boot failed')));

            // When - Try again is pressed
            await visitor.click(within(guide, button('Try again')));

            // Then - the Guide is gone and linux-2 runs
            await visitor.gone(guide);
            await visitor.see(heading('linux-2 running'));
        });
        expect(result.errors.text).toBe('');
    });

    test('a Try again the engine refuses leaves the Guide open on the refusal, the boot still failed and its repairs still offered', async () => {
        // Given - the failed Guide of linux-2, in a world whose vm_user the engine refuses
        const result = await component.render(windowOver('create-refused'), async (visitor) => {
            await launched(visitor);
            await visitor.see(within(guide, heading('Boot failed')));

            // When - Try again is pressed and the engine refuses it
            await visitor.click(within(guide, button('Try again')));

            // Then - the Guide still shows the act, now on the engine's refusal
            await visitor.see(
                within(
                    guide,
                    content(
                        'vm_user "root" in preferences.json is not a VM user: a lowercase short name (a-z first, then up to 31 of a-z, 0-9, _, . and -, never a dot last), never admin or root',
                    ),
                ),
            );
            await visitor.see(within(guide, button('Try again')));
        });
        await expect(result.tree).toMatch(file('./__expected__/repair-refused.aria.yaml'));
    });

    test('deleting a failed create asks the name the engine gave, in the window’s own dialog', async () => {
        // Given - the failed Guide of linux-2
        const result = await component.render(windowOver('create-failing'), async (visitor) => {
            await launched(visitor);
            await visitor.see(within(guide, heading('Boot failed')));

            // When - Delete linux-2 is pressed
            await visitor.click(within(guide, button('Delete linux-2')));

            // Then - the confirmation asks for linux-2 typed
            await visitor.see(within(dialog('Delete linux-2?'), field('Type linux-2 to confirm')));
        });
        expect(result.errors.text).toBe('');
    });
});
