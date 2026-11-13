import { button, component, content } from '@jterrazz/test';
import { describe, expect, test, vi } from 'vitest';

import { CopyButton } from '../copy-button';
import { Value } from '../value';

const SHOWN = 'station engine start';
const SECRET = `x9${'k7'.repeat(20)}`;

describe('CopyButton', () => {
    test('a value drawn as given is copied exactly as it is drawn', async () => {
        // Given - a value drawn beside its Copy, and a clipboard that takes it
        const written = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue();
        const result = await component.render(
            <>
                <Value>{SHOWN}</Value>
                <CopyButton name={`Copy ${SHOWN}`} text={SHOWN} />
            </>,
            async (visitor) => {
                await visitor.see(content(SHOWN));

                // When - its Copy is clicked
                await visitor.click(button(`Copy ${SHOWN}`));

                // Then - the button says it copied
                await visitor.see(button('Copied'));
            },
        );

        // Then - the clipboard holds the very text the screen draws
        expect(result.tree.text).toContain(SHOWN);
        expect(written).toHaveBeenCalledWith(SHOWN);
    });

    test('a withheld value offers no Copy', async () => {
        // Given - a value shaped like a secret, drawn beside its Copy
        const result = await component.render(
            <>
                <Value>{SECRET}</Value>
                <CopyButton name="Copy it" text={SECRET} />
            </>,
            async (visitor) => {
                await visitor.see(content('Withheld'));
            },
        );

        // Then - the value reads Withheld, and no button would put it on the clipboard
        expect(result.tree.text).not.toContain(SECRET);
        expect(result.tree.text).not.toContain('button');
    });
});
