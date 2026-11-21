import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Verdict, VerdictSkeleton } from '../verdict';

function heightOf(id: string): number {
    return document.querySelector(`#${id}`)?.getBoundingClientRect().height ?? 0;
}

describe('verdict', () => {
    test('a skeleton verdict stands as tall as a verdict of one title line and one line', async () => {
        // Given - a page verdict and a tab verdict, each beside its skeleton
        let heights: number[] = [];
        await component.render(
            <div>
                <div id="page">
                    <Verdict
                        line="Every machine answers"
                        size="page"
                        title="All is well."
                        tone="ok"
                        word="Up"
                    />
                </div>
                <div id="pageSkeleton">
                    <VerdictSkeleton size="page" />
                </div>
                <div id="tab">
                    <Verdict
                        line="relay does not answer"
                        size="tab"
                        title="relay is down."
                        tone="bad"
                        word="Down"
                    />
                </div>
                <div id="tabSkeleton">
                    <VerdictSkeleton size="tab" />
                </div>
            </div>,
            async () => {
                // When - all four are measured
                heights = ['page', 'pageSkeleton', 'tab', 'tabSkeleton'].map(heightOf);
            },
        );

        // Then - each skeleton is its verdict's height
        const [page, pageSkeleton, tab, tabSkeleton] = heights;
        expect(page).toBeGreaterThan(0);
        expect([pageSkeleton, tabSkeleton]).toStrictEqual([page, tab]);
    });

    test('a skeleton verdict with no line stands as tall as a verdict of a title alone', async () => {
        // Given - a verdict with no line, and its skeleton
        let heights: number[] = [];
        await component.render(
            <div>
                <div id="title">
                    <Verdict line="" size="tab" title="Every sign-in holds." tone="ok" word="Up" />
                </div>
                <div id="skeleton">
                    <VerdictSkeleton line={false} size="tab" />
                </div>
            </div>,
            async () => {
                // When - both are measured
                heights = [heightOf('title'), heightOf('skeleton')];
            },
        );

        // Then - their heights are one
        const [title, skeleton] = heights;
        expect(title).toBeGreaterThan(0);
        expect(skeleton).toBe(title);
    });

    test('a skeleton verdict says nothing to assistive tech', async () => {
        // Given - a skeleton verdict
        const result = await component.render(<VerdictSkeleton size="page" />, async () => {});

        // Then - the tree holds nothing
        expect(result.tree.text).toBe('');
    });
});
