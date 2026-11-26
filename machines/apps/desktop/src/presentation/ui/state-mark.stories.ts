import { TONES } from '../../domain/tone';
import type { Tone } from '../../domain/tone';
import type { StateMarkProps } from './state-mark';
import type { Story } from './story';

const WORDS: Readonly<Record<Tone, string>> = {
    bad: 'down',
    new: 'creating',
    ok: 'running',
    plain: 'stopped',
    warn: 'waits for you',
};

export const STATE_MARK_STORIES: readonly Story<StateMarkProps>[] = [
    ...TONES.map((tone) => ({ name: tone, props: { tone, word: WORDS[tone] } })),
    ...TONES.map((tone) => ({
        name: `${tone}, its word before`,
        props: { said: 'before' as const, tone, word: WORDS[tone] },
    })),
    ...TONES.map((tone) => ({
        name: `${tone}, named only`,
        props: { said: 'name' as const, tone, word: WORDS[tone] },
    })),
    { name: 'an outcome says its word', props: { said: 'after', tone: 'ok', word: 'done' } },
    { name: 'beside a sentence', props: { said: 'beside', tone: 'warn' } },
    { name: 'long text', props: { tone: 'warn', word: 'Needs a look before it can update' } },
];
