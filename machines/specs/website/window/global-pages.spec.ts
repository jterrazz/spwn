import {
    banner,
    button,
    content,
    dialog,
    heading,
    navigation,
    option,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { PINNED, website } from '../website.specification.js';

const column = navigation('Column');
const ENGINE = within(column, button('Engine', { exact: false }));
const path = navigation('Path');
const THIS_MAC = region('macbook-pro');
const MINI = region('mac-mini');
const MINI_ENGINE = within(MINI, region('Engine'));
const AIR_ENGINE = within(region('macbook-air'), region('Engine'));

describe('global pages', () => {
    test('A12 the column opens Storage beside it, every Mac in its own section', async () => {
        // Given - the window on Home over two-macs
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home', async (visitor) => {
                // When - the column's Storage is pressed
                await visitor.click(within(column, button('Storage')));

                // Then - This Mac's verdict and the Mac mini's, each in its section, the column still there
                await visitor.see(within(THIS_MAC, heading('3.0 GB can be freed.')));
                await visitor.see(within(MINI, heading('Nothing to free.')));
                await visitor.see(within(column, button('mac-mini online')));
            });
        expect(result.errors.text).toBe('');
    });

    test("A12 the column opens Engine beside it, each Mac's engine with its state and version", async () => {
        // Given - the window on Home over stories, whose engine has no release out and whose macbook-air sleeps
        const result = await website
            .storage(PINNED)
            .visit('/?world=stories&page=home', async (visitor) => {
                // When - the Engine line at the column's foot is pressed
                await visitor.click(ENGINE);

                // Then - This Mac's release and engine, the Mac mini's engine and version, the Air asleep
                await visitor.see(
                    within(THIS_MAC, heading('The latest release is not known yet.')),
                );
                await visitor.see(within(THIS_MAC, button('Check now')));
                await visitor.see(within(MINI_ENGINE, content('running')));
                await visitor.see(within(MINI_ENGINE, content('v4.0.0')));
                await visitor.see(within(AIR_ENGINE, content('asleep')));
                await visitor.gone(within(MINI, button('Check now')));
            });
        expect(result.errors.text).toBe('');
    });

    test('A13 back walks from Engine to Storage to Home, and forward returns', async () => {
        // Given - the window on Home over two-macs
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home', async (visitor) => {
                await visitor.see(content('2 things need you.'));

                // When - Storage, then Engine, is pressed in the column, then Back twice and Forward once
                await visitor.click(within(column, button('Storage')));
                await visitor.see(within(MINI, heading('Nothing to free.')));
                await visitor.click(ENGINE);
                await visitor.see(within(MINI, region('Engine')));
                await visitor.click(button('Back'));
                await visitor.see(within(MINI, heading('Nothing to free.')));
                await visitor.click(button('Back'));
                await visitor.see(content('2 things need you.'));
                await visitor.click(button('Forward'));

                // Then - Storage again, with the column
                await visitor.see(within(THIS_MAC, heading('3.0 GB can be freed.')));
                await visitor.see(within(column, button('mac-mini online')));
            });
        expect(result.errors.text).toBe('');
    });

    test('A14 a link to Engine opens it beside the pinned column, its pin drawn in the bar', async () => {
        // Given - the window opened on the Engine page over two-macs
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=engine', async (visitor) => {
                // When - the page is read
                await visitor.see(within(MINI, region('Engine')));

                // Then - the column with its Engine line, and the bar's pin
                await visitor.see(ENGINE);
                await visitor.see(within(banner(), button('Pin the column')));
            });
        expect(result.errors.text).toBe('');
    });

    test("A15 a link to a Mac's Storage tab opens Storage, the Mac's disk in its section", async () => {
        // Given - the window opened on This Mac's Storage tab over two-macs, as a relaunch or an old link does
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&machine=macbook-pro&tab=Storage', async (visitor) => {
                // When - the page is read
                await visitor.see(within(THIS_MAC, heading('3.0 GB can be freed.')));

                // Then - the Storage page, with the Mac mini's section beside This Mac's
                await visitor.see(within(MINI, heading('Nothing to free.')));
                await visitor.gone(heading('macbook-pro online'));
            });
        expect(result.errors.text).toBe('');
    });

    test("A16 the bar's New machine on Home opens the sheet", async () => {
        // Given - the window on Home over two-macs
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home', async (visitor) => {
                await visitor.see(content('2 things need you.'));

                // When - the bar's New machine is pressed
                await visitor.click(within(banner(), button('New machine')));

                // Then - the New machine sheet
                await visitor.see(dialog('New machine on mac-mini'));
            });
        expect(result.errors.text).toBe('');
    });

    test('A17 the path of a catalog item reads Machine, its shelf, then the item, and the shelf lists the places', async () => {
        // Given - the window on the app node of the Apps shelf over stories
        const result = await website
            .storage(PINNED)
            .visit('/?world=stories&page=catalog-apps&item=node', async (visitor) => {
                // When - the path is read, then its shelf pressed
                await visitor.see(within(path, button('Machine', { exact: false })));
                await visitor.see(within(path, content('node')));
                await visitor.gone(within(path, button('node')));
                await visitor.click(within(path, button('Apps')));

                // Then - the space's places, and Apps opens the shelf again
                await visitor.see(within(dialog('Places'), option('Skills', { exact: false })));
                await visitor.click(within(dialog('Places'), option('Apps', { exact: false })));
                await visitor.see(region('Apps'));
                await visitor.gone(within(path, content('node')));
            });
        expect(result.errors.text).toBe('');
    });
});
