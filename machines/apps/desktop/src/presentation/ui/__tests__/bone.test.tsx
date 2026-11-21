import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Bone } from '../bone';

function heightOf(id: string): number {
    return document.querySelector(`#${id}`)?.getBoundingClientRect().height ?? 0;
}

describe('Bone', () => {
    test('a bone holds the line it stands in as tall as its text would', async () => {
        // Given - one paragraph of text and one holding a bone instead
        let heights: number[] = [];
        await component.render(
            <div>
                <p id="text">Memory free</p>
                <p id="bone">
                    <Bone width="long" />
                </p>
            </div>,
            async () => {
                // When - both are measured
                heights = [heightOf('text'), heightOf('bone')];
            },
        );

        // Then - the bone's line is the text's height
        const [text, bone] = heights;
        expect(text).toBeGreaterThan(0);
        expect(bone).toBe(text);
    });

    test('a bone says nothing to assistive tech', async () => {
        // Given - a bone alone
        const result = await component.render(<Bone />, async () => {});

        // Then - the tree holds nothing
        expect(result.tree.text).toBe('');
    });
});
