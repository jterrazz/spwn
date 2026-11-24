import { describe, expect, test } from 'vitest';

import { imagesOn } from '../images-on';

describe('images-on', () => {
    test("a Mac's section lists the images built on it alone", () => {
        // Given - three images, two on book and one on mini
        const images = [
            { Mac: 'book', Name: 'base' },
            { Mac: 'mini', Name: 'gate' },
            { Mac: 'book', Name: 'gate' },
        ];

        // When - the images on book are kept
        const kept = imagesOn(images, 'book');

        // Then - both of book's, in the engine's order
        expect(kept.map((image) => image.Name)).toStrictEqual(['base', 'gate']);
    });
});
