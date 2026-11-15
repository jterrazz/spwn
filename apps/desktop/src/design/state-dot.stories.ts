import type { ComponentProps } from 'react';

import { TONES } from '../tone';
import type { StateDot } from './state-dot';
import type { Story } from './story';

export const STATE_DOT_STORIES: readonly Story<ComponentProps<typeof StateDot>>[] = TONES.map(
    (tone) => ({
        name: tone,
        props: { tone, word: tone },
    }),
);
