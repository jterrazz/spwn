import type { MachinefilePlan } from '@jterrazz/station-api';
import {
    button,
    component,
    content,
    dialog,
    disabled,
    file,
    heading,
    main,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Screen } from '../../../design/index';
import type { Out } from '../../../engine/client';
import { keyOf } from '../../../engine/frame';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';
import { Differs } from '../differs';
import { PlanSteps } from '../plan';

const recordingFor = await recordingsOf(
    'machinefile-differs',
    'machinefile-kept',
    'mini-offline',
    'one-mac',
    'stories-file',
    'two-macs',
);

function builtInPlan(call: 'machinefile.drift' | 'machinefile.plan'): MachinefilePlan {
    const asked = keyOf({
        call,
        in: { Machinefile: 'station-macos' },
    });
    const exchange = recordingFor('two-macs').exchanges.find((one) => keyOf(one.request) === asked);
    const answer = exchange?.answers.find(
        (one): one is { out: Out<typeof call> } =>
            typeof one === 'object' && one !== null && 'out' in one,
    );
    if (!answer) {
        throw new Error(`two-macs records no answer to ${asked}.`);
    }
    return 'Plan' in answer.out ? answer.out.Plan : answer.out;
}

function windowOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Packages' });
}

describe('Packages', () => {
    test('a Mac with no file of its name keeps an honest head, and its refusal in the body', async () => {
        // Given - the window on This Mac's Packages over one-mac
        const result = await component.render(
            windowOver('one-mac', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its file has been read, and refused
                await visitor.see(
                    content('no Machinefile named jterrazz-macbook-pro.', { exact: false }),
                );
            },
        );

        // Then - the head stays honest, and the refusal with its fix sits in the body
        await expect(result.tree).toMatch(file('./__expected__/one-mac-this-mac.aria.yaml'));
    });

    test('a Mac its applied file holds whole shows no difference, and one whose file asks more shows what it lacks', async () => {
        // Given - This Mac's Packages over machinefile-kept, then over machinefile-differs
        const kept = await component.render(
            windowOver('machinefile-kept', 'jterrazz-macbook-pro'),
            async (visitor) => {
                await visitor.see(content('lazygit'));
            },
        );
        const differs = await component.render(
            windowOver('machinefile-differs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - both have been read to their inventory
                await visitor.see(heading('Differs from its file 2'));
                await visitor.see(within(region('jterrazz-macbook-pro'), content('ripgrep')));
            },
        );

        // Then - the first has no differences panel, the second lists ripgrep and fd to install
        await expect(kept.tree).toMatch(file('./__expected__/machinefile-kept.aria.yaml'));
        await expect(differs.tree).toMatch(file('./__expected__/machinefile-differs.aria.yaml'));
    });

    test("a line its machine lacks is a hollow square of its layer and a chip in its version's place", async () => {
        // Given - This Mac's Packages over machinefile-differs, where ripgrep is not installed
        let square = '';
        let layer = '';
        await component.render(
            windowOver('machinefile-differs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its layer is drawn
                await visitor.see(within(region('jterrazz-macbook-pro'), content('ripgrep')));
                square =
                    document.querySelector('i[title="ripgrep"]')?.getAttribute('data-tone') ?? '';
                layer =
                    document.querySelector('section[aria-label="jterrazz-macbook-pro"]')
                        ?.innerHTML ?? '';
            },
        );

        // Then - its square is hollow in its condition's tone, and the chip says it in the engine's word
        expect(square).toBe('warn');
        expect(layer).toContain('>not installed<');
    });

    test("the file's lines switch to its settings, still grouped by the layer that sets each", async () => {
        // Given - the window on This Mac's Packages over stories-file, whose file sets one setting
        const result = await component.render(
            windowOver('stories-file', 'macbook-pro'),
            async (visitor) => {
                await visitor.see(within(region('macbook-pro'), content('ghostty')));

                // When - Settings is chosen
                await visitor.click(within(main(), button('Settings1', { exact: true })));
                await visitor.see(within(region('macbook-pro'), content('git-shortcuts')));
            },
        );

        // Then - the one setting, under the layer of the file that sets it
        await expect(result.tree).toMatch(file('./__expected__/stories-settings.aria.yaml'));
    });

    test('the name of the file a machine follows opens Machine files', async () => {
        // Given - the window on This Mac's Packages over two-macs
        const result = await component.render(
            windowOver('two-macs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                await visitor.see(content('lazygit'));

                // When - the name of its file is clicked in the head
                await visitor.click(
                    within(main(), button('jterrazz-macbook-pro', { exact: true })),
                );

                // Then - Machine files opens on that file
                await visitor.see(heading('jterrazz-macbook-pro.machine.yaml'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a machine the engine does not answer for shows its refusal', async () => {
        // Given - the window on the Mac that does not answer, over mini-offline
        const result = await component.render(
            windowOver('mini-offline', 'jterrazz-mac-mini'),
            async (visitor) => {
                // When - the tab has heard from its transport
                await visitor.see(heading('No verdict from the engine.'));
            },
        );

        // Then - each read says it was not answered, and nothing is invented in its place
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-mini.aria.yaml'));
    });

    test("what differs from a file says what the machine would do, and the file's side is not given", async () => {
        // Given - the drift of This Mac from the built-in station-macos, read over two-macs
        let planned = false;
        const plan = builtInPlan('machinefile.drift');
        const result = await component.render(
            <Screen>
                <Differs
                    onPlan={() => {
                        planned = true;
                    }}
                    plan={plan}
                />
            </Screen>,
            async (visitor) => {
                // When - the file's side is chosen, then the machine's, and Plan is asked
                await visitor.click(button('Change the file'));
                await visitor.see(disabled(button('Capture')));
                await visitor.click(button('Change the machine'));
                await visitor.click(button('Plan'));
            },
        );

        // Then - each missing item says what the machine would do, and Plan opens the plan
        expect(planned).toBe(true);
        await expect(result.tree).toMatch(file('./__expected__/station-macos-differs.aria.yaml'));
    });

    test('a plan is its steps that change something, grouped by what each does, each from its layer', async () => {
        // Given - the plan of the built-in station-macos for This Mac, read over two-macs
        const plan = builtInPlan('machinefile.plan');

        // When - it is drawn
        const result = await component.render(<PlanSteps plan={plan} />, async (visitor) => {
            await visitor.see(heading('To install 2'));
        });

        // Then - its installs and its settings, each from the file whose layer lists it, what stays left out
        await expect(result.tree).toMatch(file('./__expected__/station-macos-plan.aria.yaml'));
    });

    test('an upgrade starts on the click, with no confirmation, and the engine says it is done', async () => {
        // Given - This Mac's Packages over machinefile-differs
        const result = await component.render(
            windowOver('machinefile-differs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - Upgrade now is clicked
                await visitor.click(button('Upgrade now'));

                // Then - no dialog asks, and the engine's answer settles the gesture done
                await visitor.gone(dialog('Upgrade what the file of jterrazz-macbook-pro lists?'));
                await visitor.see(content('Done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('an upgrade the engine refuses fails, and its refusal can be dismissed', async () => {
        // Given - the Packages of api over machinefile-differs, api stopped
        const result = await component.render(
            windowOver('machinefile-differs', 'api'),
            async (visitor) => {
                await visitor.gone(button('Dismiss'));

                // When - Upgrade now is clicked
                await visitor.click(button('Upgrade now'));

                // Then - the gesture fails on the engine's refusal, never done, and offers to dismiss it
                await visitor.see(button('Dismiss'));
                await visitor.gone(content('Done'));

                // When - it is dismissed
                await visitor.click(button('Dismiss'));

                // Then - the refusal is gone
                await visitor.gone(button('Dismiss'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
