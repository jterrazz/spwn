import type { ComponentProps } from 'react';

import { TONES } from '../tone';
import type { Tone } from '../tone';
import type { StatePill } from './state-pill';
import type { Story } from './story';

const WORDS: Readonly<Record<Tone, string>> = {
    bad: 'Down',
    new: 'Creating',
    ok: 'Running',
    plain: 'Stopped',
    warn: 'Needs you',
};

export const STATE_PILL_STORIES: readonly Story<ComponentProps<typeof StatePill>>[] = [
    ...TONES.map((tone) => ({ name: tone, props: { tone, word: WORDS[tone] } })),
    ...TONES.map((tone) => ({
        name: `${tone}, quiet`,
        props: { quiet: true, tone, word: WORDS[tone] },
    })),
    { name: 'long text', props: { tone: 'warn', word: 'Needs a look before it can update' } },
    { name: 'as written', props: { asWritten: true, tone: 'warn', word: 'ends in 40 min' } },
];
