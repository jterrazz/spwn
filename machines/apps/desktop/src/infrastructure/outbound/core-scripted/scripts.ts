import type { CoreLine } from '../../../application/ports/core.port';
import type { CoreScript, ScriptedRun } from './scripted-core';

export const CORE_SCRIPTS = [
    'failing',
    'holding',
    'installed',
    'missing',
    'terminal',
    'update',
    'update-failing',
    'update-holding',
] as const;

export type CoreScriptName = (typeof CORE_SCRIPTS)[number];

const out = (text: string): CoreLine => ({ stream: 'out', text });

const err = (text: string): CoreLine => ({ stream: 'err', text });

const DOWNLOADING = [
    out('Detecting the newest release...'),
    out('Version: v0.21.0'),
    out('Downloading spwn_darwin_arm64.tar.gz...'),
    out('Downloading station-darwin-arm64...'),
];

const INSTALLED: ScriptedRun = {
    code: 0,
    lines: [
        ...DOWNLOADING,
        out('  ✓ spwn v0.21.0 in /Users/you/.local/bin, also as spwn-agents'),
        out('  ✓ station v0.21.0 in /Users/you/.local/bin, also as spwn-machines'),
        out("  Added to /Users/you/.zshrc: alias agents='spwn-agents'"),
        out("  Added to /Users/you/.zshrc: alias machines='spwn-machines'"),
    ],
};

const STARTED: ScriptedRun = {
    code: 0,
    lines: [
        out('✓ engine · installed, running · v0.21.0'),
        out(
            "◐ local network · needs you · approve station on macbook-pro's screen, or in System Settings > Privacy & Security > Local Network",
        ),
    ],
};

const FAILED: ScriptedRun = {
    code: 1,
    lines: [
        ...DOWNLOADING,
        err('Error: station-darwin-arm64 does not match its checksum, nothing installed'),
    ],
};

const HELD: ScriptedRun = { code: 0, holds: true, lines: DOWNLOADING };

const UPDATING = [
    out('Detecting the newest release...'),
    out('Version: v0.22.0'),
    out('Downloading spwn_darwin_arm64.tar.gz...'),
    out('Downloading station-darwin-arm64...'),
];

const UPDATED: ScriptedRun = {
    code: 0,
    lines: [
        ...UPDATING,
        out('  ✓ spwn v0.22.0 in /Users/you/.local/bin, also as spwn-agents'),
        out('  ✓ station v0.22.0 in /Users/you/.local/bin, also as spwn-machines'),
    ],
};

const RESTARTED: ScriptedRun = {
    code: 0,
    lines: [out('✓ engine · installed, running · v0.22.0')],
};

const UPDATE_FAILED: ScriptedRun = {
    code: 1,
    lines: [
        ...UPDATING,
        err('Error: station-darwin-arm64 does not match its checksum, nothing installed'),
    ],
};

export const SCRIPTS: Readonly<Record<CoreScriptName, CoreScript>> = {
    failing: { install: FAILED, present: false, start: STARTED },
    holding: { install: HELD, present: false, start: STARTED },
    installed: { install: INSTALLED, present: true, start: STARTED },
    missing: { install: INSTALLED, present: false, start: STARTED },
    terminal: { appearsAt: 3, install: INSTALLED, present: false, start: STARTED },
    update: { answering: true, install: UPDATED, present: true, start: RESTARTED },
    'update-failing': { answering: true, install: UPDATE_FAILED, present: true, start: RESTARTED },
    'update-holding': {
        answering: true,
        install: { code: 0, holds: true, lines: UPDATING },
        present: true,
        start: RESTARTED,
    },
};

export function scriptOf(name: null | string): CoreScript | undefined {
    const found = CORE_SCRIPTS.find((one) => one === name);
    return found === undefined ? undefined : SCRIPTS[found];
}
