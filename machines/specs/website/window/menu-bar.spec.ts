import { button, content, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const menuBar = region('Menu bar');
const ON = button('On');
const OFF = button('Off');
const PRESSED_ON = /aria-pressed="true"[^>]*>On<\/button>/u;
const PRESSED_OFF = /aria-pressed="true"[^>]*>Off<\/button>/u;

describe('menu bar', () => {
    test('B1 a hidden menu bar is shown by its switch, which then reads On', async () => {
        // Given - the proof build whose menu bar is hidden
        const result = await website.visit(
            '/?world=stories&page=settings&menubar=hidden',
            async (visitor) => {
                await visitor.see(within(menuBar, content('Show in menu bar')));

                // When - On is pressed
                await visitor.click(within(menuBar, ON));
                await visitor.see(within(menuBar, content('Spwn Menu, at every login')));
            },
        );

        // Then - On is the one pressed
        expect(result.errors.text).toBe('');
        expect(result.html.text).toMatch(PRESSED_ON);
    });

    test('B2 a shown menu bar is hidden by its switch, which then reads Off', async () => {
        // Given - the proof build whose menu bar is shown
        const result = await website.visit(
            '/?world=stories&page=settings&menubar=shown',
            async (visitor) => {
                await visitor.see(within(menuBar, content('Show in menu bar')));

                // When - Off is pressed
                await visitor.click(within(menuBar, OFF));
                await visitor.see(within(menuBar, content('Spwn Menu, at every login')));
            },
        );

        // Then - Off is the one pressed
        expect(result.errors.text).toBe('');
        expect(result.html.text).toMatch(PRESSED_OFF);
    });

    test('B3 a menu bar macOS holds for approval says where to allow it', async () => {
        // Given - the proof build whose shown menu bar waits for the owner's approval
        const result = await website.visit(
            '/?world=stories&page=settings&menubar=approval',
            async (visitor) => {
                // When - On is pressed
                await visitor.click(within(menuBar, ON));

                // Then - the sentence naming System Settings, and On pressed
                await visitor.see(
                    within(
                        menuBar,
                        content(
                            'macOS waits for you: allow Spwn in System Settings, General, Login Items.',
                        ),
                    ),
                );
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).toMatch(PRESSED_ON);
    });

    test("B4 a refused switch says so with the shell's message, and stays Off", async () => {
        // Given - the proof build whose menu bar macOS refuses to show
        const result = await website.visit(
            '/?world=stories&page=settings&menubar=refusing',
            async (visitor) => {
                // When - On is pressed
                await visitor.click(within(menuBar, ON));

                // Then - the refusal with the shell's message, and Off still pressed
                await visitor.see(within(menuBar, content('The menu bar could not be switched.')));
                await visitor.see(
                    within(menuBar, content('Operation not permitted', { exact: false })),
                );
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).toMatch(PRESSED_OFF);
    });

    test('B5 a menu bar that is off says so beside its switch, which reads Off', async () => {
        // Given - the proof build whose menu bar is hidden
        const result = await website.visit(
            '/?world=stories&page=settings&menubar=hidden',
            async (visitor) => {
                // When - Settings is drawn
                await visitor.see(within(menuBar, content('Show in menu bar')));

                // Then - the line that says it is off
                await visitor.see(
                    within(menuBar, content('Spwn Menu is off: it is not in the menu bar.')),
                );
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).toMatch(PRESSED_OFF);
    });

    test('B6 a menu bar the install showed and macOS holds for approval says where to allow it', async () => {
        // Given - the proof build whose menu bar the install registered, waiting for the owner's approval
        const result = await website.visit(
            '/?world=stories&page=settings&menubar=waiting',
            async (visitor) => {
                // When - Settings is drawn, nothing pressed
                await visitor.see(within(menuBar, content('Show in menu bar')));

                // Then - the sentence naming System Settings
                await visitor.see(
                    within(
                        menuBar,
                        content(
                            'macOS waits for you: allow Spwn in System Settings, General, Login Items.',
                        ),
                    ),
                );
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).toMatch(PRESSED_ON);
    });
});
