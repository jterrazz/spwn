import type { Request } from '@jterrazz/station-api';
import { button, component, content, dialog, field, file, region, within } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport, stagedReplay } from '../../../engine/replay';
import type { Stage } from '../../../engine/replay';
import { recordingsOf } from '../../../fixtures/recordings';
import { Window } from '../../../shell/window';
import { PAGES } from '../../registry';
import { AddPasswordSheet } from '../add-sheet';
import { PasswordsPage } from '../passwords';

const recordingFor = await recordingsOf('two-macs');

const SHEET = dialog('Add a password');
const VALUE = field('Value');
const TYPED = 'good';

// value: what a test saw of the password field while the sheet was open: the element, and the page's markup then.
type Seen = { input: HTMLInputElement | null; drawn: string };

function Vault() {
    const [adding, setAdding] = useState(true);
    return (
        <>
            <PasswordsPage go={() => undefined} opened={undefined} />
            {adding ? (
                <AddPasswordSheet
                    onClose={() => {
                        setAdding(false);
                    }}
                />
            ) : null}
        </>
    );
}

function vaultOver(stage: Stage = stagedReplay(recordingFor('two-macs'))) {
    return (
        <EngineProvider client={new Client(stage.transport)}>
            <Vault />
        </EngineProvider>
    );
}

function addsOf(stage: Stage): readonly Request[] {
    return stage.asked().filter((request) => 'call' in request && request.call === 'vault.add');
}

function look(seen: Seen): void {
    seen.input = document.querySelector<HTMLInputElement>('input[type="password"]');
    seen.drawn = document.body.innerHTML;
}

function unseen(): Seen {
    return { drawn: '', input: null };
}

// value: the field as drawn while it held the value (its attributes, whether the markup carried the value), and
// value: what it held once the sheet was done with it.
function fieldOf({ input, drawn }: Seen) {
    return {
        autocomplete: input?.getAttribute('autocomplete'),
        drawn: drawn.includes(TYPED),
        held: input?.value,
        spellcheck: input?.getAttribute('spellcheck'),
        type: input?.getAttribute('type'),
        valueAttribute: input?.hasAttribute('value'),
    };
}

