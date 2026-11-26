import type { Request } from '@jterrazz/station-api';
import { button, component, content, dialog, field, file, region, within } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../composition/compose';
import { documentPresence } from '../../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../../infrastructure/outbound/engine-link/client';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../../infrastructure/outbound/engine-replay/replay';
import type { Stage } from '../../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../../dependencies/dependencies';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../registry';
import { AddPasswordSheet } from '../add-sheet';
import { PasswordsPage } from '../passwords';

const recordingFor = await recordingsOf('two-macs', 'one-mac');

const SHEET = dialog('Add a password');
const VALUE = field('Value');
const LABEL = field('Label');
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
        <EngineProvider
            dependencies={{
                engine: new Client(stage.transport),
                opener: NO_OPENER,
                presence: documentPresence(),
            }}
        >
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
    test("the vault's bar offers Add a password, which opens its sheet", async () => {
        // Given - the window on Passwords over two-macs
        const dependencies = composeReplay({ recording: recordingFor('two-macs') });
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={{ page: 'passwords' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                // When - Add a password is pressed in the bar
                await visitor.click(button('Add a password'));

                // Then - the sheet opens on the engine's offer, its intro under the title
                await visitor.see(within(SHEET, content('It is kept in', { exact: false })));
                await visitor.see(within(SHEET, button('Typed', { exact: false })));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('the sheet draws the two ways, the sign-in the Mac can keep with its sentence and a proposed label, and no value', async () => {
        // Given - the sheet over two-macs, gh the one sign-in this Mac can keep
        const seen = unseen();
        const result = await component.render(vaultOver(), async (visitor) => {
            // When - it is drawn
            await visitor.see(within(SHEET, content('Kept as an OAuth token', { exact: false })));
            look(seen);

            // Then - both ways, the sign-in's card, and no kind to choose
            await visitor.see(within(SHEET, button('Typed', { exact: false })));
            await visitor.see(within(SHEET, button('gh', { exact: false })));
            await visitor.gone(within(SHEET, button('fine-grained token')));
        });
        expect(result.errors.text).toBe('');
        expect(seen.input).toBeNull();
        await expect(result.tree).toMatch(file('./__expected__/two-macs-add.aria.yaml'));
    });

    test('a sign-in kept adds with its program alone: no kind, no value', async () => {
        // Given - the sheet over two-macs
        const stage = stagedReplay(recordingFor('two-macs'));
        const result = await component.render(vaultOver(stage), async (visitor) => {
            await visitor.see(within(SHEET, content('Kept as an OAuth token', { exact: false })));

            // When - the label is typed over the proposal, and it is added
            await visitor.fill(within(SHEET, LABEL), 'GitHub sign-in');
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
                in: { From: 'sign-in', Label: 'GitHub sign-in', Program: 'gh' },
            }),
        ]);
    });

    test('with no sign-in to keep the way stays, says so, and Add stays off', async () => {
        // Given - the sheet over one-mac, where no program is signed in
        const stage = stagedReplay(recordingFor('one-mac'));
        const result = await component.render(vaultOver(stage), async (visitor) => {
            // When - it is drawn
            await visitor.see(
                within(SHEET, content('Sign in to a program on this Mac first', { exact: false })),
            );

            // Then - both ways are there
            await visitor.see(within(SHEET, button('Typed', { exact: false })));
        });
        expect(result.errors.text).toBe('');
        expect(addsOf(stage)).toStrictEqual([]);
        await expect(result.tree).toMatch(file('./__expected__/one-mac-add.aria.yaml'));
    });

    test('a typed program offers its kinds, and none when it has one', async () => {
        // Given - the sheet over two-macs
        const result = await component.render(vaultOver(), async (visitor) => {
            // When - the typed way is picked, then npm
            await visitor.click(within(SHEET, button('Typed', { exact: false })));
            await visitor.click(within(SHEET, button('npm')));

            // Then - its kinds, and its password field
            await visitor.see(within(SHEET, button('fine-grained token')));
            await visitor.see(within(SHEET, VALUE));
        });
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-add-typed.aria.yaml'));
    });

    test('a program with one kind draws no kind row', async () => {
        // Given - the sheet over two-macs, the typed way
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('Typed', { exact: false })));

            // When - Codex is picked
            await visitor.click(within(SHEET, button('Codex')));

            // Then - no kind to choose
            await visitor.gone(within(SHEET, button('API key')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a program with two kinds offers both: Claude takes an OAuth token or an API key', async () => {
        // Given - the sheet over two-macs, the typed way
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('Typed', { exact: false })));

            // When - Claude is picked
            await visitor.click(within(SHEET, button('Claude')));

            // Then - both its kinds are offered
            await visitor.see(within(SHEET, button('OAuth token')));
            await visitor.see(within(SHEET, button('API key')));
        });
        expect(result.errors.text).toBe('');
    });

    // invariant: the three conditions of a typed password: an uncontrolled password field, one request, emptied.
    test('a typed value lives in the field alone, leaves in one request, and the field is emptied', async () => {
        // Given - the sheet over two-macs, Cloudflare typed under its default kind, its value typed
        const stage = stagedReplay(recordingFor('two-macs'));
        const seen = unseen();
        const result = await component.render(vaultOver(stage), async (visitor) => {
            await visitor.click(within(SHEET, button('Typed', { exact: false })));
            await visitor.click(within(SHEET, button('Cloudflare')));
            await visitor.fill(within(SHEET, LABEL), 'Cloudflare deploy');
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
                    Kind: 'token',
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
            await visitor.click(within(SHEET, button('Typed', { exact: false })));
            await visitor.click(within(SHEET, button('npm')));
            await visitor.fill(within(SHEET, LABEL), 'GitHub deploy');
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

    test('an empty label is refused on the label, in the engine words', async () => {
        // Given - the sheet over two-macs, npm typed with a value and no label
        const seen = unseen();
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('Typed', { exact: false })));
            await visitor.click(within(SHEET, button('npm')));
            await visitor.fill(within(SHEET, VALUE), TYPED);
            look(seen);

            // When - it is added
            await visitor.click(within(SHEET, button('Add password')));

            // Then - the engine's refusal, in its words
            await visitor.see(within(SHEET, content('a password needs a label', { exact: false })));
        });
        expect(result.errors.text).toBe('');
        expect(seen.input?.value).toBe('');
        await expect(result.tree).toMatch(
            file('./__expected__/two-macs-add-empty-label.aria.yaml'),
        );
    });

    test('an empty value is refused on the value field', async () => {
        // Given - the sheet over two-macs, npm typed with nothing in its field
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('Typed', { exact: false })));
            await visitor.click(within(SHEET, button('npm')));
            await visitor.fill(within(SHEET, LABEL), 'npm deploy');

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
        // Given - the sheet over two-macs, a value typed for the first program
        const seen = unseen();
        const result = await component.render(vaultOver(), async (visitor) => {
            await visitor.click(within(SHEET, button('Typed', { exact: false })));
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
        // Given - the sheet over two-macs, a value typed
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
