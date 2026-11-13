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

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('two-macs');

const byApp = region('By app');

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
            await visitor.see(content('What can be rebuilt holds 34.8 GB.', { exact: false }));

            // When - the footer's gesture is clicked
            await visitor.click(button('Free 34.8 GB'));

            // Then - the engine's outcomes stay held once nothing is left to rebuild
            const freed = region('Done here');
            await visitor.see(within(freed, content('homebrew')));
            await visitor.gone(button('Free 34.8 GB'));
        });
        expect(result.errors.text).toBe('');
    });

    test("an app's Free frees that app alone", async () => {
        // Given - This Mac's Storage tab over two-macs
        const mac = tabOver('two-macs', 'jterrazz-macbook-pro');
        const result = await component.render(mac, async (visitor) => {
            await visitor.see(within(byApp, button('Free go')));

            // When - go's Free is clicked
            await visitor.click(within(byApp, button('Free go')));

            // Then - go leaves what the engine reads again, the others stay, and what it freed is held
            await visitor.gone(within(byApp, button('Free go')));
            await visitor.see(within(byApp, button('Free homebrew')));
            await visitor.see(within(region('Done here'), content('19.4 GB freed')));
        });
        expect(result.errors.text).toBe('');
    });

    test('move takes a misplaced folder where its app looks, and its row leaves the page', async () => {
        // Given - This Mac's Storage tab over two-macs, where ~/.tart is misplaced
        const mac = tabOver('two-macs', 'jterrazz-macbook-pro');
        const result = await component.render(mac, async (visitor) => {
            await visitor.see(within(byApp, content('misplaced 73.3 GB')));

            // When - its Move is clicked
            await visitor.click(button('Move tart'));

            // Then - disk.read read again lists nothing misplaced, and a move claims to free nothing
            await visitor.gone(within(byApp, button('Move tart')));
            await visitor.gone(dialog());
            await visitor.see(within(region('Done here'), content('tart', { exact: false })));
            await visitor.gone(within(region('Done here'), content('0 B freed')));
        });
        expect(result.errors.text).toBe('');
    });

    test("a machine's Storage tab erases an app's data from its By app row, by its name typed", async () => {
        // Given - This Mac's Storage tab over two-macs, where ollama holds data alone
        const mac = tabOver('two-macs', 'jterrazz-macbook-pro');
        const result = await component.render(mac, async (visitor) => {
            await visitor.see(within(byApp, button('Erase… ollama')));

            // When - Erase… ollama is clicked and its name typed
            await visitor.click(within(byApp, button('Erase… ollama')));
            const asked = dialog('Erase ollama?');
            await visitor.fill(within(asked, field('Type ollama to confirm')), 'ollama');
            const erase = enabled(button('Erase'));
            await visitor.click(within(asked, erase));

            // Then - ollama leaves what the engine reads again, and what the erase did is held
            await visitor.gone(asked);
            await visitor.gone(within(byApp, button('Erase… ollama')));
            await visitor.see(within(region('Done here'), content('ollama')));
        });
        expect(result.errors.text).toBe('');
    });

    test('erase asks for the name typed, then the data leaves the page', async () => {
        // Given - Storage over two-macs
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            const data = region('Your data');
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
            await visitor.see(within(region('Done here'), content('ollama')));
        });
        expect(result.errors.text).toBe('');
    });

    test('an erase the engine refuses shows its words and its hint, and the row stays', async () => {
        // Given - Storage over two-macs, where OrbStack erases its own data
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            const data = region('Your data');

            // When - Erase… orbstack is confirmed with its name typed
            await visitor.click(within(data, button('Erase… orbstack')));
            const asked = dialog('Erase orbstack?');
            await visitor.fill(within(asked, field('Type orbstack to confirm')), 'orbstack');
            await visitor.click(within(asked, button('Erase')));

            // Then - the engine's refusal, as it came, and the row as it was
            const said = content('has no data station can erase', { exact: false });
            await visitor.see(within(data, said));
            await visitor.see(
                within(data, content('OrbStack erases its own data, in its Settings')),
            );
            await visitor.see(within(data, button('Erase… orbstack')));
        });
        expect(result.errors.text).toBe('');
    });

    test('delete of an image asks yes, then the image leaves the list and its outcome stays above it', async () => {
        // Given - Storage over two-macs, two versions of station-linux built
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            const images = region('Images');
            await visitor.see(within(images, content('2026.10.20')));

            // When - its Delete is clicked, then confirmed
            await visitor.click(within(images, button('Delete station-linux 2026.10.20')));
            const asked = dialog('Delete station-linux 2026.10.20?');
            await visitor.click(within(asked, button('Delete')));

            // Then - images.list read again holds the newest alone, and the delete's outcome is held above it
            await visitor.gone(asked);
            await visitor.gone(within(images, content('2026.10.20')));
            await visitor.see(within(images, content('2026.11.01')));
            await visitor.see(within(region('Done here'), content('station-linux 2026.10.20')));
        });
        expect(result.errors.text).toBe('');
    });
});
