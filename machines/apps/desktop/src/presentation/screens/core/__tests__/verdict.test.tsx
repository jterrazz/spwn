import {
    button,
    component,
    content,
    file,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import type { ReactNode } from 'react';
import { describe, expect, test } from 'vitest';

import { INSTALL_LINE } from '../../../../domain/core';
import { scriptedCore } from '../../../../infrastructure/outbound/core-scripted/scripted-core';
import { SCRIPTS } from '../../../../infrastructure/outbound/core-scripted/scripts';
import { documentPresence } from '../../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../../infrastructure/outbound/engine-link/client';
import { fittedLink } from '../../../../infrastructure/outbound/engine-replay/fitted';
import { helloOf } from '../../../../infrastructure/outbound/engine-replay/hello';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../../infrastructure/outbound/engine-replay/replay';
import type { Stage } from '../../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../../dependencies/dependencies';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('two-macs');

const rail = navigation('Rail');
const PANEL = region('The engine on this Mac');
const HOME = content('2 things need you.');

function verdictOver(fit: 'app-older' | 'engine-older'): { stage: Stage; tree: ReactNode } {
    const recording = recordingFor('two-macs');
    const stage = stagedReplay(recording, fittedLink(recording, fit));
    const core = scriptedCore(SCRIPTS.update, {
        onStarted: () => {
            stage.restart({ ...helloOf(recording), started_at: 'updated' }, recording);
        },
    });
    const tree = (
        <EngineProvider
            dependencies={{
                core,
                engine: new Client(stage.transport),
                opener: NO_OPENER,
                presence: documentPresence(),
            }}
        >
            <Window initial={{ page: 'home' }} pages={PAGES} />
        </EngineProvider>
    );
    return { stage, tree };
}

describe('verdict', () => {
    test('an engine older than this app is the panel in the one frame, says it in the foot, and is asked nothing', async () => {
        // Given - an engine whose hello names a revision below the oldest accepted
        const { stage, tree } = verdictOver('engine-older');
        const result = await component.render(tree, async (visitor) => {
            // When - the window opens on it
            await visitor.see(within(PANEL, heading('Update station here')));

            // Then - the install line beside Update, and the verdict on the rail's Engine line
            await visitor.see(within(PANEL, content(INSTALL_LINE)));
            await visitor.see(within(rail, content('Engine older than this app')));
        });
        expect(stage.asked()).toStrictEqual([]);
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/fit-engine-older.aria.yaml'));
    });

    test('this app older than the engine is the panel with the window update named', async () => {
        // Given - an engine whose hello names a revision above the newest known
        const { stage, tree } = verdictOver('app-older');
        const result = await component.render(tree, async (visitor) => {
            // When - the window opens on it
            await visitor.see(within(PANEL, heading('Update Spwn')));

            // Then - the verdict on the rail's Engine line
            await visitor.see(within(rail, content('This app older than the engine')));
        });
        expect(stage.asked()).toStrictEqual([]);
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/fit-app-older.aria.yaml'));
    });

    test('pressing Update restarts a fitting engine, and its pages are drawn in the main the panel stood in', async () => {
        // Given - the verdict panel over an older engine
        const { tree } = verdictOver('engine-older');
        const mains: (Element | null)[] = [];
        const result = await component.render(tree, async (visitor) => {
            await visitor.see(PANEL);
            mains.push(document.querySelector('main'));

            // When - Update is pressed and the engine comes back fitting
            await visitor.click(within(PANEL, button('Update')));

            // Then - Home is drawn, and the foot no longer says the verdict
            await visitor.see(HOME);
            await visitor.gone(within(rail, content('Engine older than this app')));
            mains.push(document.querySelector('main'));
        });
        expect(mains[0]).toBeInstanceOf(HTMLElement);
        expect(mains[1]).toBe(mains[0]);
        expect(result.errors.text).toBe('');
    });
});
