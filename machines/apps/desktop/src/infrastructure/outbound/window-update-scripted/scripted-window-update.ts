import type {
    HeardProgress,
    WindowUpdatePort,
} from '../../../application/ports/window-update.port';
import type {
    WindowChecked,
    WindowInstalled,
    WindowProgress,
    WindowRefused,
} from '../../../domain/window-update';

export const WINDOW_SCRIPTS = [
    'current',
    'install-failing',
    'install-holding',
    'out',
    'refused',
] as const;

export type WindowScriptName = (typeof WINDOW_SCRIPTS)[number];

const MB = 1_000_000;
const TOTAL = 18 * MB;
const CURRENT = '0.21.0';
const OUT: WindowChecked = {
    current: CURRENT,
    date: '2026-11-19T09:00:00Z',
    kind: 'out',
    version: '0.22.0',
};

type Install = { steps: readonly WindowProgress[]; end?: WindowInstalled | WindowRefused };

type WindowScript = { check: WindowChecked | WindowRefused; installs: readonly Install[] };

const chunk = (downloaded: number): WindowProgress => ({
    downloaded,
    kind: 'downloaded',
    total: TOTAL,
});

const INSTALLS: readonly Install[] = [
    {
        end: { kind: 'installed' },
        steps: [chunk(6 * MB), chunk(12 * MB), chunk(TOTAL), { kind: 'installed' }],
    },
];

const SCRIPTS: Readonly<Record<WindowScriptName, WindowScript>> = {
    current: { check: { current: CURRENT, kind: 'current' }, installs: INSTALLS },
    'install-failing': {
        check: OUT,
        installs: [
            {
                end: {
                    code: 'window.update-install',
                    kind: 'refused',
                    message:
                        'failed to move the new app into place: Permission denied (os error 13)',
                },
                steps: [chunk(6 * MB), chunk(TOTAL)],
            },
            ...INSTALLS,
        ],
    },
    'install-holding': { check: OUT, installs: [{ steps: [chunk(7_400_000)] }] },
    out: { check: OUT, installs: INSTALLS },
    refused: {
        check: {
            code: 'window.update-unreachable',
            kind: 'refused',
            message:
                'error sending request for url (https://github.com/jterrazz/spwn/releases/latest/download/latest.json)',
        },
        installs: INSTALLS,
    },
};

function isWindowScript(name: string): name is WindowScriptName {
    return WINDOW_SCRIPTS.some((known) => known === name);
}

export function windowScriptOf(name: string): undefined | WindowScript {
    return isWindowScript(name) ? SCRIPTS[name] : undefined;
}

async function played(
    run: Install,
    heard: HeardProgress,
    gap: number,
): Promise<WindowInstalled | WindowRefused> {
    return await new Promise((resolve) => {
        for (const [index, step] of run.steps.entries()) {
            setTimeout(() => {
                heard(step);
            }, index * gap);
        }
        const { end } = run;
        if (end !== undefined) {
            setTimeout(() => {
                resolve(end);
            }, run.steps.length * gap);
        }
    });
}

// value: the proof build's shell: fixed versions, no network, and each install the next of the script's, its last again.
export function scriptedWindowUpdate(script: WindowScript, gap = 0): WindowUpdatePort {
    let installs = 0;
    return {
        check: async () => await Promise.resolve(script.check),
        install: async (heard) => {
            const run = script.installs[Math.min(installs, script.installs.length - 1)];
            installs += 1;
            return run === undefined
                ? await Promise.resolve({ kind: 'installed' })
                : await played(run, heard, gap);
        },
    };
}
