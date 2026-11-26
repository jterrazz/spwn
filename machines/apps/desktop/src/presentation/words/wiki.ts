import type { WikiServedFinding, WikiServedGroup } from '@jterrazz/station-api';

import type { Tone } from '../../domain/tone';

// outside: a group's fold and a finding's level are closed in packages/api but are not in words.json yet; these two
// outside: tables word them until the engine does.
export const WIKI_FOLDED_WORDS: Readonly<Record<WikiServedGroup['Folded'], string>> = {
    '': '',
    author: 'folded by its author',
    budget: 'folded for this budget',
    seat: 'folded for',
};

export const WIKI_LEVEL_TONES: Readonly<Record<WikiServedFinding['Level'], Tone>> = {
    '': 'plain',
    ambient: 'plain',
    'needs-you': 'warn',
    remark: 'new',
};

export const WIKI_LEVEL_WORDS: Readonly<Record<WikiServedFinding['Level'], string>> = {
    '': 'finding',
    ambient: 'ambient',
    'needs-you': 'needs you',
    remark: 'remark',
};

export const WIKI_DECISION_WORDS = {
    amendedBy: 'amended by',
    decides: 'decides',
    record: 'record',
    unread: 'its record is not in this wiki',
} as const;

export const WIKI_WORDS = {
    all: 'All',
    budget: 'Reading time',
    floor: 'Floor',
    floorShort: 'E',
    ladder: 'Floors',
    folded: 'folded at this reading time',
    home: 'Wiki',
    homeLine: 'Every page of this Mac’s wiki, by the seat that reads it.',
    open: 'Open',
    fold: 'Fold',
    page: 'This page',
    seats: 'Seats',
    table: 'Table',
    unplaced: 'on neither axis',
} as const;
