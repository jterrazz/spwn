import type {
    AppsReport,
    DiskAmount,
    DiskHolding,
    DiskOutcome,
    PortsPort,
    PortsSharedPort,
} from '@jterrazz/station-api';

import type { CoreKind } from '../../application/use-cases/core-flow';
import type { UpdateKind } from '../../application/use-cases/update-flow';
import type { Tone } from '../../domain/tone';
import type { OwnState } from './words';

type Tones<T extends PropertyKey> = Readonly<Record<T, Tone>>;

export const OWN_STATE_TONES: Tones<OwnState> = {
    connecting: 'new',
    done: 'ok',
    'engine-absent': 'bad',
    'no-answer': 'plain',
    'not-given': 'plain',
    'not-measured': 'plain',
    'not-started': 'plain',
    'not-wired': 'plain',
    reading: 'new',
    stale: 'plain',
    unknown: 'warn',
    untyped: 'bad',
    'window-error': 'plain',
    'window-out': 'new',
};

export const CORE_TONES: Tones<CoreKind> = {
    failed: 'bad',
    installed: 'ok',
    installing: 'new',
    looking: 'plain',
    missing: 'plain',
    started: 'new',
    starting: 'new',
    waiting: 'new',
};

export const UPDATE_TONES: Tones<UpdateKind> = {
    failed: 'bad',
    idle: 'plain',
    restarting: 'new',
    updated: 'ok',
    updating: 'new',
};

export const NEWEST_TONE: Tone = 'ok';

export const AWAKE_ON_TONE = 'ok' satisfies Tone;

export const REVERSIBLE_TONE: Tone = 'ok';

// outside: a colour of identity is design/'s type, which engine/ never imports: these are the colours the Disk
// outside: bar's kinds and holders take, and design/'s type checks each where a page hands it over.
export type ShareColour =
    | 'blue'
    | 'indigo'
    | 'pink'
    | 'purple'
    | 'sky'
    | 'slate'
    | 'system'
    | 'teal';

export const UNKNOWN_SHARE_COLOUR: ShareColour = 'system';

export const DISK_KIND_COLOURS: Readonly<Record<DiskAmount['Kind'], ShareColour>> = {
    apps: 'blue',
    caches: 'teal',
    free: 'system',
    machines: 'indigo',
    system: 'slate',
    'your-data': 'purple',
};

export const DISK_HOLDER_COLOURS: Readonly<Record<DiskHolding['Holder'], ShareColour>> = {
    free: 'system',
    images: 'purple',
    leftovers: 'pink',
    logs: 'sky',
    'machine-disks': 'blue',
    'other-vms': 'indigo',
    'this-mac': 'slate',
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
