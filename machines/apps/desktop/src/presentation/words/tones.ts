import type {
    AppsReport,
    DiskAmount,
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
    unknown: 'plain',
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

// outside: a grey of the neutral ramp is ui/'s type (ui/bar.tsx's Step), which words/ never imports: these are the
// outside: four the Disk bar's kinds take, named by the token that draws them, and the bar checks each it is handed.
export type DiskStep = 's3' | 't2' | 't3' | 't4';

export const UNKNOWN_DISK_STEP: DiskStep = 't3';

// invariant: a kind is told apart by its place and its word, never by a hue: the ten kinds that hold bytes cycle
// invariant: the ramp in this table's order, and free takes none, being the empty track.
export const DISK_KIND_STEPS: Readonly<Record<Exclude<DiskAmount['Kind'], 'free'>, DiskStep>> = {
    apps: 't2',
    caches: 't3',
    'image-cache': 't4',
    images: 's3',
    leftovers: 't2',
    logs: 't3',
    'machine-disks': 't4',
    'other-vms': 's3',
    system: 't2',
    'your-data': 't3',
};

export const PORT_ANSWER_TONES: Tones<PortsPort['Answer']> = {
    '': 'plain',
    starting: 'new',
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
