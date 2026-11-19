import { button, component, content, file, heading, navigation, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { scriptedCore } from '../../../../infrastructure/outbound/core-scripted/scripted-core';
import { SCRIPTS } from '../../../../infrastructure/outbound/core-scripted/scripts';
import type { CoreScriptName } from '../../../../infrastructure/outbound/core-scripted/scripts';
import { documentPresence } from '../../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../../infrastructure/outbound/engine-link/client';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../../dependencies/dependencies';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('two-macs');

const sidebar = navigation('Sidebar');
const INSTALLED = heading('spwn is installed. Start the engine next.');

function noEngineWith(script: CoreScriptName) {
    const stage = stagedReplay(recordingFor('two-macs'), {
        kind: 'absent',
        reason: 'connection refused',
        retryIn: 30_000,
        socket: '/Users/you/.local/state/station/engine.sock',
    });
    const core = scriptedCore(SCRIPTS[script], {
        onStarted: () => {
            stage.show({ hello: undefined, kind: 'open', refused: undefined, socket: 'replay' });
        },
    });
    return (
        <EngineProvider
            dependencies={{
                core,
                coreLookEvery: 10,
                engine: new Client(stage.transport),
                opener: NO_OPENER,
                presence: documentPresence(),
            }}
        >
            <Window initial={{ page: 'home' }} pages={PAGES} />
        </EngineProvider>
    );
}

describe('CorePage', () => {
    test('a Mac with no core is offered the install, the command and its promises, in a quiet window', async () => {
        // Given - no engine answers and no station lies in the core's folder
        const result = await component.render(noEngineWith('missing'), async (visitor) => {
            // When - the window has looked at the disk
            await visitor.see(heading('spwn is not installed on this Mac.'));
        });

        // Then - one panel on Home, and a sidebar whose items wait for an engine
        await expect(result.tree).toMatch(file('./__expected__/missing.aria.yaml'));
    });

    test("the install shows the installer's own lines while it runs", async () => {
        // Given - an installer that says its lines and holds
        const result = await component.render(noEngineWith('holding'), async (visitor) => {
            // When - Install is pressed
            await visitor.click(button('Install'));

            // Then - the lines as sent, under a progress that names no percentage
            await visitor.see(content('Downloading station-darwin-arm64...'));
        });
        await expect(result.tree).toMatch(file('./__expected__/installing.aria.yaml'));
    });

    test('a cancelled install goes back to the offer', async () => {
        // Given - an installer under way
        const result = await component.render(noEngineWith('holding'), async (visitor) => {
            await visitor.click(button('Install'));
            await visitor.see(content('Downloading station-darwin-arm64...'));

            // When - it is cancelled
            await visitor.click(button('Cancel'));

            // Then - the offer again, nothing installed
            await visitor.see(heading('spwn is not installed on this Mac.'));
        });
        expect(result.errors.text).toBe('');
    });

    test("a failed install says the installer's last error line as sent, and keeps its output", async () => {
        // Given - an installer that ends on a checksum that does not match
        const result = await component.render(noEngineWith('failing'), async (visitor) => {
            // When - Install is pressed
            await visitor.click(button('Install'));

            // Then - the error line is the sentence, the exit code below it
            await visitor.see(
                heading(
                    'Error: station-darwin-arm64 does not match its checksum, nothing installed',
                ),
            );
        });
        await expect(result.tree).toMatch(file('./__expected__/failed.aria.yaml'));
    });

    test('a terminal install is waited for, with the command and a way back', async () => {
        // Given - no core, and none coming
        const result = await component.render(noEngineWith('missing'), async (visitor) => {
            // When - the owner says he installs from the terminal
            await visitor.click(button("I'll install it from the terminal"));

            // Then - the page looks for station every few seconds
            await visitor.see(heading('Waiting for spwn on this Mac.'));
        });
        await expect(result.tree).toMatch(file('./__expected__/waiting.aria.yaml'));
    });

    test('a Mac installed from the terminal moves on by itself once station appears', async () => {
        // Given - no core yet, and a terminal install that lands on the third look
        const result = await component.render(noEngineWith('terminal'), async (visitor) => {
            // When - the owner waits on the terminal
            await visitor.click(button("I'll install it from the terminal"));

            // Then - the installed panel comes without a further gesture
            await visitor.see(INSTALLED);
        });
        expect(result.errors.text).toBe('');
    });

    test('a core installed but not started offers the start', async () => {
        // Given - a station in the core's folder, no engine answering
        const result = await component.render(noEngineWith('installed'), async (visitor) => {
            // When - the window has looked at the disk
            await visitor.see(INSTALLED);
        });

        // Then - what is on the Mac, what starting does, and its command
        await expect(result.tree).toMatch(file('./__expected__/installed.aria.yaml'));
    });

    test('the start stays offered after Later', async () => {
        // Given - an installed core, no engine answering
        const result = await component.render(noEngineWith('installed'), async (visitor) => {
            await visitor.see(INSTALLED);

            // When - Later is chosen
            await visitor.click(button('Later'));

            // Then - a short panel still offers the start
            await visitor.see(heading('The engine is not started.'));
            await visitor.see(button('Start the engine'));
        });
        expect(result.errors.text).toBe('');
    });

    test('once the started engine answers, the normal window replaces the panel', async () => {
        // Given - an installed core whose start succeeds
        const result = await component.render(noEngineWith('installed'), async (visitor) => {
            // When - the engine is started
            await visitor.click(button('Start the engine'));

            // Then - Home is the engine's, and the sidebar lists its machines
            await visitor.see(content('2 things need you.'));
            await visitor.see(within(sidebar, button('api stopped')));
        });
        expect(result.errors.text).toBe('');
    });
});
