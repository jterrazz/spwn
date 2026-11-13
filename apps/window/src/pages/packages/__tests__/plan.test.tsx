import type { Target } from '@jterrazz/station-api';
import { button, component, content, dialog, heading, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Screen } from '../../../design/index';
import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport } from '../../../engine/replay';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';
import { GestureConfirm } from '../../kit/gestures';
import { PlanView } from '../plan';

const recordingFor = await recordingsOf('machinefile-differs', 'machinefile-kept');

const MAC = 'jterrazz-macbook-pro';

function planOver(world: string, name: string, target?: Target) {
    const client = new Client(replayTransport(recordingFor(world)));
    return (
        <EngineProvider client={client}>
            <Screen>
                <PlanView back={() => undefined} name={name} target={target} />
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
                await visitor.see(heading('To install'));

                // When - Apply is clicked, then confirmed in the dialog calls.json asks
                await visitor.click(button('Apply'));
                await visitor.see(asked(MAC));
                await visitor.click(within(asked(MAC), button('Apply')));

                // Then - the dialog closes and the engine's answer settles the gesture done
                await visitor.gone(asked(MAC));
                await visitor.see(content('Done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a plan with nothing to do is applied the same way', async () => {
        // Given - the plan of This Mac over machinefile-kept, whose file it already holds
        const result = await component.render(
            planOver('machinefile-kept', MAC),
            async (visitor) => {
                await visitor.see(heading('To keep'));

                // When - Apply is clicked and confirmed
                await visitor.click(button('Apply'));
                await visitor.click(within(asked(MAC), button('Apply')));

                // Then - the engine answers done
                await visitor.see(content('Done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("an apply the engine refuses shows the engine's refusal and its hint", async () => {
        // Given - the plan of api over machinefile-differs, api stopped
        const result = await component.render(
            planOver('machinefile-differs', 'api', { name: 'api' }),
            async (visitor) => {
                // When - Apply is clicked and confirmed
                await visitor.click(button('Apply'));
                await visitor.click(within(asked('api'), button('Apply')));

                // Then - the gesture fails on the engine's own hint, never one of the window's, and can be dismissed
                await visitor.see(content('station machines start api'));
                await visitor.see(button('Dismiss'));
                await visitor.gone(content('Done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a cancelled apply sends nothing', async () => {
        // Given - the plan of This Mac over machinefile-differs
        const result = await component.render(
            planOver('machinefile-differs', MAC),
            async (visitor) => {
                await visitor.see(heading('To install'));

                // When - Apply is clicked, then cancelled
                await visitor.click(button('Apply'));
                await visitor.click(within(asked(MAC), button('Cancel')));

                // Then - no answer is shown
                await visitor.gone(asked(MAC));
                await visitor.gone(content('Done'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
