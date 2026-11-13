import type { CoreVerdictTone } from '@jterrazz/station-api';

export type Tone = CoreVerdictTone;

export const TONES: readonly Tone[] = ['ok', 'warn', 'bad', 'new', 'plain'];