describe('add password sheet', () => {
    test("the vault's toolbar offers Add a password, which opens its sheet", async () => {
        // Given - the window on Passwords over two-macs
        const client = new Client(replayTransport(recordingFor('two-macs')));
        const result = await component.render(
            <EngineProvider client={client}>
                <Window initial={{ page: 'passwords' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                // When - Add a password is pressed in the toolbar
                await visitor.click(button('Add a password'));

                // Then - the sheet opens on the programs the engine lists
                await visitor.see(within(SHEET, button('Cloudflare')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('the sheet draws the programs, the ways in, the kinds and the label, and no value from a sign-in', async () => {
        // Given - the sheet over two-macs, gh the first program the engine lists
        const seen = unseen();
        const result = await component.render(vaultOver(), async (visitor) => {
            // When - it is drawn
            await visitor.see(within(SHEET, content('Station takes the token', { exact: false })));
            look(seen);

            // Then - the sheet offers both ways for gh, and every kind
            await visitor.see(within(SHEET, button('Typed', { exact: false })));
            await visitor.see(within(SHEET, button('fine-grained token')));
        });
        expect(result.errors.text).toBe('');
        expect(seen.input).toBeNull();
        await expect(result.tree).toMatch(file('./__expected__/two-macs-add.aria.yaml'));
    });

    test('a program that captures adds from its sign-in by default, and sends no value', async () => {
        // Given - the sheet over two-macs
        const stage = stagedReplay(recordingFor('two-macs'));
        const result = await component.render(vaultOver(stage), async (visitor) => {
            await visitor.see(within(SHEET, content('Station takes the token', { exact: false })));

            // When - its kind and label are given, and it is added
            await visitor.click(within(SHEET, button('token')));
            await visitor.fill(within(SHEET, field('Label')), 'GitHub sign-in');
            await visitor.click(within(SHEET, button('Add password')));

            // Then - the sheet is gone, and the vault lists the new password
            await visitor.gone(SHEET);
            await visitor.see(
                within(region('Passwords'), button('GitHub sign-in', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
        expect(addsOf(stage)).toStrictEqual([
            expect.objectContaining({
                in: { From: 'sign-in', Kind: 'token', Label: 'GitHub sign-in', Program: 'gh' },
            }),
        ]);
    });

    test('a program that cannot be captured offers only the typed way', async () => {
        // Given - the sheet over two-macs
        const result = await component.render(vaultOver(), async (visitor) => {
            // When - npm is picked
            await visitor.click(within(SHEET, button('npm')));

            // Then - the typed way alone, and its password field
            await visitor.see(within(SHEET, VALUE));
            await visitor.gone(within(SHEET, button('From a sign-in', { exact: false })));
        });
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-add-typed.aria.yaml'));
    });

    // invariant: the three conditions of a typed password: an uncontrolled password field, one request, emptied.
    test('a typed value lives in the field alone, leaves in one request, and the field is emptied', async () => {
        // Given - the sheet over two-macs, Cloudflare picked as an API key, its value typed
        const stage = stagedReplay(recordingFor('two-macs'));
        const seen = unseen();
        const result = await component.render(vaultOver(stage), async (visitor) => {
            await visitor.click(within(SHEET, button('Cloudflare')));
            await visitor.click(within(SHEET, button('API key')));
            await visitor.fill(within(SHEET, field('Label')), 'Cloudflare deploy');
            await visitor.fill(within(SHEET, VALUE), TYPED);
            look(seen);

            // When - it is added
            await visitor.click(within(SHEET, button('Add password')));

            // Then - the sheet is gone, and the vault lists the new password
            await visitor.gone(SHEET);
            await visitor.see(
                within(region('Passwords'), button('Cloudflare deploy', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
        expect(fieldOf(seen)).toStrictEqual({
            autocomplete: 'off',
            drawn: false,
            held: '',
            spellcheck: 'false',
            type: 'password',
            valueAttribute: false,
        });
        expect(result.html.text).not.toContain(TYPED);
        expect(addsOf(stage)).toStrictEqual([
            expect.objectContaining({
                in: {
                    From: 'typed',
                    Kind: 'api-key',
                    Label: 'Cloudflare deploy',
                    Program: 'wrangler',
                    Value: TYPED,
                },
            }),
        ]);
    });

    test('a refused label is said under the label, the sheet stays open and the value is gone', async () => {
        // Given - the sheet over two-macs, npm typed under a label the vault already holds
        const seen = unseen();
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('npm')));
            await visitor.click(within(SHEET, button('token')));
            await visitor.fill(within(SHEET, field('Label')), 'GitHub deploy');
            await visitor.fill(within(SHEET, VALUE), TYPED);
            look(seen);

            // When - it is added
            await visitor.click(within(SHEET, button('Add password')));

            // Then - the engine's refusal, in its words, and the sheet still open
            await visitor.see(
                within(SHEET, content('a password is already labeled', { exact: false })),
            );
            await visitor.see(within(SHEET, VALUE));
        });
        expect(result.errors.text).toBe('');
        expect(seen.input?.value).toBe('');
        expect(result.html.text).not.toContain(TYPED);
        await expect(result.tree).toMatch(
            file('./__expected__/two-macs-add-label-taken.aria.yaml'),
        );
    });

    test('an empty value is refused on the value field', async () => {
        // Given - the sheet over two-macs, npm typed with nothing in its field
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('npm')));
            await visitor.click(within(SHEET, button('token')));
            await visitor.fill(within(SHEET, field('Label')), 'npm deploy');

            // When - it is added
            await visitor.click(within(SHEET, button('Add password')));

            // Then - the engine's refusal, in its words
            await visitor.see(
                within(SHEET, content('a typed password cannot be empty', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(
            file('./__expected__/two-macs-add-empty-value.aria.yaml'),
        );
    });

    test('cancel empties the field before the sheet goes', async () => {
        // Given - the sheet over two-macs, a value typed for npm
        const seen = unseen();
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('npm')));
            await visitor.fill(within(SHEET, VALUE), TYPED);
            look(seen);

            // When - Cancel is clicked
            await visitor.click(within(SHEET, button('Cancel')));

            // Then - the sheet is gone
            await visitor.gone(SHEET);
        });
        expect(result.errors.text).toBe('');
        expect(seen.input?.value).toBe('');
    });

    test('a way that sends no value empties the field it leaves', async () => {
        // Given - the sheet over two-macs, a value typed for gh
        const seen = unseen();
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('Typed', { exact: false })));
            await visitor.fill(within(SHEET, VALUE), TYPED);
            look(seen);

            // When - the sign-in is picked again
            await visitor.click(within(SHEET, button('From a sign-in', { exact: false })));

            // Then - the field is gone
            await visitor.gone(within(SHEET, VALUE));
        });
        expect(result.errors.text).toBe('');
        expect(seen.input?.value).toBe('');
    });
});
