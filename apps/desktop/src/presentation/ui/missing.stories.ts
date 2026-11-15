import type { ComponentProps } from 'react';

import type { Missing, MissingMark } from './missing';
import type { Story } from './story';

export const MISSING_STORIES: readonly Story<ComponentProps<typeof Missing>>[] = [
    {
        name: 'a field no call gives',
        props: { children: 'The accounts it shares and the ports it opens' },
    },
    {
        name: 'long text',
        props: {
            children:
                'What no call answers is named here, as the field it lacks, however long the field is to name in full',
        },
    },
];

export const MISSING_MARK_STORIES: readonly Story<ComponentProps<typeof MissingMark>>[] = [
    { name: 'its own word', props: {} },
    { name: 'another word', props: { children: 'Not measured' } },
];
