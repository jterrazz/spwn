import type {
    AccountsAccount,
    AppsReport,
    DiskOutcome,
    DiskRow,
    PortsPort,
    PortsSharedPort,
} from '@jterrazz/station-api';

import type { Tone } from '../tone';
import type { MacState, ShareEnd } from './provisional';
import type { OwnState } from './words';

type Tones<T extends PropertyKey> = Readonly<Record<T, Tone>>;

export const OWN_STATE_TONES: Tones<OwnState> = {
    connecting: 'new',
    done: 'ok',
    'engine-absent': 'bad',
    'no-answer': 'plain',
    'no-finding': 'plain',
    'not-given': 'plain',
    'not-measured': 'plain',
    'not-wired': 'plain',
    reading: 'new',
    unknown: 'warn',
    untyped: 'bad',
    'window-error': 'plain',
};

export const NEWEST_TONE: Tone = 'ok';

export const AWAKE_ON_TONE = 'ok' satisfies Tone;

export const REVERSIBLE_TONE: Tone = 'ok';

// outside: a colour of identity is design/'s type, which engine/ never imports: these are the colours the Disk
// outside: groups take, and design/'s type checks each where a page hands it over.
type GroupColour = 'pink' | 'purple' | 'slate' | 'system' | 'teal';

export const DISK_GROUP_COLOURS: Readonly<Record<DiskRow['Group'], GroupColour>> = {
    'can-rebuild': 'teal',
    misplaced: 'pink',
    'never-touched': 'slate',
    'your-data': 'purple',
};

export const PORT_ANSWER_TONES: Tones<PortsPort['Answer']> = {
    '': 'plain',
    starting: 'warn',
    unreachable: 'warn',
};

// outside: a port's Reach is a plain number in packages/api (words.ts, PORT_REACH_WORDS): a reach this table does
// outside: not know is drawn plain.
export const PORT_REACH_TONES: Readonly<Partial<Record<number, Tone>>> = {
    0: 'plain',
    1: 'plain',
    2: 'plain',
};

export const APP_WHY_TONES: Tones<AppsReport['Why']> = {
    '': 'plain',
    failing: 'warn',
    interrupted: 'warn',
    late: 'warn',
    missing: 'warn',
    outdated: 'warn',
    partial: 'warn',
    unmanaged: 'warn',
};

export const MAC_STATE_TONES: Tones<MacState> = {
    asleep: 'plain',
    reachable: 'ok',
    unreachable: 'plain',
};

export const APP_STATE_TONES: Tones<AppsReport['State']> = {
    '': 'plain',
    absent: 'plain',
    broken: 'bad',
    degraded: 'warn',
    ok: 'ok',
    pending: 'new',
    unknown: 'plain',
};

export const DISK_RESULT_TONES: Tones<DiskOutcome['Result']> = {
    '': 'plain',
    absent: 'plain',
    broken: 'bad',
    degraded: 'warn',
    ok: 'ok',
    pending: 'new',
    unknown: 'plain',
};

export const SHARE_STATE_TONES: Tones<PortsSharedPort['state']> = {
    '': 'plain',
    closing: 'plain',
    gone: 'plain',
    'not-allowed': 'bad',
    refused: 'bad',
    shared: 'ok',
    unknown: 'plain',
    waiting: 'new',
};

export const SHARE_END_TONES: Tones<ShareEnd['kind']> = {
    ended: 'plain',
    'ends-in': 'warn',
    unread: 'plain',
};

export const ACCOUNT_STATE_TONES: Tones<AccountsAccount['state']> = {
    '': 'plain',
    absent: 'plain',
    broken: 'bad',
    degraded: 'warn',
    ok: 'ok',
    pending: 'new',
    unknown: 'plain',
};
