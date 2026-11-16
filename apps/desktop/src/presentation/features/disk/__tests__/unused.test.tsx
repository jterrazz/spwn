import type { Request } from '@jterrazz/station-api';
import {
    button,
    component,
    content,
    dialog,
    disabled,
    enabled,
    field,
    heading,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { isRecord } from '../../../../domain/request-key';
import { documentPresence } from '../../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../../infrastructure/outbound/engine-link/client';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../../infrastructure/outbound/engine-replay/replay';
import type { Recording, Stage } from '../../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../../dependencies/dependencies';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../../screens/registry';

const recordingFor = await recordingsOf(
    'leftovers',
    'leftover-running',
    'old-home-running',
    'record-unreadable',
);

const SUBJECT = 'station-linux@2026.10.20, old-vm';

const NAMES = ['station-linux@2026.10.20', 'old-vm'];

function staged(recording: Recording): Stage {
    return stagedReplay(recording, {
        hello: { pid: 41, started_at: '2026-11-16T09:00:00Z', version: 'dev' },
        kind: 'open',
        refused: undefined,
        socket: 'replay',
    });
}

function storageOn(stage: Stage) {
    const dependencies = {
        engine: new Client(stage.transport),
        opener: NO_OPENER,
        presence: documentPresence(),
    };
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={{ page: 'storage' }} pages={PAGES} />
        </EngineProvider>
    );
}

function storageOver(recording: Recording) {
    return pageOver(recording, { page: 'storage' });
}

function freesAsked(stage: Stage): readonly unknown[] {
    return stage
        .asked()
        .flatMap((request: Request) =>
            'call' in request && request.call === 'disk.free-unused' ? [request.in] : [],
        );
}

function readsAsked(stage: Stage, read: string): number {
    return stage
        .asked()
        .filter(
            (request) =>
                ('call' in request && request.call === read) ||
                ('watch' in request && request.watch === read),
        ).length;
}

function answeredBy(recording: Recording, other: Recording): Recording {
    return { ...recording, gestures: other.gestures ?? [] };
}

function withStation(recording: Recording, change: (station: Record<string, unknown>) => unknown) {
    return {
        ...recording,
        exchanges: recording.exchanges.map((exchange) =>
            exchange.request.watch === 'disk.read'
                ? {
                      ...exchange,
                      answers: exchange.answers.map((answer: unknown) =>
                          isRecord(answer) && isRecord(answer.out) && isRecord(answer.out.Station)
                              ? {
                                    ...answer,
                                    out: { ...answer.out, Station: change(answer.out.Station) },
                                }
                              : answer,
                      ),
                  }
                : exchange,
        ),
    };
}

const disk = region('Disk');
const freed = region('Can be freed');
const kept = region('Kept');
const done = region('Done here');
const freeIt = enabled(button('Free'));
const freeHeld = disabled(button('Free'));
const freeStation = enabled(button('Free 12 GB'));
const freeCaches = enabled(button('Free 34.8 GB'));

