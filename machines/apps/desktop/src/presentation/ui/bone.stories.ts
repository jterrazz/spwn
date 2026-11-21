import type { BoneProps } from './bone';
import type { Story } from './story';

export const BONE_STORIES: readonly Story<BoneProps>[] = [
    { name: 'short', props: { width: 'short' } },
    { name: 'mid', props: { width: 'mid' } },
    { name: 'long', props: { width: 'long' } },
    { name: 'wide', props: { width: 'wide' } },
];
