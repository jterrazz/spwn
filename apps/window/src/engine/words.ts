import type {
    AccountsAccount,
    ActivityReport,
    ActivitySample,
    AppsReport,
    DiskOutcome,
    PortsPort,
    PortsSharedPort,
} from '@jterrazz/station-api';

import { SEPARATOR } from '../separator';
import type { AccountSide, DiskCall, MacState, NeedCall } from './provisional';

type Words<T extends PropertyKey> = Readonly<Record<T, string>>;

// outside: the codes of these tables are a plain number or string in packages/api, so no compile catches a new
// outside: one; the README lists them as owed a closed type, and a code they do not word is shown as it came.
type WordsOfAnOpenCode = Readonly<Partial<Record<number | string, string>>>;

export const NO_STATE_WORD = 'no state given';

export const ANY_VERSION_WORD = 'any version';

// value: what the window says of its own reading of a call or of a gesture's fate, never of an engine code: an
// value: answer not yet come, none at all, a refusal sent with no type, a slot no call fills, a gesture done or of
// value: unknown outcome.
export const OWN_STATE_WORDS = {
    connecting: 'Connecting',
    done: 'done',
    'engine-absent': 'No answer',
    'no-answer': 'no answer',
    'no-finding': 'No finding',
    'not-given': 'Not given yet',
    'not-measured': 'Not measured',
    'not-wired': 'Not wired',
    reading: 'Reading',
    unknown: 'unknown',
    untyped: 'error',
    'window-error': 'Error',
} as const;

export type OwnState = keyof typeof OWN_STATE_WORDS;

export const NEWEST_WORD = 'Newest';

export const DISK_PENDING_WORD = 'reading';

export const THIS_MAC_WORD = 'This Mac';

export const NO_PURPOSE_WORD = 'No purpose given';

export const BATTERY_WORDS = { charging: 'charging', onBattery: 'on battery' } as const;

export const SILENT_METER_WORDS = {
    line: `The meter did not answer in time${SEPARATOR}no value is known`,
    title: 'No value is known.',
} as const;

export const MAC_STATE_WORDS: Words<MacState> = {
    asleep: 'asleep',
    reachable: 'online',
    unreachable: 'not seen',
};

export const MACHINE_KIND_WORDS: WordsOfAnOpenCode = {
    computer: 'Mac',
    server: 'server',
    workspace: 'workspace',
};

export const AWAKE_WORDS = {
    neverSleeps: 'never sleeps',
    off: 'Off',
    on: 'On',
} as const;

export const POWER_SOURCE_WORDS: WordsOfAnOpenCode = { ac: 'on power', battery: 'on battery' };

// outside: a need names its fix by its call and no verb; until OverviewNeed carries the button's word, this is
// outside: the word of each call the window makes from a need, the README listing it as owed.
export const NEED_CALL_VERBS: Words<NeedCall> = {
    'apps.update': 'Update',
    'background.remove': 'Remove',
    'disk.move': 'Move',
    'engine.start': 'Start the engine',
};

// outside: a need whose call the window does not make yet is drawn by this word on a disabled button, its call
// outside: never named to the owner.
export const NEED_UNWIRED_WORD = 'Fix';

export const SSH_WORDS = { answers: 'answers', silent: 'does not answer' } as const;

export const APP_STATE_WORDS: Words<AppsReport['State']> = {
    '': NO_STATE_WORD,
    absent: 'not installed',
    broken: 'failed',
    degraded: 'degraded',
    ok: 'installed',
    pending: 'pending',
    unknown: 'not known',
};

export const APP_WHY_WORDS: Words<AppsReport['Why']> = {
    '': '',
    failing: 'failing',
    interrupted: 'interrupted',
    late: 'late',
    missing: 'missing',
    outdated: 'update waiting',
    partial: 'partial',
    unmanaged: 'not managed',
};

// outside: apps.list marks no entry a skill (provisional.ts, isSkill), so no kind of the engine's names one.
export const SKILL_KIND_NAME = 'agent skill';

export const REVERSIBLE_WORD = 'Can be undone';

export function newestOutWord(newest: string): string {
    return `${newest} is out`;
}

export const DISK_GESTURE_VERBS: Words<DiskCall> = {
    'disk.erase': 'Erase',
    'disk.free': 'Free',
    'disk.move': 'Move',
};

export const DISK_GESTURE_LABELS: Words<DiskCall> = {
    'disk.erase': 'Erase…',
    'disk.free': 'Free',
    'disk.move': 'Move',
};

export const DISK_RESULT_WORDS: Words<DiskOutcome['Result']> = {
    '': NO_STATE_WORD,
    absent: 'not installed',
    broken: 'failed',
    degraded: 'partly done',
    ok: 'done',
    pending: 'pending',
    unknown: 'not known',
};

export const ACTIVITY_SOURCE_WORDS: Words<ActivityReport['source']> = {
    '': 'no history of it kept by its engine',
    direct: 'two samples read now, 1 s apart',
    engine: '',
    'mac-process': 'its process on its Mac: its guest does not answer',
};

export const PRESSURE_WORDS: Words<'' | NonNullable<ActivitySample['pressure']>> = {
    '': 'pressure not given yet',
    critical: 'pressure critical',
    normal: 'pressure normal',
    warn: 'pressure warn',
};

export const THERMAL_WORDS: WordsOfAnOpenCode = {
    '-1': 'not read',
    '0': 'Nominal',
    '1': 'Fair',
    '2': 'Serious',
    '3': 'Critical',
    '4': 'Critical',
};

export const PORT_ANSWER_WORDS: Words<PortsPort['Answer']> = {
    '': 'answers',
    starting: 'starting',
    unreachable: 'unreachable',
};

export const PORT_REACH_WORDS: WordsOfAnOpenCode = {
    0: 'Local',
    1: 'Private',
    2: 'Outside Tailscale',
};

export const SHARE_STATE_WORDS: Words<PortsSharedPort['state']> = {
    '': NO_STATE_WORD,
    closing: 'closing',
    gone: 'gone',
    'not-allowed': 'not allowed',
    refused: 'refused',
    shared: 'shared',
    unknown: 'unknown',
    waiting: 'waiting',
};

export const ACCOUNT_STATE_WORDS: Words<AccountsAccount['state']> = {
    '': NO_STATE_WORD,
    absent: 'signed out',
    broken: 'broken',
    degraded: 'needs a look',
    ok: 'signed in',
    pending: 'reading',
    unknown: 'unknown',
};

export const ACCOUNT_REASON_WORDS: WordsOfAnOpenCode = {
    expiring: 'ends soon',
    failing: 'failing',
    late: 'late',
    missing: 'missing',
    refused: 'refused',
    unreachable: 'unreachable',
    unset: 'not set',
};

export const ACCOUNT_ORIGIN_WORDS: WordsOfAnOpenCode = {
    '?': 'origin unknown',
    own: 'kept here',
    'own+shared': 'kept here, and a copy',
    shared: 'a copy, shared',
};

// outside: machinefile.drift words a port's audience and no side left out, nor an account's (provisional.ts,
// outside: accountSideOf): these are the words of what it leaves unsaid, until each side carries its own.
export const DIFFERING_WORDS: Words<AccountSide> = {
    'not-in-the-file': 'Not in the file',
    'not-shared': 'Not shared',
    shared: 'Shared',
};
