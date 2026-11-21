import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Tile } from '../../ui/index';
import { Placeheld, Skeleton, SkeletonHead, SkeletonPanel, SkeletonTile } from '../skeleton';

function heightOf(id: string): number {
    return document.querySelector(`#${id}`)?.getBoundingClientRect().height ?? 0;
}

describe('skeleton', () => {
    test('a skeleton is busy, and says the one reading sentence while its bones stay silent', async () => {
        // Given - a page head and a panel of three rows, not answered yet
        const result = await component.render(
            <Skeleton what="the ports of macbook-pro">
                <SkeletonPanel count={3} title="What listens on this Mac" trail="fit" />
            </Skeleton>,
            async () => {},
        );

        // Then - the region is busy and the tree holds the sentence alone
        expect(document.querySelector('[aria-busy="true"]')).not.toBeNull();
        expect(result.tree.text).toBe('- status: Reading the ports of macbook-pro…');
    });

    test('a skeleton head is busy and says what it reads', async () => {
        // Given - the head of a page not answered yet
        const result = await component.render(
            <SkeletonHead size="tab" what="macbook-pro" />,
            async () => {},
        );

        // Then - the tree holds the reading sentence alone
        expect(result.tree.text).toBe('- status: Reading macbook-pro…');
    });

    test('a skeleton tile keeps its label and stands as tall as an answered tile', async () => {
        // Given - a CPU tile answered and one not answered yet
        let heights: number[] = [];
        await component.render(
            <div>
                <div id="answered">
                    <Tile foot="of 8 cores" label="CPU" value="12" />
                </div>
                <div id="skeleton">
                    <SkeletonTile label="CPU" />
                </div>
            </div>,
            async () => {
                // When - both are measured
                heights = [heightOf('answered'), heightOf('skeleton')];
            },
        );

        // Then - their heights are one
        const [answered, skeleton] = heights;
        expect(answered).toBeGreaterThan(0);
        expect(skeleton).toBe(answered);
    });

    test('an answer held while another is read is dimmed and busy, and an answer shown is neither', async () => {
        // Given - the same answer drawn held and drawn as shown
        let seen: string[] = [];
        await component.render(
            <div>
                <div id="held">
                    <Placeheld held>
                        <p>CPU 12 %</p>
                    </Placeheld>
                </div>
                <div id="shown">
                    <Placeheld held={false}>
                        <p>CPU 12 %</p>
                    </Placeheld>
                </div>
            </div>,
            async () => {
                // When - each wrapper's state is read
                seen = ['held', 'shown'].map((id) => {
                    const box = document.querySelector(`#${id} > div`);
                    const style = box ? getComputedStyle(box) : undefined;
                    return `${box?.getAttribute('aria-busy')} ${style?.display} ${style?.opacity}`;
                });
            },
        );

        // Then - the held one is busy, a box of its own and dimmed; the shown one adds no box
        expect(seen).toStrictEqual(['true flex 0.55', 'false contents 1']);
    });
});
