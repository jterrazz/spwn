import { button, component, content, file, heading, navigation, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { Link } from '../../../../application/ports/engine.port';
import { refreshRefused, updated } from '../../../../infrastructure/outbound/core-scripted/release';
import { scriptedCore } from '../../../../infrastructure/outbound/core-scripted/scripted-core';
import { SCRIPTS } from '../../../../infrastructure/outbound/core-scripted/scripts';
import { documentPresence } from '../../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../../infrastructure/outbound/engine-link/client';
import { helloOf } from '../../../../infrastructure/outbound/engine-replay/hello';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../../infrastructure/outbound/engine-replay/replay';
import type { Recording } from '../../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../../dependencies/dependencies';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf(
    'release-behind',
    'release-first-refused',
    'release-never-asked',
    'release-unreachable',
    'stories',
);

const sidebar = navigation('Sidebar');
const BEHIND = heading('spwn v0.22.0 is out.');
const ROW = button('v0.22.0 available Update');

type Over = { script?: 'update' | 'update-failing'; link?: Link };

function engineOver(recording: Recording, { script, link }: Over = {}) {
    const stage = stagedReplay(recording, link);
    const after = updated(recording);
    const core =
        script === undefined
            ? undefined
            : scriptedCore(SCRIPTS[script], {
                  onStarted: () => {
                      stage.restart({ ...helloOf(after), pid: 1, started_at: 'restarted' }, after);
                  },
              });
    return (
        <EngineProvider
            dependencies={{
                core,
                engine: new Client(stage.transport),
                opener: NO_OPENER,
                presence: documentPresence(),
            }}
        >
            <Window initial={{ page: 'engine' }} pages={PAGES} />
        </EngineProvider>
    );
}

describe('EnginePage', () => {
    test('a Mac behind the latest release is offered the update, and its sidebar foot says so', async () => {
        // Given - an engine whose reading is behind
        const result = await component.render(
            engineOver(recordingFor('release-behind')),
            async (visitor) => {
                // When - the page has read the release
                await visitor.see(BEHIND);

                // Then - the foot's row names the release as sent
                await visitor.see(within(sidebar, ROW));
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/engine-behind.aria.yaml'));
    });

    test("a Mac not behind shows the engine's head and no offer", async () => {
        // Given - an engine whose reading is not behind
        const result = await component.render(
            engineOver(recordingFor('stories')),
            async (visitor) => {
                // When - the page has read the release
                await visitor.see(heading('The latest release is not known yet.'));
            },
        );

        // Then - the head as sent, Check now, and no Update anywhere
        await expect(result.tree).toMatch(file('./__expected__/engine-stories.aria.yaml'));
    });

    test("the last refused ask is said in the engine's words beside when it last answered", async () => {
        // Given - an engine whose last ask of GitHub was refused
        const result = await component.render(
            engineOver(recordingFor('release-unreachable')),
            async (visitor) => {
                // When - the page has read the release
                await visitor.see(BEHIND);

                // Then - the refusal as sent, after the check it keeps
                await visitor.see(
                    content('GitHub: dial tcp: connection refused', { exact: false }),
                );
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/engine-unreachable.aria.yaml'));
    });

    test("a refused Check now draws the engine's message", async () => {
        // Given - an engine whose ask of GitHub is refused
        const recording = refreshRefused(recordingFor('release-unreachable'));
        const result = await component.render(engineOver(recording), async (visitor) => {
            await visitor.see(BEHIND);

            // When - Check now is pressed
            await visitor.click(button('Check now'));

            // Then - the refusal is the engine's own sentence
            await visitor.see(
                content('The latest release was not read: GitHub: dial tcp: connection refused.', {
                    exact: false,
                }),
            );
        });
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/engine-check-refused.aria.yaml'));
    });

    test('a reading refused before any answer draws no checked time, and one never asked draws neither', async () => {
        // Given - a world whose first ask was refused, then a world that never asked
        // When - each page has read the release
        const first = await component.render(
            engineOver(recordingFor('release-first-refused')),
            async (visitor) => {
                await visitor.see(
                    content('GitHub: dial tcp: connection refused', { exact: false }),
                );
            },
        );
        const never = await component.render(
            engineOver(recordingFor('release-never-asked')),
            async (visitor) => {
                await visitor.see(heading('The latest release is not known yet.'));
            },
        );

        // Then - no Checked line in either, and no What changed without notes
        await expect(first.tree).toMatch(file('./__expected__/engine-first-refused.aria.yaml'));
        await expect(never.tree).toMatch(file('./__expected__/engine-unasked.aria.yaml'));
    });

    test('one press of Update runs the installer then the engine start, and the row goes once the engine is not behind', async () => {
        // Given - a behind engine and a core whose install and start succeed
        const result = await component.render(
            engineOver(recordingFor('release-behind'), { script: 'update' }),
            async (visitor) => {
                await visitor.see(BEHIND);

                // When - Update is pressed
                await visitor.click(within(navigation('Sidebar'), ROW));
                await visitor.click(button('Update'));

                // Then - the update ends, and the engine's new head replaces the offer
                await visitor.see(heading('Updated.'));
                await visitor.see(heading('This Mac runs the latest release.'));
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/engine-updated.aria.yaml'));
    });

    test("a failed update says the installer's last error line and offers Retry", async () => {
        // Given - a behind engine and an installer that fails its checksum
        const result = await component.render(
            engineOver(recordingFor('release-behind'), { script: 'update-failing' }),
            async (visitor) => {
                await visitor.see(BEHIND);

                // When - Update is pressed
                await visitor.click(button('Update'));

                // Then - the error line is the sentence
                await visitor.see(
                    heading(
                        'Error: station-darwin-arm64 does not match its checksum, nothing installed',
                    ),
                );
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/engine-update-failed.aria.yaml'));
    });

    test('an engine older than the release calls shows no row and no release', async () => {
        // Given - an engine whose hello offers no release call
        const link: Link = {
            hello: { calls: ['engine.read'], pid: 1, started_at: 'older', version: 'v0.20.0' },
            kind: 'open',
            refused: undefined,
            socket: 'replay',
        };
        const result = await component.render(
            engineOver(recordingFor('release-behind'), { link }),
            async (visitor) => {
                // When - the page is open
                await visitor.see(heading('No verdict from the engine.'));
            },
        );

        // Then - no Update offered anywhere
        await expect(result.tree).toMatch(file('./__expected__/engine-older.aria.yaml'));
    });
});
