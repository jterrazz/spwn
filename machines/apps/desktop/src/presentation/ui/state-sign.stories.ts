import { TONES } from '../../domain/tone';
import type { StateSignProps } from './state-sign';
import type { Story } from './story';

export const STATE_SIGN_STORIES: readonly Story<StateSignProps>[] = TONES.map((tone) => ({
    name: tone,
    props: { tone },
}));
