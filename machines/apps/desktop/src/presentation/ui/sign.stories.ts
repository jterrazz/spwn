import type { SignProps } from './sign';
import type { Story } from './story';

export const SIGN_STORIES: readonly Story<SignProps>[] = [
    { name: 'in the bar', props: {} },
    { name: "at the head of the core's panel", props: { size: 'large' } },
];
