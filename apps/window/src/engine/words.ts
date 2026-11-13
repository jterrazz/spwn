import type {
    AccountsAccount,
    ActivityReport,
    ActivitySample,
    AppsReport,
    CatalogApp,
    CatalogStored,
    CoreCallWireError,
    CoreFindingFinding,
    DiskOutcome,
    DiskRow,
    MachinefileAsked,
    MachinefileLine,
    MachinefilePlan,
    MachinefileSeen,
    MachinefileStep,
    MachinesCheckStatus,
    OverviewNeed,
    PortsPort,
    PortsReservation,
    PortsSharedPort,
} from '@jterrazz/station-api';

import type { DiskCall, EngineState, MacState, NeedCall, VmState } from './provisional';

type Words<T extends PropertyKey> = Readonly<Record<T, string>>;

// outside: the codes of these tables are a plain number or string in packages/api, so no compile catches a new
// outside: one; the README lists them as owed a closed type, and a code they do not word is shown as it came.
type WordsOfAnOpenCode = Readonly<Partial<Record<number | string, string>>>;

export const ERROR_TYPE_WORDS: Words<'' | NonNullable<CoreCallWireError['type']>> = {
    '': 'error',
    declined: 'declined',
    'engine-stopped': 'engine stopped',
    failed: 'failed',
    'far-too-old': 'far too old',
    interrupted: 'interrupted',
    'needs-confirmation': 'needs confirmation',
    'needs-terminal': 'needs a terminal',
    'not-for-this-machine': 'not for this machine',
    'not-found': 'not found',
    refused: 'refused',
    unreachable: 'unreachable',
};

export const SEVERITY_WORDS: Words<CoreFindingFinding['Severity']> = {
    '': 'no severity given',
    broken: 'broken',
    'needs-a-look': 'needs a look',
};

export const SECTION_WORDS: Words<OverviewNeed['Section']> = {
    '': 'no section given',
    accounts: 'Accounts',
    activity: 'Activity',
    apps: 'Apps',
    awake: 'Awake',
    background: 'Background',
    disk: 'Disk',
    images: 'Images',
    overview: 'Overview',
    ports: 'Ports',
    settings: 'Settings',
    ssh: 'SSH',
};

export const NO_STATE_WORD = 'no state given';

export const ANY_VERSION_WORD = 'any version';

// value: what the window says of its own reading of a call or of a gesture's fate, never of an engine code: an
// value: answer not yet come, none at all, a slot no call fills, a gesture done or of unknown outcome.
export const OWN_STATE_WORDS = {
    connecting: 'Connecting',
    done: 'done',
    'engine-absent': 'No answer',
    'no-answer': 'no answer',
    'no-finding': 'No finding',
    'not-given': 'Not given yet',
    'not-measured': 'Not measured',
    'not-wired': 'Not wired',
    'read-once': 'Read once',
    reading: 'Reading',
    unknown: 'unknown',
    'window-error': 'Error',
} as const;

export type OwnState = keyof typeof OWN_STATE_WORDS;

export const NEWEST_WORD = 'Newest';

export const DISK_PENDING_WORD = 'reading';

export const THIS_MAC_WORD = 'This Mac';

export const BATTERY_WORDS = { charging: 'charging', onBattery: 'on battery' } as const;

export const SILENT_METER_WORDS = {
    line: 'The meter did not answer in time · no value is known',
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
    until: 'awake until',
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

export const VM_STATE_WORDS: Words<VmState> = {
    broken: 'broken',
    creating: 'creating',
    running: 'running',
    stale: 'stale',
    stopped: 'stopped',
    suspended: 'suspended',
    unknown: 'unknown',
};

export const ENGINE_STATE_WORDS: Words<EngineState> = {
    'not-installed': 'not installed',
    running: 'running',
    stopped: 'stopped',
};

export const VM_CHECK_WORDS: Words<'' | NonNullable<MachinesCheckStatus['state']>> = {
    '': NO_STATE_WORD,
    failing: 'failing',
    passing: 'passing',
    waiting: 'waiting',
};

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

// value: an app is the plain kind, worded by nothing: a row draws the kind of a manager or a setting alone.
export const CATALOG_KIND_WORDS: Words<NonNullable<CatalogApp['Kind']>> = {
    app: '',
    manager: 'manager',
    setting: 'setting',
};

export const CATALOG_KIND_NAMES: Words<NonNullable<CatalogApp['Kind']>> = {
    app: 'app',
    manager: 'manager',
    setting: 'setting',
};

export const SKILL_KIND_NAME = 'agent skill';

export const CATALOG_SHELF_WORDS: Words<MachinefileStep['Kind']> = {
    '': 'Apps',
    app: 'Apps',
    manager: 'Apps',
    setting: 'Settings',
};

export const CATALOG_LEVEL_WORDS: Words<CatalogStored['Level']> = {
    '': 'no level given',
    cache: 'cache',
    data: 'data',
    kept: 'kept',
    settings: 'settings',
};

export const CATALOG_LEVEL_LINES: Words<CatalogStored['Level']> = {
    '': '',
    cache: 'rebuilt when needed',
    data: 'yours, kept',
    kept: 'kept',
    settings: 'your settings, kept',
};

export const REVERSIBLE_WORD = 'Can be undone';

export function newestOutWord(newest: string): string {
    return `${newest} is out`;
}

export const CONDITION_WORDS: Words<MachinefileSeen['Condition']> = {
    '': 'no condition given',
    absent: 'not installed',
    changed: 'changed',
    extra: 'extra',
    'not-managed': 'not managed',
    outdated: 'outdated',
    present: 'present',
    unknown: 'not known',
};

export const ACTION_WORDS: Words<MachinefileLine['Action']> = {
    '': 'no action given',
    adopt: 'adopt it',
    install: 'install it',
    keep: 'keep it',
    leave: 'leave it',
    remove: 'remove it',
    set: 'set it',
};

export const PLAN_GROUP_WORDS: Words<MachinefileLine['Action']> = {
    '': 'No action given',
    adopt: 'To adopt',
    install: 'To install',
    keep: 'Unchanged',
    leave: 'Left as is',
    remove: 'To remove',
    set: 'To set',
};

export const NEED_WORDS: Words<MachinefileAsked['Need']> = {
    '': 'no need given',
    'app-store-login': 'an App Store sign-in',
    license: 'a licence accepted',
    sudo: 'your password',
    terminal: 'a terminal',
};

export const RESTART_WORDS: Words<NonNullable<MachinefilePlan['Restarts']>[number]> = {
    logout: 'a log out',
    reboot: 'a restart of the machine',
    'restart-dock': 'the Dock restarted',
    'restart-finder': 'the Finder restarted',
    'restart-menu-bar': 'the menu bar restarted',
};

export const DISK_GROUP_WORDS: Words<DiskRow['Group']> = {
    '': 'No group given',
    'can-rebuild': 'Can rebuild',
    misplaced: 'Misplaced',
    'never-touched': 'Never touched',
    'your-data': 'Your data',
};

export const DISK_KIND_WORDS: Words<DiskRow['Group']> = {
    '': 'not grouped',
    'can-rebuild': 'cache',
    misplaced: 'misplaced',
    'never-touched': 'never touched',
    'your-data': 'data',
};

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

export const SHARE_TO_WORDS: Words<NonNullable<PortsReservation['to']>> = {
    local: 'Local',
    private: 'Private',
    public: 'Public',
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
