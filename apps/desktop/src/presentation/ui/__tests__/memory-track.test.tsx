import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { MemoryTrack } from '../memory-track';

describe('MemoryTrack', () => {
    test('a segment sits at its offset, as wide as its share, filled as far as it is used', async () => {
        // Given - a machine holding a sixth of its Mac, a quarter in, two thirds of it used
        const result = await component.render(
            <MemoryTrack
                label="Memory of linux-1"
                segments={[
                    { at: 0.25, identity: 'teal', key: 'linux-1', used: 0.66, width: 0.1667 },
                ]}
            />,
        );

        // Then - the three shares are handed to the CSS as they came
        expect(result.html.text).toContain('style="--at: 0.25; --w: 0.1667; --used: 0.66;"');
        expect(result.html.text).toContain('data-identity="teal"');
    });

    test('a segment past the track is held inside it, and a use past full is full', async () => {
        // Given - a segment starting at nine tenths, half the track wide, three times used
        const result = await component.render(
            <MemoryTrack
                label="Memory of over"
                segments={[{ at: 0.9, key: 'over', used: 3, width: 0.5 }]}
            />,
        );

        // Then - it ends where the track ends, and its fill is whole
        expect(result.html.text).toContain('style="--at: 0.9; --w: 0.1; --used: 1;"');
    });

    test('a share no total gave is drawn empty, never NaN', async () => {
        // Given - a segment measured against a total of zero
        const result = await component.render(
            <MemoryTrack
                label="Memory of nothing"
                segments={[
                    { at: Number.NaN, key: 'none', used: 0.5, width: Number.POSITIVE_INFINITY },
                ]}
            />,
        );

        // Then - it takes no room
        expect(result.html.text).toContain('style="--at: 0; --w: 0; --used: 0.5;"');
        expect(result.html.text).not.toMatch(/NaN|Infinity/u);
    });

    test('a share with no use known is reserved and left unfilled', async () => {
        // Given - a machine's share whose use the engine did not measure
        const result = await component.render(
            <MemoryTrack
                label="Memory of agent-7f"
                segments={[{ at: 0.0833, identity: 'blue', key: 'agent-7f', width: 0.1667 }]}
            />,
        );

        // Then - it is placed, and no use is handed to the CSS
        expect(result.html.text).toContain('style="--at: 0.0833; --w: 0.1667;"');
        expect(result.html.text).not.toContain('--used');
    });
});
