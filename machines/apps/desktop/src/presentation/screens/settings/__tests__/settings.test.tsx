import { button, component, content, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../composition/compose';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import {
    menuBarScriptOf,
    scriptedMenuBar,
} from '../../../../infrastructure/outbound/menu-bar-scripted/scripted-menu-bar';
import { EngineProvider } from '../../../dependencies/dependencies';
import type { Route } from '../../../navigation/route';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('stories');

const COMMAND_COMMA = 'Meta>},{/Meta';
const engine = region('Engine');

function windowOn(initial: Route, menuBar?: string) {
    const script = menuBar === undefined ? undefined : menuBarScriptOf(menuBar);
    const dependencies = composeReplay({
        menuBar: script === undefined ? undefined : scriptedMenuBar(script),
        recording: recordingFor('stories'),
    });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

describe('Settings', () => {
    test('⌘, opens the app’s Settings, which names its engine and opens the Engine page', async () => {
        // Given - the window on Home over stories
        const result = await component.render(windowOn({ page: 'home' }), async (visitor) => {
            await visitor.see(content('2 things need you.'));

            // When - ⌘, is pressed
            await visitor.press(COMMAND_COMMA);

            // Then - Settings holds the engine's line, whose Open leads to the Engine page
            await visitor.see(within(engine, content('Engine running · v4.0.0')));
            await visitor.click(within(engine, button('Open')));
            await visitor.see(heading('The latest release is not known yet.'));
        });
        expect(result.errors.text).toBe('');
    });

    test('the menu bar’s switch on Settings holds what macOS holds, and a press switches it', async () => {
        // Given - Settings over a menu bar shown at every login
        const result = await component.render(
            windowOn({ page: 'settings' }, 'shown'),
            async (visitor) => {
                await visitor.see(within(region('Menu bar'), content('Show in menu bar')));

                // When - Off is pressed
                await visitor.click(within(region('Menu bar'), button('Off')));
                await visitor.see(within(region('Menu bar'), button('Off')));
            },
        );

        // Then - the panel, Off pressed
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/settings-menu-bar.aria.yaml'));
    });
});
