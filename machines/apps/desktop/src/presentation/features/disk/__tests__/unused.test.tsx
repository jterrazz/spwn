import { button, component, content, dialog, field, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import {
    answeredBy,
    disk,
    done,
    eachAsking,
    everyOutcomeBroken,
    freeCaches,
    freed,
    freeIt,
    freesAsked,
    freeStation,
    HELLO_AGAIN,
    kept,
    MINI,
    NAMES,
    narrowedTo,
    readsAsked,
    sentTo,
    staged,
    storageOn,
    storageOver,
    SUBJECT,
    THIS_MAC,
    withStation,
} from './unused-stage';

const recordingFor = await recordingsOf(
    'leftovers',
    'leftover-running',
    'old-home-running',
    'record-unreadable',
    'two-macs',
);

describe('unused', () => {
    test('its Free is offered by the Call as the primary, the caches Free drawn plain beside it', async () => {
        // Given - Storage over leftovers, an unused image and a leftover disk whose CanFree names its Call
        const sentence = content('12.0 GB can be freed: 1 unused image and 1 leftover disk.');
        const result = await component.render(
            storageOver(recordingFor('leftovers')),
            async (visitor) => {
                // When - its disk has been read
                await visitor.see(within(disk, sentence));

                // Then - Station's Free and the caches' Free, and each row worded by its kind alone
                await visitor.see(within(disk, freeStation));
                await visitor.see(within(disk, freeCaches));
                await visitor.see(within(freed, content('Unused image')));
                await visitor.see(within(freed, content('Leftover disk')));
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
            '~/.tart holds VMs of yours (linux-dev), so Machines never frees it: move each into ~/.local/share/station/vms/vms first.',
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

    test("its gesture sends exactly the engine's names, none of Kept, asks yes or no naming them, and reads again", async () => {
        // Given - Storage over leftovers, ~/.tart kept beside the two rows to free, its requests kept
        const stage = staged(recordingFor('leftovers'));
        const consequence = content(
            `Deleted for good: ${SUBJECT}. No machine and no file of yours is touched.`,
        );
        let before = { disks: 0, images: 0 };
        const result = await component.render(storageOn(stage), async (visitor) => {
            await visitor.see(within(freed, content('old-vm')));
            await visitor.see(within(kept, content('~/.tart')));
            before = {
                disks: readsAsked(stage, 'disk.read'),
                images: readsAsked(stage, 'images.list'),
            };

            // When - Free 12 GB is pressed, and the engine's question, which types nothing, answered yes
            await visitor.click(within(disk, freeStation));
            const asked = dialog('Free 12 GB on macbook-pro?');
            await visitor.see(within(asked, consequence));
            await visitor.gone(within(asked, field(`Type ${SUBJECT} to confirm`)));
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

    test('an outcome the engine says broken is held with its reason, the others as freed', async () => {
        // Given - Storage over old-home-running, the old folder and the unused image to free while tart runs
        const result = await component.render(
            storageOver(recordingFor('old-home-running')),
            async (visitor) => {
                await visitor.see(within(freed, content('Old folder')));

                // When - Free 7 GB is answered yes
                await visitor.click(within(disk, button('Free 7 GB')));
                await visitor.click(within(dialog('Free 7 GB on macbook-pro?'), freeIt));

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
            "old-vm is not freeable: in use, gone or not Machines', so nothing is freed",
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
            await visitor.see(within(THIS_MAC, heading('A machine record cannot be read.')));

            // Then - the engine's sentence, no Free of what nothing uses, the caches Free as the primary
            await visitor.see(within(disk, content(reason)));
            await visitor.see(within(disk, freeCaches));
            await visitor.gone(button('Free 12 GB'));
            await visitor.gone(freed);
        });
        expect(result.html.text).toMatch(/class="[^"]*primary[^"]*"[^>]*>Free 34.8 GB</u);
        expect(result.errors.text).toBe('');
    });

    test('every outcome broken is held as failed with its reason, and no done sits beside the button', async () => {
        // Given - Storage over old-home-running, every outcome of its free broken and the rows still listed
        const page = storageOver(everyOutcomeBroken(recordingFor('old-home-running')));
        const result = await component.render(page, async (visitor) => {
            // When - Free 7 GB is answered yes
            await visitor.click(within(disk, button('Free 7 GB')));
            await visitor.click(within(dialog('Free 7 GB on macbook-pro?'), freeIt));

            // Then - each reason is held, the rows stay listed, and the Disk panel says nothing is done
            await visitor.see(within(done, content('the image is in use')));
            await visitor.see(within(done, content('tart is running')));
            await visitor.see(within(freed, content('~/.tart')));
            await visitor.gone(within(disk, content('done')));
        });
        expect(result.errors.text).toBe('');
    });

    test("the other Mac's Free, pressed once a question is cancelled, sends its own names and never the first", async () => {
        // Given - Storage over two-macs, the mini's section holding an image of its own to free
        const own = 'base-ubuntu@2026.10.20';
        const other = 'mini-image@2026.10.01';
        const stage = staged(eachAsking(recordingFor('two-macs'), own, other));
        const result = await component.render(storageOn(stage), async (visitor) => {
            await visitor.click(within(disk, button('Free 3 GB')));
            const asked = dialog('Free 3 GB on macbook-pro?');
            await visitor.see(
                within(asked, content(`Deleted for good: ${own}.`, { exact: false })),
            );

            // When - the question is cancelled, then the Free of the mini's own section pressed
            await visitor.click(within(asked, button('Cancel')));
            const miniDisk = within(MINI, region('Disk'));
            await visitor.click(within(miniDisk, button('Free 2 GB')));
            const miniAsked = dialog('Free 2 GB on mac-mini?');
            await visitor.see(
                within(miniAsked, content(`Deleted for good: ${other}.`, { exact: false })),
            );
        });

        // Then - each Mac was sent its own names alone
        expect(sentTo(stage)).toStrictEqual([
            { names: [own], target: undefined },
            { names: [other], target: { name: 'mac-mini' } },
        ]);
        expect(result.errors.text).toBe('');
    });

    test('names that change under the open question are asked again with the new subject, nothing done', async () => {
        // Given - Storage over leftovers, the question open for both names, the next read listing the image only
        const first = 'base-ubuntu@2026.10.20';
        const stage = staged(recordingFor('leftovers'));
        const result = await component.render(storageOn(stage), async (visitor) => {
            await visitor.click(within(disk, freeStation));
            const asked = dialog('Free 12 GB on macbook-pro?');
            await visitor.see(
                within(asked, content(`Deleted for good: ${SUBJECT}.`, { exact: false })),
            );

            // When - the names change, and the old question answered yes
            stage.restart(HELLO_AGAIN, narrowedTo(recordingFor('leftovers'), first, SUBJECT));
            await visitor.see(within(disk, button('Free 3 GB')));
            await visitor.click(within(asked, freeIt));

            // Then - the yes named the old names, so the engine asks again for the new ones, nothing is done
            await visitor.see(
                within(asked, content(`Deleted for good: ${first}.`, { exact: false })),
            );
            await visitor.gone(done);
        });
        expect(freesAsked(stage)).toStrictEqual([
            { Names: NAMES },
            { Confirm: SUBJECT, Names: [first] },
        ]);
        expect(result.errors.text).toBe('');
    });
});
