import type { Target } from '@jterrazz/station-api';
import { button, component, content, dialog, heading, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../../composition/__tests__/page-over';
import { composeReplay } from '../../../../../composition/compose';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../../../dependencies/dependencies';
import { GestureConfirm } from '../../../../kit/gestures';
import { Screen } from '../../../../ui/index';
import { PlanView } from '../plan';

const recordingFor = await recordingsOf(
    'machinefile-differs',
    'machinefile-extra',
    'machinefile-kept',
);

const MAC = 'macbook-pro';

function planOver(world: string, name: string, target?: Target) {
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Screen>
                <PlanView
                    asked={target === undefined ? {} : { Machine: name }}
                    back={() => undefined}
                    name={name}
                    target={target}
                />
            </Screen>
            <GestureConfirm />
        </EngineProvider>
    );
}

const asked = (name: string) => dialog(`Apply the plan of ${name}?`);

describe('Plan', () => {
    test('a plan is applied once confirmed, and the engine says it is done', async () => {
        // Given - This Mac's Packages over machinefile-differs, its plan opened
        const result = await component.render(
            pageOver(recordingFor('machinefile-differs'), {
                machine: MAC,
                page: 'machine',
                tab: 'Packages',
            }),
            async (visitor) => {
                await visitor.click(button('Plan'));
                await visitor.see(heading('To install 2'));

                // When - Apply is clicked, then confirmed in the dialog confirms.json asks
                await visitor.click(button('Apply'));
                await visitor.see(asked(MAC));
                await visitor.click(within(asked(MAC), button('Apply')));

                // Then - the dialog closes and the engine's answer settles the gesture done
                await visitor.gone(asked(MAC));
                await visitor.see(content('done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("the closing bar says what an apply leaves, in the engine's words, beside Back and Apply", async () => {
        // Given - the plan of This Mac over machinefile-differs, whose drift recording gives Apply.Stays
        const result = await component.render(
            planOver('machinefile-differs', MAC),
            async (visitor) => {
                await visitor.see(heading('To install 2'));

                // Then - the bar carries the sentence the engine words, and the two buttons
                await visitor.see(content('Nothing is removed. 4 apps stay as they are.'));
                await visitor.see(button('Back'));
                await visitor.see(button('Apply'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('the closing bar names the extras an apply leaves, only when the engine says some', async () => {
        // Given - the plan of This Mac over machinefile-extra, whose file no longer lists bat
        const extra = await component.render(
            planOver('machinefile-extra', MAC),
            async (visitor) => {
                // Then - the bar says which extra stays, beside what stays
                await visitor.see(content('bat stays: Apply leaves extras.'));
                await visitor.see(content('Nothing is removed. 3 apps stay as they are.'));
            },
        );
        expect(extra.errors.text).toBe('');

        // Given - the plan of This Mac over machinefile-kept, which leaves none
        const kept = await component.render(planOver('machinefile-kept', MAC), async (visitor) => {
            await visitor.see(content('Nothing is removed.', { exact: false }));

            // Then - no extras sentence is drawn
            await visitor.gone(content('Apply leaves extras.', { exact: false }));
        });
        expect(kept.errors.text).toBe('');
    });

    test('a plan with nothing to do is applied the same way', async () => {
        // Given - the plan of This Mac over machinefile-kept, whose file it already holds
        const result = await component.render(
            planOver('machinefile-kept', MAC),
            async (visitor) => {
                await visitor.see(content('apps as declared'));
                await visitor.gone(heading('To install', { exact: false }));

                // When - Apply is clicked and confirmed
                await visitor.click(button('Apply'));
                await visitor.click(within(asked(MAC), button('Apply')));

                // Then - the engine answers done
                await visitor.see(content('done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("an apply the engine refuses shows the engine's refusal, with no command", async () => {
        // Given - the plan of api over machinefile-differs, api stopped
        const result = await component.render(
            planOver('machinefile-differs', 'api', { name: 'api' }),
            async (visitor) => {
                // When - Apply is clicked and confirmed
                await visitor.click(button('Apply'));
                await visitor.click(within(asked('api'), button('Apply')));

                // Then - the gesture fails with the engine's words, its hint that is a command never drawn, and can be dismissed
                await visitor.see(heading('api is stopped.'));
                await visitor.gone(content('station machines start api'));
                await visitor.see(button('Dismiss'));
                await visitor.gone(content('done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a cancelled apply sends nothing', async () => {
        // Given - the plan of This Mac over machinefile-differs
        const result = await component.render(
            planOver('machinefile-differs', MAC),
            async (visitor) => {
                await visitor.see(heading('To install 2'));

                // When - Apply is clicked, then cancelled
                await visitor.click(button('Apply'));
                await visitor.click(within(asked(MAC), button('Cancel')));

                // Then - no answer is shown
                await visitor.gone(asked(MAC));
                await visitor.gone(content('done'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
