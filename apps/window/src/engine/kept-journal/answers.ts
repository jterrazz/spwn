import type { MachinesMacView } from '@jterrazz/station-api';

import type { JournalAnswer, JournalGroup, JournalLine, JournalReadOutput } from './journal';

// outside: PROVISIONAL, what the kept journal would answer over the bench's worlds, written by hand until the engine
// outside: keeps one (journal.ts says how this folder goes). Every title, sum and sentence is written as the engine
// outside: would send it, none computed by the window; outside stories, This Mac is named by the listing's own name
// outside: for it, so the answer holds no Mac's name of its own.

function thisMacOf(macs: readonly MachinesMacView[]): string {
    return macs.find((mac) => mac.ThisMac)?.Name ?? '';
}

const LINE: JournalLine = {
    At: '',
    By: '',
    Detail: '',
    On: null,
    Outcome: 'done',
    Purpose: '',
    Retry: null,
    Subject: null,
    Text: '',
    Upgrade: null,
};

function line(at: string, text: string, fields: Partial<JournalLine>): JournalLine {
    return { ...LINE, At: `2026-11-${at}:00Z`, Text: text, ...fields };
}

const STORIES_TODAY: JournalGroup = {
    Lines: [
        line('14T14:07', 'gate-2 deleted', {
            By: 'you',
            Detail: '8 GB freed',
            On: ['mac-mini'],
        }),
        line('14T14:06', 'infra restarted', {
            By: 'you',
            On: ['infra'],
            Subject: { Kind: 'machine', Name: 'infra' },
        }),
    ],
    Night: false,
    Sum: '',
    Title: 'Today',
};

const STORIES_NIGHT: JournalGroup = {
    Lines: [
        line('14T03:15', 'ripgrep', {
            Detail: 'download timed out',
            On: ['linux-1'],
            Outcome: 'failed',
            Retry: {
                Call: 'apps.update',
                Input: { Names: ['ripgrep'] },
                Machine: 'linux-1',
                Verb: 'Retry',
            },
            Subject: { Kind: 'app', Name: 'ripgrep' },
        }),
        line('14T03:12', 'node', {
            On: ['macbook-pro', 'mac-mini', 'agent-7f', 'api', 'linux-1', 'infra'],
            Subject: { Kind: 'app', Name: 'node' },
            Upgrade: { From: '24.11.0', To: '24.11.1' },
        }),
        line('14T03:12', 'gh', {
            On: ['macbook-pro'],
            Subject: { Kind: 'app', Name: 'gh' },
            Upgrade: { From: '2.97.0', To: '2.98.0' },
        }),
        line('14T03:13', 'uv', {
            On: ['agent-7f', 'api', 'linux-1', 'infra', 'review-bot'],
            Subject: { Kind: 'app', Name: 'uv' },
            Upgrade: { From: '0.9.7', To: '0.9.9' },
        }),
        line('14T03:14', 'orbstack', {
            Detail: 'updates itself',
            On: ['macbook-pro'],
            Outcome: 'held',
            Subject: { Kind: 'app', Name: 'orbstack' },
        }),
    ],
    Night: true,
    Sum: '3 moved, 1 held, 1 failed',
    Title: 'Last night',
};

const STORIES_YESTERDAY: JournalGroup = {
    Lines: [
        line('13T18:02', 'linux-3 created', {
            By: 'agent-7f',
            On: ['linux-3'],
            Purpose: 'review the docs lot',
            Subject: { Kind: 'machine', Name: 'linux-3' },
        }),
        line('13T17:40', 'Port 3000 shared, private', {
            By: 'you',
            On: ['linux-1'],
            Subject: { Kind: 'machine', Name: 'linux-1' },
        }),
        line('13T09:15', 'infra restarted', {
            By: 'its Mac',
            Detail: 'after an update',
            On: ['infra'],
            Subject: { Kind: 'machine', Name: 'infra' },
        }),
    ],
    Night: false,
    Sum: '',
    Title: 'Yesterday',
};

function stories(): JournalReadOutput {
    return { Groups: [STORIES_TODAY, STORIES_NIGHT, STORIES_YESTERDAY] };
}

function failedNight(here: string): JournalReadOutput {
    return {
        Groups: [
            {
                Lines: [
                    line('14T03:15', 'mise', {
                        Detail: 'download timed out',
                        On: [here],
                        Outcome: 'failed',
                        Retry: {
                            Call: 'apps.update',
                            Input: { Managers: ['mise'] },
                            Machine: here,
                            Verb: 'Retry',
                        },
                        Subject: { Kind: 'app', Name: 'mise' },
                    }),
                    line('14T03:12', 'node', {
                        On: [here, 'api', 'linux-1', 'infra'],
                        Subject: { Kind: 'app', Name: 'node' },
                        Upgrade: { From: '24.11.0', To: '24.11.1' },
                    }),
                    line('14T03:13', 'gh', {
                        On: [here],
                        Subject: { Kind: 'app', Name: 'gh' },
                        Upgrade: { From: '2.97.0', To: '2.98.0' },
                    }),
                ],
                Night: true,
                Sum: '2 moved, 1 failed',
                Title: 'Last night',
            },
            {
                Lines: [
                    line('13T18:02', 'linux-1 created', {
                        By: 'agent-7f',
                        On: ['linux-1'],
                        Purpose: 'run the docs lot',
                        Subject: { Kind: 'machine', Name: 'linux-1' },
                    }),
                    line('13T11:30', 'infra restarted', {
                        By: 'you',
                        On: ['infra'],
                        Subject: { Kind: 'machine', Name: 'infra' },
                    }),
                ],
                Night: false,
                Sum: '',
                Title: 'Yesterday',
            },
        ],
    };
}

function calmNight(here: string): JournalReadOutput {
    return {
        Groups: [
            {
                Lines: [
                    line('14T03:12', 'node', {
                        On: [here, 'api', 'linux-1'],
                        Subject: { Kind: 'app', Name: 'node' },
                        Upgrade: { From: '24.11.0', To: '24.11.1' },
                    }),
                    line('14T03:13', 'jq', {
                        On: ['api', 'linux-1'],
                        Subject: { Kind: 'app', Name: 'jq' },
                        Upgrade: { From: '1.7', To: '1.7.1' },
                    }),
                ],
                Night: true,
                Sum: '2 moved',
                Title: 'Last night',
            },
            {
                Lines: [
                    line('13T16:20', 'infra stopped', {
                        By: 'you',
                        On: ['infra'],
                        Subject: { Kind: 'machine', Name: 'infra' },
                    }),
                ],
                Night: false,
                Sum: '',
                Title: 'Yesterday',
            },
        ],
    };
}

export const ANSWERS: readonly JournalAnswer[] = [
    { fleet: 'agent-7f,api,linux-1 | infra,linux-3,review-bot | ?', out: stories },
    { fleet: 'api,linux-1 | infra', out: (macs) => failedNight(thisMacOf(macs)) },
    { fleet: 'api,linux-1 | infra?', out: (macs) => calmNight(thisMacOf(macs)) },
    { fleet: '', out: () => ({ Groups: [] }) },
];
