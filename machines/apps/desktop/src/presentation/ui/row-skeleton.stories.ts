import type { RowSkeletonProps } from './row-skeleton';
import type { Story } from './story';

export const ROW_SKELETON_STORIES: readonly Story<RowSkeletonProps>[] = [
    { name: 'a title and a sentence', props: {} },
    { name: 'a title alone', props: { sentence: false } },
    { name: 'a lead icon and a trail', props: { lead: 'icon', trail: 'fit' } },
    { density: 'compact', name: 'compact', props: { sentence: false } },
    { name: 'passwords columns', props: { lead: 'icon', template: 'passwords' } },
];
