import type { Story } from './story';
import type { VerdictSkeletonProps } from './verdict';

export const VERDICT_SKELETON_STORIES: readonly Story<VerdictSkeletonProps>[] = [
    { name: 'page', props: { size: 'page' } },
    { name: 'tab', props: { size: 'tab' } },
    { name: 'tab, a title alone', props: { line: false, size: 'tab' } },
];
