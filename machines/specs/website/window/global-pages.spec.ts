import { button, content, dialog, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const THIS_MAC = region('macbook-pro');
const MINI = region('mac-mini');
const MINI_ENGINE = within(MINI, region('Engine'));
const AIR_ENGINE = within(region('macbook-air'), region('Engine'));

describe('global pages', () => {
    test('A12 the sidebar opens Storage beside it, every Mac in its own section', async () => {
        // Given - the window on Home over two-macs
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            // When - the sidebar's Storage is pressed
            await visitor.click(within(sidebar, button('Storage')));

            // Then - This Mac's verdict and the Mac mini's, each in its section, the sidebar still there
            await visitor.see(within(THIS_MAC, heading('3.0 GB can be freed.')));
            await visitor.see(within(MINI, heading('Nothing to free.')));
            await visitor.see(within(sidebar, heading('Machine')));
        });
        expect(result.errors.text).toBe('');
    });

    test("A12 the sidebar opens Engine beside it, each Mac's engine with its state and version", async () => {
        // Given - the window on Home over stories, whose engine has no release out and whose macbook-air sleeps
        const result = await website.visit('/?world=stories&page=home', async (visitor) => {
            // When - the sidebar's Engine is pressed
            await visitor.click(within(sidebar, button('Engine')));

            // Then - This Mac's release and engine, the Mac mini's engine and version, the Air asleep
            await visitor.see(within(THIS_MAC, heading('The latest release is not known yet.')));
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
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            await visitor.see(content('2 things need you.'));

            // When - Storage, then Engine, is pressed in the sidebar, then Back twice and Forward once
            await visitor.click(within(sidebar, button('Storage')));
            await visitor.see(within(MINI, heading('Nothing to free.')));
            await visitor.click(within(sidebar, button('Engine')));
            await visitor.see(within(MINI, region('Engine')));
            await visitor.click(button('Back'));
            await visitor.see(within(MINI, heading('Nothing to free.')));
            await visitor.click(button('Back'));
            await visitor.see(content('2 things need you.'));
            await visitor.click(button('Forward'));

            // Then - Storage again, with the sidebar
            await visitor.see(within(THIS_MAC, heading('3.0 GB can be freed.')));
            await visitor.see(within(sidebar, heading('Machine')));
        });
        expect(result.errors.text).toBe('');
    });

    test('A14 a link to Engine opens it beside the sidebar, its toggle drawn', async () => {
        // Given - the window opened on the Engine page over two-macs
        const result = await website.visit('/?world=two-macs&page=engine', async (visitor) => {
            // When - the page is read
            await visitor.see(within(MINI, region('Engine')));

            // Then - the sidebar and its toggle are there
            await visitor.see(within(sidebar, button('Engine')));
            await visitor.see(button('Show or hide the sidebar'));
        });
        expect(result.errors.text).toBe('');
    });

    test("A15 a link to a Mac's Storage tab opens Storage, the Mac's disk in its section", async () => {
        // Given - the window opened on This Mac's Storage tab over two-macs, as a relaunch or an old link does
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Storage',
            async (visitor) => {
                // When - the page is read
                await visitor.see(within(THIS_MAC, heading('3.0 GB can be freed.')));

                // Then - the Storage page, with the Mac mini's section beside This Mac's
                await visitor.see(within(MINI, heading('Nothing to free.')));
                await visitor.gone(heading('macbook-pro online'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("A16 the sidebar's + opens New machine", async () => {
        // Given - the window on the Apps shelf over two-macs, which has no New machine of its own
        const result = await website.visit(
            '/?world=two-macs&page=catalog-apps',
            async (visitor) => {
                await visitor.see(region('Apps'));

                // When - the + beside the space's name is pressed
                await visitor.click(within(sidebar, button('New machine')));

                // Then - the New machine sheet
                await visitor.see(dialog('New machine on mac-mini'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