describe('unused', () => {
    test('its Free is offered by the Call as the primary, the caches Free drawn plain beside it', async () => {
        // Given - Storage over leftovers, an unused image and a leftover disk whose CanFree names its Call
        const sentence = content('12.0 GB can be freed: 1 unused image and 1 leftover disk.');
        const result = await component.render(
            storageOver(recordingFor('leftovers')),
            async (visitor) => {
                // When - its disk has been read
                await visitor.see(within(disk, sentence));

                // Then - Station's Free and the caches' Free, and each row worded by its holder and its kind
                await visitor.see(within(disk, freeStation));
                await visitor.see(within(disk, freeCaches));
                await visitor.see(within(freed, content('Images · Unused image')));
                await visitor.see(within(freed, content('Leftovers · Leftover disk')));
            },
        );
        expect(result.html.text).toMatch(/class="[^"]*primary[^"]*"[^>]*>Free 12 GB</u);
        expect(result.html.text).not.toMatch(/class="[^"]*primary[^"]*"[^>]*>Free 34.8 GB</u);
        expect(result.html.text).not.toMatch(/class="[^"]*danger[^"]*"[^>]*>Free 12 GB</u);
        expect(result.errors.text).toBe('');
    });

    test('a folder the engine keeps is drawn with its sentence, and no gesture', async () => {
        // Given - Storage over leftovers, where ~/.tart holds a VM of the owner's
        const said = content(
            '~/.tart holds VMs of yours (linux-dev), so station never frees it: move each into ~/.local/share/station/vms/vms first.',
        );
        const result = await component.render(
            storageOver(recordingFor('leftovers')),
            async (visitor) => {
                // When - its disk has been read
                await visitor.see(within(kept, content('~/.tart')));

                // Then - the engine's sentence, no button in the panel, and no such row to free
                await visitor.see(within(kept, said));
                await visitor.gone(within(kept, button('Free', { exact: false })));
                await visitor.gone(within(freed, content('~/.tart')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("its gesture sends exactly the engine's names, asks the engine's subject typed, and reads again", async () => {
        // Given - Storage over leftovers, its requests kept
        const stage = staged(recordingFor('leftovers'));
        const consequence = content(
            `Deleted for good: ${SUBJECT}. No machine and no file of yours is touched.`,
        );
        let before = { disks: 0, images: 0 };
        const result = await component.render(storageOn(stage), async (visitor) => {
            await visitor.see(within(freed, content('old-vm')));
            before = {
                disks: readsAsked(stage, 'disk.read'),
                images: readsAsked(stage, 'images.list'),
            };

            // When - Free 12 GB is pressed, the subject the engine asks typed, and confirmed
            await visitor.click(within(disk, freeStation));
            const asked = dialog('Free 12 GB on macbook-pro?');
            await visitor.see(within(asked, consequence));
            await visitor.fill(within(asked, field(`Type ${SUBJECT} to confirm`)), SUBJECT);
            await visitor.click(within(asked, freeIt));

            // Then - what was freed is held, and the rows leave
            await visitor.gone(asked);
            await visitor.see(within(done, content('old-vm')));
            await visitor.see(within(done, content('9 GB freed')));
            await visitor.gone(freed);
        });
        expect(freesAsked(stage)).toStrictEqual([
            { Names: NAMES },
            { Confirm: SUBJECT, Names: NAMES },
        ]);
        expect(readsAsked(stage, 'disk.read')).toBeGreaterThan(before.disks);
        expect(readsAsked(stage, 'images.list')).toBeGreaterThan(before.images);
        expect(result.errors.text).toBe('');
    });

    test('a name typed otherwise is refused, nothing is sent with it, and nothing is read again', async () => {
        // Given - Storage over leftovers, the engine's question open
        const stage = staged(recordingFor('leftovers'));
        let before = 0;
        const result = await component.render(storageOn(stage), async (visitor) => {
            await visitor.click(within(disk, freeStation));
            const asked = dialog('Free 12 GB on macbook-pro?');
            const typed = within(asked, field(`Type ${SUBJECT} to confirm`));
            await visitor.see(typed);
            before = readsAsked(stage, 'disk.read');

            // When - a name the engine did not ask is typed
            await visitor.fill(typed, 'old-vm');

            // Then - Free is held, and the rows stay
            await visitor.see(within(asked, freeHeld));
            await visitor.see(within(freed, content('old-vm')));
        });
        expect(freesAsked(stage)).toStrictEqual([{ Names: NAMES }]);
        expect(readsAsked(stage, 'disk.read')).toBe(before);
        expect(result.errors.text).toBe('');
    });

    test('an outcome the engine says broken is held with its reason, the others as freed', async () => {
        // Given - Storage over old-home-running, the old folder and the unused image to free while tart runs
        const subject = 'station-linux@2026.10.20, ~/.tart';
        const result = await component.render(
            storageOver(recordingFor('old-home-running')),
            async (visitor) => {
                await visitor.see(within(freed, content('Leftovers · Old folder')));

                // When - Free 7 GB is confirmed with the engine's subject typed
                await visitor.click(within(disk, button('Free 7 GB')));
                const asked = dialog('Free 7 GB on macbook-pro?');
                await visitor.fill(within(asked, field(`Type ${subject} to confirm`)), subject);
                await visitor.click(within(asked, freeIt));

                // Then - ~/.tart is held as failed with the engine's reason, the image as freed
                await visitor.see(within(done, content('tart is running')));
                await visitor.see(within(done, content('3 GB freed')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a name the engine no longer frees is refused in its words, and nothing is freed', async () => {
        // Given - Storage over leftovers, whose leftover runs by the time the free reaches the engine
        const refusing = answeredBy(recordingFor('leftovers'), recordingFor('leftover-running'));
        const said = content(
            "old-vm is not freeable: in use, gone or not station's, so nothing is freed",
            { exact: false },
        );
        const result = await component.render(storageOver(refusing), async (visitor) => {
            // When - Free 12 GB is pressed
            await visitor.click(within(disk, freeStation));

            // Then - the engine's sentence, no question, and nothing held as done
            await visitor.see(within(disk, said));
            await visitor.gone(dialog());
            await visitor.gone(done);
        });
        expect(result.errors.text).toBe('');
    });

    test('a record the engine cannot read refuses the free in its words', async () => {
        // Given - Storage over leftovers, a machine record damaged by the time the free reaches the engine
        const refusing = answeredBy(recordingFor('leftovers'), recordingFor('record-unreadable'));
        const said = content(
            '~/.local/state/station/vms/api.json cannot be read: nothing is freed until it can',
            { exact: false },
        );
        const result = await component.render(storageOver(refusing), async (visitor) => {
            // When - Free 12 GB is pressed
            await visitor.click(within(disk, freeStation));

            // Then - the engine's sentence, and nothing held as done
            await visitor.see(within(disk, said));
            await visitor.gone(done);
        });
        expect(result.errors.text).toBe('');
    });

    test('with no Call there is no Free of it: its sentence is drawn, and the caches Free is the primary', async () => {
        // Given - Storage over leftovers, its station part as the engine says it while a record cannot be read
        const reason =
            '~/.local/state/station/vms/api.json cannot be read: nothing is offered to free until it can.';
        const unreadable = withStation(recordingFor('leftovers'), (station) => ({
            ...station,
            CanFree: { Bytes: 0, Sentence: reason },
            Verdict: { Line: '', Sentence: 'A machine record cannot be read.', Tone: 'warn' },
        }));
        const result = await component.render(storageOver(unreadable), async (visitor) => {
            // When - its disk has been read
            await visitor.see(heading('A machine record cannot be read.'));

            // Then - the engine's sentence, no Free of what nothing uses, the caches Free as the primary
            await visitor.see(within(disk, content(reason)));
            await visitor.see(within(disk, freeCaches));
            await visitor.gone(button('Free 12 GB'));
            await visitor.gone(freed);
        });
        expect(result.html.text).toMatch(/class="[^"]*primary[^"]*"[^>]*>Free 34.8 GB</u);
        expect(result.errors.text).toBe('');
    });
});
