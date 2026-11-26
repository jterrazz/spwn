import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { drawingOf, Illustration, ILLUSTRATIONS } from '../illustration';

describe('Illustration', () => {
    test('every drawing fits its frame, centred, and a failed boot is not drawn as a failed build', () => {
        // Given - every drawing a guide can name
        const drawings = ILLUSTRATIONS.map((name) => ({ name, ...drawingOf(name) }));

        // When - each one's reach is read against the frame its view box draws
        const overflowing = drawings
            .filter(({ reach }) => reach.width > 280 || reach.height > 190)
            .map(({ name }) => name);

        // Then - none leaves its frame, and the two failures are two drawings
        expect(overflowing).toStrictEqual([]);
        expect(drawingOf('boot-failed').shapes).not.toStrictEqual(drawingOf('build-failed').shapes);
    });

    test('a drawing is decoration: assistive tech reads nothing of it', async () => {
        // Given - every drawing, drawn together
        const result = await component.render(
            <>
                {ILLUSTRATIONS.map((name) => (
                    <Illustration key={name} name={name} />
                ))}
            </>,
        );

        // Then - the accessibility tree is empty, and each drawing is drawn
        expect(result.tree.text).toBe('');
        expect(result.html.text.match(/data-illustration="/gu)).toHaveLength(ILLUSTRATIONS.length);
    });
});
