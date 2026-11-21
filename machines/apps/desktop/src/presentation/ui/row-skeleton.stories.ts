import type { RowSkeletonProps } from './row-skeleton';
import type { Story } from './story';

export const ROW_SKELETON_STORIES: readonly Story<RowSkeletonProps>[] = [
    { name: 'a title and a sentence', props: {} },
    { name: 'a title alone', props: { sentence: false } },
    { name: 'a lead icon and a trail', props: { lead: 'icon', trail: 'fit' } },
    { name: 'slim', props: { sentence: false, size: 'slim' } },
    { name: 'passwords columns', props: { lead: 'icon', template: 'passwords' } },
];
