import { component } from '@jterrazz/test';
import type { ReactNode } from 'react';
import { describe, expect, test } from 'vitest';

import { Row } from '../row';
import { RowSkeleton } from '../row-skeleton';

function heightOf(id: string): number {
    return document.querySelector(`#${id}`)?.getBoundingClientRect().height ?? 0;
}

async function heightsOf(answered: ReactNode, skeleton: ReactNode): Promise<number[]> {
    let heights: number[] = [];
    await component.render(
        <div>
            <div id="answered">{answered}</div>
            <div id="skeleton">{skeleton}</div>
        </div>,
        async () => {
            heights = [heightOf('answered'), heightOf('skeleton')];
        },
    );
    return heights;
}

describe('RowSkeleton', () => {
    test('a skeleton row stands as tall as the row with a sentence it stands for', async () => {
        // Given - a row of a title and a sentence, and its skeleton
        // When - both are measured
        const [answered, skeleton] = await heightsOf(
            <Row sentence="since 9:41" title="GitHub" trail={<span>Sign out</span>} />,
            <RowSkeleton trail="fit" />,
        );

        // Then - their heights are one
        expect(answered).toBeGreaterThan(0);
        expect(skeleton).toBe(answered);
    });

    test('a skeleton row with no sentence stands as tall as a row of a title alone', async () => {
        // Given - a row of a title alone, and its skeleton
        // When - both are measured
        const [answered, skeleton] = await heightsOf(
            <Row title="GitHub" />,
            <RowSkeleton sentence={false} />,
        );

        // Then - their heights are one
        expect(answered).toBeGreaterThan(0);
        expect(skeleton).toBe(answered);
    });

    test('a skeleton row says nothing to assistive tech', async () => {
        // Given - a skeleton row with a lead and a trail
        const result = await component.render(
            <RowSkeleton lead="icon" trail="fit" />,
            async () => {},
        );

        // Then - the tree holds nothing
        expect(result.tree.text).toBe('');
    });
});
