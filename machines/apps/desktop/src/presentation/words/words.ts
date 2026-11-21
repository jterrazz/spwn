import type {
    ActivitySample,
    AppsReport,
    DiskOutcome,
    PortsPort,
    PortsSharedPort,
} from '@jterrazz/station-api';

import type { CoreKind } from '../../application/use-cases/core-flow';
import type { UpdateKind } from '../../application/use-cases/update-flow';
import type { WindowRelease } from '../../application/use-cases/window-update-flow';
import { bytes } from '../../domain/format';
import type { WindowUpdateCode } from '../../domain/window-update';
import type { DiskCall } from './provisional';

type Words<T extends PropertyKey> = Readonly<Record<T, string>>;

// outside: the codes of these tables are a plain number or string in packages/api, so no compile catches a new
// outside: one; the README lists them as owed a closed type, and a code they do not word is shown as it came.
type WordsOfAnOpenCode = Readonly<Partial<Record<number | string, string>>>;

export const NO_STATE_WORD = 'no state given';

// value: what the window says of its own reading of a call or of a gesture's fate, never of an engine code: an
// value: answer not yet come, none at all, a refusal sent with no type, a slot no call fills, a gesture done or of
// value: unknown outcome.
export const OWN_STATE_WORDS = {
    connecting: 'connecting',
    done: 'done',
    'engine-absent': 'no answer',
    'no-answer': 'no answer',
    'not-given': 'not given yet',
    'not-measured': 'not measured',
    'not-started': 'not started',
    'not-wired': 'not wired',
    reading: 'reading',
    unknown: 'unknown',
    untyped: 'error',
    'window-error': 'error',
    'window-out': 'release out',
} as const;

export type OwnState = keyof typeof OWN_STATE_WORDS;

export const CORE_WORDS: Words<CoreKind> = {
    failed: 'failed',
    installed: 'installed',
    installing: 'installing',
    looking: 'looking',
    missing: 'not installed',
    started: 'starting',
    starting: 'starting',
    waiting: 'waiting',
};

export const CORE_MACHINES_WORDS: Words<CoreKind> = {
    failed: 'None yet',
    installed: 'Start the engine to see them',
    installing: 'None yet',
    looking: 'None yet',
    missing: 'None yet',
    started: 'Start the engine to see them',
    starting: 'Start the engine to see them',
    waiting: 'None yet',
};

export const CORE_FOOT_WORDS: Words<CoreKind> = {
    failed: 'No engine on this Mac',
    installed: 'Engine not started',
    installing: 'Installing Spwn',
    looking: 'No engine on this Mac',
    missing: 'No engine on this Mac',
    started: 'Engine starting',
    starting: 'Engine starting',
    waiting: 'No engine on this Mac',
};

export const UPDATE_WORDS: Words<UpdateKind> = {
    failed: 'failed',
    idle: 'idle',
    restarting: 'restarting',
    updated: 'updated',
    updating: 'updating',
};

// value: the window's own words around the engine's release reading; the head and a refusal are the engine's.
export const RELEASE_WORDS = {
    available: 'available',
    checked: 'Checked',
    checkNow: 'Check now',
    update: 'Update',
    whatChanged: 'What changed',
} as const;

// value: the window's own words of its own release, said by the window about itself; no engine sends them.
export const WINDOW_RELEASE_WORDS = {
    checking: 'Checking for a release',
    checkNow: 'Check now',
    checkTitle: 'Read the latest release of this window now',
    install: 'Install and relaunch',
    latest: 'Latest release',
    relaunching: 'Relaunching Spwn',
    title: 'Window',
    tryAgain: 'Try again',
    update: 'Update',
} as const;

export const WINDOW_REFUSAL_WORDS: Words<WindowUpdateCode> = {
    'window.update-busy': 'The window is installing its update already.',
    'window.update-gone': 'The window holds no update to install: check again.',
    'window.update-install': 'The update did not install. Spwn is unchanged.',
    'window.update-unpublished': 'The latest release carries no update for this window yet.',
    'window.update-unreachable': 'The window could not read its latest release.',
    'window.update-unsigned': "The update was refused: its signature is not this window's.",
};

const WINDOW_REFUSAL_HINTS: Readonly<Partial<Record<WindowUpdateCode, string>>> = {
    'window.update-install': 'If Spwn runs from Downloads, move it to Applications first.',
};

export function windowRefusalHint(code: WindowUpdateCode): string | undefined {
    return WINDOW_REFUSAL_HINTS[code];
}

export function windowCurrentWord(current: string): string {
    return `This window runs v${current}, the latest release.`;
}

export function windowOutWord({ current, version }: WindowRelease): string {
    return `Spwn v${version} is out; this window runs v${current}.`;
}

export function windowDownloadingWord(version: string): string {
    return `Downloading Spwn v${version}`;
}

export function windowDownloadedWord(downloaded: number, total: number | undefined): string {
    return total === undefined ? bytes(downloaded) : `${bytes(downloaded)} of ${bytes(total)}`;
}

export function windowAvailableWord(version: string): string {
    return `Spwn v${version} available`;
}

export const NEWEST_WORD = 'newest';

export const DISK_PENDING_WORD = 'reading';

export const THIS_MAC_WORD = 'This Mac';

export const THE_MAC_ITSELF_WORD = 'the Mac itself';

export const NO_PURPOSE_WORD = 'No purpose given';

export const SHOW_OLDER_WORD = 'Show older';

// value: the window's words around a create that failed: the step's sentence and the lines are the engine's.
export const FAILED_CREATE_WORDS = {
    lines: 'What its console last wrote',
    none: 'The engine kept no line of its console.',
} as const;

export const BATTERY_WORDS = { charging: 'charging', onBattery: 'on battery' } as const;

export const MACHINE_KIND_WORDS: WordsOfAnOpenCode = {
    server: 'server',
    workspace: 'workspace',
};

export const AWAKE_WORDS = {
    neverSleeps: 'never sleeps',
    off: 'off',
    on: 'on',
} as const;

export const AWAKE_GESTURE_WORDS = {
    close: 'Close',
    howLong: 'How long',
    open: 'Awake…',
    stop: 'Stop keeping awake',
} as const;

export const SHARE_REACH_GESTURE_WORDS = { who: 'Who reaches' } as const;

export const POWER_SOURCE_WORDS: WordsOfAnOpenCode = { ac: 'on power', battery: 'on battery' };

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

export const REVERSIBLE_WORD = 'can be undone';

export function newestOutWord(newest: string): string {
    return `${newest} is out`;
}

export const DISK_GESTURE_VERBS: Words<DiskCall> = {
    'disk.erase': 'Erase',
    'disk.free': 'Free',
    'disk.free-unused': 'Free',
    'disk.move': 'Move',
};

export const DISK_GESTURE_LABELS: Words<DiskCall> = {
    'disk.erase': 'Erase…',
    'disk.free': 'Free',
    'disk.free-unused': 'Free',
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

// outside: machinefile.drift words a port's audience and an account's side, and leaves the side of what it does not
// outside: hold empty: this is the word of that nothing.
export const NOT_SHARED_WORD = 'not shared';
