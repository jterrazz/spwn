import type { ComponentProps } from 'react';

import type { Story } from './story';
import type { TypeTag } from './type-tag';

export const TYPE_TAG_STORIES: readonly Story<ComponentProps<typeof TypeTag>>[] = [
    { name: 'a kind', props: { children: 'manager' } },
    { name: 'another', props: { children: 'setting' } },
    { name: 'long text', props: { children: 'a kind long enough to run on' } },
];
