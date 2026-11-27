import { WIKI_GRAPH, WIKI_WORDS } from '../canvas/samples/wiki';
import { Icon } from '../icon';
import type { Story } from '../story';
import type { CollapsedBarProps } from './collapsed-bar';
import type { OsSpaceSceneProps } from './os-space';
import { BAR_WORDS, STRIP_WORDS } from './os-space-sample';
import type { StripProps } from './strip';

const nothing = (): void => undefined;

const BOUNDARIES: StripProps = {
    icon: <Icon name="book" />,
    line: 'What Spwn is not, and what each of its parts refuses to be, for anyone weighing a feature that sits near an edge.',
    meta: ['Produit', 'E1 Produit', '4 sections'],
    needs: 3,
    onOpen: nothing,
    title: 'Boundaries',
    words: STRIP_WORDS,
};

export const STRIP_STORIES: readonly Story<StripProps>[] = [
    { name: 'title and line', props: BOUNDARIES },
    { name: 'title alone', props: { ...BOUNDARIES, line: undefined } },
    { name: 'nothing waits', props: { ...BOUNDARIES, needs: 0 } },
    {
        name: 'long line',
        props: {
            ...BOUNDARIES,
            line: `${BOUNDARIES.line ?? ''} ${BOUNDARIES.line ?? ''}`,
            title: 'Boundaries, what Spwn is not and what each of its parts refuses to be',
        },
    },
];

const BAR: CollapsedBarProps = {
    graph: WIKI_GRAPH,
    onLower: nothing,
    onRead: nothing,
    path: ['S:product', 'P:product/boundaries', 'C:product/boundaries:0'],
    reads: 1,
    shared: BAR_WORDS,
    words: WIKI_WORDS,
};

export const COLLAPSED_BAR_STORIES: readonly Story<CollapsedBarProps>[] = [
    { name: 'a section read, three deep', props: BAR },
    {
        name: 'the last section read',
        props: { ...BAR, path: ['S:product', 'P:product/boundaries', 'C:product/boundaries:3'] },
    },
    { name: 'the wiki home read', props: { ...BAR, path: [] } },
];

export const OS_SPACE_STORIES: readonly Story<OsSpaceSceneProps>[] = [
    { name: 'lowered, a page open', props: { start: ['S:product', 'P:product/boundaries'] } },
    { name: 'lowered, at the root', props: { start: [] } },
    {
        name: 'raised, a section read',
        props: {
            start: ['S:product', 'P:product/boundaries', 'C:product/boundaries:2'],
            startRaised: true,
        },
    },
];
