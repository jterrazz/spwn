import {
    button,
    component,
    content,
    dialog,
    disabled,
    enabled,
    field,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('two-macs');

const byApp = region('By app');
const THIS_MAC = region('macbook-pro');
const thisByApp = within(THIS_MAC, byApp);
const thisDone = within(THIS_MAC, region('Done here'));
const thisImages = within(THIS_MAC, region('Images'));

function storageOver(world: string) {
    return pageOver(recordingFor(world), { page: 'storage' });
}

function tabOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Storage' });
}

describe('disk gestures', () => {
    test("a machine's Free of what can be rebuilt asks nothing, then lists what the engine freed", async () => {
        // Given - linux-1's Storage tab over two-macs, two rows it can rebuild, 1.5 GB in all
        const result = await component.render(tabOver('two-macs', 'linux-1'), async (visitor) => {
            await visitor.see(within(byApp, content('node')));

            // When - its one primary gesture is clicked
            await visitor.click(button('Free 1.5 GB'));

            // Then - no dialog, the engine's outcome per app held, and disk.read read again holds no such row
            const freed = region('Done here');
            await visitor.see(within(freed, content('1.2 GB freed')));
            await visitor.see(within(freed, content('node')));
            await visitor.gone(within(byApp, button('Free node')));
            await visitor.gone(dialog());
        });
        expect(result.errors.text).toBe('');
    });

    test('freeing what can be rebuilt on Storage frees every rebuildable row, and the footer leaves', async () => {
        // Given - Storage over two-macs, 34.8 GB that can be rebuilt
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            await visitor.see(within(THIS_MAC, content('Caches hold 34.8 GB.', { exact: false })));

            // When - the footer's gesture is clicked
            await visitor.click(within(THIS_MAC, button('Free 34.8 GB')));

            // Then - the engine's outcomes stay held once nothing is left to rebuild
            const freed = thisDone;
            await visitor.see(within(freed, content('homebrew')));
            await visitor.gone(within(THIS_MAC, button('Free 34.8 GB')));
        });
        expect(result.errors.text).toBe('');
    });

    test("an app's Free frees that app alone", async () => {
        // Given - This Mac's section of Storage over two-macs
        const mac = storageOver('two-macs');
        const result = await component.render(mac, async (visitor) => {
            await visitor.see(within(thisByApp, button('Free go')));

            // When - go's Free is clicked
            await visitor.click(within(thisByApp, button('Free go')));

            // Then - go leaves what the engine reads again, the others stay, and what it freed is held
            await visitor.gone(within(thisByApp, button('Free go')));
            await visitor.see(within(thisByApp, button('Free homebrew')));
            await visitor.see(within(thisDone, content('19.4 GB freed')));
        });
        expect(result.errors.text).toBe('');
    });

    test('move takes a misplaced folder where its app looks, and its row leaves the page', async () => {
        // Given - This Mac's section of Storage over two-macs, where ~/.tart is misplaced
        const mac = storageOver('two-macs');
        const result = await component.render(mac, async (visitor) => {
            await visitor.see(within(thisByApp, content('misplaced 73.3 GB')));

            // When - its Move is clicked
            await visitor.click(within(thisByApp, button('Move tart')));

            // Then - disk.read read again lists nothing misplaced, and a move claims to free nothing
            await visitor.gone(within(thisByApp, button('Move tart')));
            await visitor.gone(dialog());
            await visitor.see(within(thisDone, content('tart', { exact: false })));
            await visitor.gone(within(thisDone, content('0 B freed')));
        });
        expect(result.errors.text).toBe('');
    });

    test('erase asks for the name typed, then the data leaves the page', async () => {
        // Given - Storage over two-macs
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            const data = thisByApp;
            await visitor.see(within(data, button('Erase… ollama')));

            // When - Erase… ollama is clicked, a wrong name typed, then the right one, in a dialog whose verb is Erase
            await visitor.click(within(data, button('Erase… ollama')));
            const asked = dialog('Erase ollama?');
            const typed = within(asked, field('Type ollama to confirm'));
            const held = disabled(button('Erase'));
            const free = enabled(button('Erase'));
            await visitor.fill(typed, 'olla');
            await visitor.see(within(asked, held));
            await visitor.fill(typed, 'ollama');
            await visitor.see(within(asked, free));
            await visitor.click(within(asked, button('Erase')));

            // Then - ollama is no row of disk.read read again, and what the erase did is held above the lists
            await visitor.gone(asked);
            await visitor.gone(within(data, button('Erase… ollama')));
            await visitor.see(within(data, button('Erase… claude')));
            await visitor.see(within(thisDone, content('ollama')));
        });
        expect(result.errors.text).toBe('');
    });

    test('an erase the engine refuses shows its words and its hint, and the row stays', async () => {
        // Given - Storage over two-macs, where the engine answers the erase of xcode with disk.app-running
        const recording = recordingFor('two-macs');
        const refusing = {
            ...recording,
            gestures: [
                ...(recording.gestures ?? []),
                {
                    after: [],
                    answers: [
                        {
                            error: {
                                code: 'disk.app-running',
                                detail: { process: 'Xcode', subject: 'xcode', verb: 'erase' },
                                hint: 'quit Xcode, then station disk erase xcode',
                                message: 'Xcode is running: its data cannot be erased now',
                                subject: 'xcode',
                                type: 'failed',
                            },
                            v: 2,
                        },
                    ],
                    request: { call: 'disk.erase', in: { App: 'xcode', Confirm: 'xcode' } },
                },
            ],
        };
        const result = await component.render(
            pageOver(refusing, { page: 'storage' }),
            async (visitor) => {
                const data = thisByApp;

                // When - Erase… xcode is confirmed with its name typed
                await visitor.click(within(data, button('Erase… xcode')));
                const asked = dialog('Erase xcode?');
                await visitor.fill(within(asked, field('Type xcode to confirm')), 'xcode');
                await visitor.click(within(asked, button('Erase')));

                // Then - the engine's sentence and its hint, as they came, and the row as it was
                const said = content('Xcode is running: its data cannot be erased now', {
                    exact: false,
                });
                await visitor.see(within(data, said));
                await visitor.see(
                    within(data, content('quit Xcode, then station disk erase xcode')),
                );
                await visitor.see(within(data, button('Erase… xcode')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a delete of an image the engine refuses as in use shows its sentence and its hint, and the row as it was', async () => {
        // Given - Storage over two-macs, where the engine answers the delete with image.in-use
        const recording = recordingFor('two-macs');
        const refusing = {
            ...recording,
            gestures: (recording.gestures ?? []).map((gesture) =>
                gesture.request.call === 'images.delete'
                    ? {
                          ...gesture,
                          after: [],
                          answers: [
                              {
                                  error: {
                                      code: 'image.in-use',
                                      detail: { version: '2026.10.20', vms: 'linux-1' },
                                      hint: 'station images lists the versions you can delete',
                                      message:
                                          'image base-ubuntu 2026.10.20 has machines made from it: linux-1',
                                      subject: 'base-ubuntu',
                                      type: 'refused',
                                  },
                                  v: 2,
                              },
                          ],
                      }
                    : gesture,
            ),
        };
        const result = await component.render(
            pageOver(refusing, { page: 'storage' }),
            async (visitor) => {
                const images = thisImages;
                await visitor.see(within(images, content('2026.10.20')));

                // When - its Delete is clicked, then confirmed
                await visitor.click(within(images, button('Delete base-ubuntu 2026.10.20')));
                const asked = dialog('Delete base-ubuntu 2026.10.20?');
                await visitor.click(within(asked, button('Delete')));

                // Then - the engine's sentence and its hint, and the row is still listed
                await visitor.see(
                    within(images, content('has machines made from it: linux-1', { exact: false })),
                );
                await visitor.see(
                    within(images, content('station images lists the versions you can delete')),
                );
                await visitor.see(within(images, button('Delete base-ubuntu 2026.10.20')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('delete of an image asks yes, then the image leaves the list and its outcome stays above it', async () => {
        // Given - Storage over two-macs, two versions of base-ubuntu built
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            const images = thisImages;
            await visitor.see(within(images, content('2026.10.20')));

            // When - its Delete is clicked, then confirmed
            await visitor.click(within(images, button('Delete base-ubuntu 2026.10.20')));
            const asked = dialog('Delete base-ubuntu 2026.10.20?');
            await visitor.click(within(asked, button('Delete')));

            // Then - images.list read again holds the newest alone, and the delete's outcome is held above it
            await visitor.gone(asked);
            await visitor.gone(within(images, content('2026.10.20')));
            await visitor.see(within(images, content('2026.11.01')));
            await visitor.see(within(thisDone, content('base-ubuntu 2026.10.20')));
        });
        expect(result.errors.text).toBe('');
    });
});
