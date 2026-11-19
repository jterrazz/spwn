import type { FleetMac, FleetMachine, FleetState } from './fleet-register';
import type { Identity } from './identity';
import { LONG_NAME } from './story';

// value: the fleet the board's stories draw: the landing's three Macs and their machines, measured in GB and
// value: placed as shares of each Mac, macOS first.

const RUNNING: FleetState = { tone: 'ok', word: 'running' };
const HEALTHY: FleetState = { tone: 'ok', word: 'healthy' };
const CREATING: FleetState = { tone: 'new', word: 'creating' };
const RESTARTING: FleetState = { tone: 'new', word: 'restarting' };
const STOPPED: FleetState = { tone: 'plain', word: 'stopped' };
const SUSPENDED: FleetState = { tone: 'plain', word: 'suspended' };
const DOWN: FleetState = { tone: 'bad', word: 'down' };
const NOT_SEEN: FleetState = { tone: 'plain', word: 'not seen' };

export const ASLEEP: FleetState = { tone: 'plain', word: 'asleep' };
export const MAC_NOT_SEEN: FleetState = { tone: 'warn', word: 'not seen' };
export const ENGINE_STOPPED: FleetState = { tone: 'bad', word: 'engine stopped' };
export const ENGINE_NOT_INSTALLED: FleetState = { tone: 'warn', word: 'engine not installed' };

export const ALL_RUNNING: FleetState = { tone: 'ok', word: 'all running' };
export const NO_MACHINE_YET: FleetState = { tone: 'plain', word: 'no machine yet' };
export const ONE_DOWN: FleetState = { tone: 'bad', word: '1 machine down' };
export const ONE_MAC_NOT_SEEN: FleetState = { tone: 'warn', word: '1 Mac not seen' };

type Drawn = {
    name: string;
    identity: Identity;
    purpose: string;
    state?: FleetState;
    cpu?: string;
    gb?: number;
    used?: number;
    sentence?: string;
    line?: string;
};

type MacDrawn = {
    name: string;
    total: number;
    kind?: string | undefined;
    state: FleetState;
    cpu: string;
    thisMac?: boolean;
    measured?: boolean;
};

const SYSTEM_GB = 4;

function machinesOf(total: number, drawn: readonly Drawn[]): FleetMachine[] {
    let at = SYSTEM_GB;
    return drawn.map(({ gb = 0, used, state = RUNNING, cpu = '', sentence, ...rest }) => {
        const share =
            gb > 0
                ? {
                      at: at / total,
                      used: used === undefined ? undefined : used / gb,
                      width: gb / total,
                  }
                : undefined;
        at += gb;
        return {
            ...rest,
            cpu,
            memory: gb > 0 ? `${String(used ?? 0)} / ${String(gb)}` : '',
            sentence: sentence ?? 'Running. Nothing to do.',
            share,
            state,
        };
    });
}

export function macOf(mac: MacDrawn, drawn: readonly Drawn[] = []): FleetMac {
    const held = drawn.reduce((sum, one) => sum + (one.gb ?? 0), 0);
    const measured = mac.measured ?? true;
    return {
        cpu: mac.cpu,
        free: measured ? `${String(mac.total - SYSTEM_GB - held)} free` : '',
        kind: mac.kind,
        machines: machinesOf(mac.total, drawn),
        name: mac.name,
        state: mac.state,
        system: measured ? { at: 0, used: 1, width: SYSTEM_GB / mac.total } : undefined,
        thisMac: mac.thisMac ?? false,
    };
}

export const AGENT: Drawn = {
    cpu: '1 %',
    gb: 8,
    identity: 'blue',
    name: 'agent-7f',
    purpose: 'Write the changelog of v4',
    used: 5.3,
};
export const LINUX_1: Drawn = {
    cpu: '9 %',
    gb: 8,
    identity: 'teal',
    line: 'Apply it from its Packages tab.',
    name: 'linux-1',
    purpose: 'Lot L2.5, settings from a machine file',
    sentence: 'Running, with 2 differences from its file.',
    used: 5.3,
};
export const LINUX_2: Drawn = {
    gb: 8,
    identity: 'indigo',
    name: 'linux-2',
    purpose: 'Lot L3, the Machines page',
    sentence: 'Step 2 of 4, installing its apps.',
    state: CREATING,
    used: 0,
};
export const API: Drawn = {
    identity: 'purple',
    name: 'api',
    purpose: "The API's staging copy",
    sentence: 'Stopped. It reserves no memory.',
    state: STOPPED,
};
export const SLEEPER: Drawn = {
    identity: 'sky',
    name: 'notes',
    purpose: 'Draft the release notes',
    sentence: 'Suspended.',
    state: SUSPENDED,
};
export const RESTARTED: Drawn = {
    gb: 8,
    identity: 'teal',
    name: 'gate-1',
    purpose: 'Run the checks of every lot',
    sentence: 'Restarting.',
    state: RESTARTING,
    used: 1.2,
};
const INFRA: Drawn = {
    cpu: '10 %',
    gb: 24,
    identity: 'pink',
    name: 'infra',
    purpose: "The cluster's control plane",
    sentence: 'Healthy.',
    state: HEALTHY,
    used: 16,
};
export const DOWN_INFRA: Drawn = {
    gb: 24,
    identity: 'pink',
    line: 'Restart it, or read its log in Activity.',
    name: 'infra',
    purpose: "The cluster's control plane",
    sentence: 'Down since 09:12: its health check failed three times in a row.',
    state: DOWN,
};
export const LINUX_3: Drawn = {
    cpu: '17 %',
    gb: 8,
    identity: 'sky',
    name: 'linux-3',
    purpose: 'Rebuild the docs site',
    used: 5.3,
};
export const REVIEW_BOT: Drawn = {
    cpu: '0.9 %',
    gb: 4,
    identity: 'lilac',
    name: 'review-bot',
    purpose: 'Review every pull request of studio-station',
    used: 2.7,
};
export const MINI_MACHINES: readonly Drawn[] = [INFRA, LINUX_3, REVIEW_BOT];

export const PRO: MacDrawn = {
    cpu: '22 %',
    kind: '4 machines',
    name: 'macbook-pro',
    state: ALL_RUNNING,
    thisMac: true,
    total: 48,
};
export const MINI: MacDrawn = {
    cpu: '36 %',
    kind: '3 machines',
    name: 'mac-mini',
    state: ALL_RUNNING,
    total: 64,
};
export const AIR: MacDrawn = {
    cpu: '3 %',
    name: 'macbook-air',
    state: NO_MACHINE_YET,
    total: 24,
};

export const FLEET: readonly FleetMac[] = [
    macOf(PRO, [AGENT, LINUX_1, LINUX_2, API]),
    macOf(MINI, MINI_MACHINES),
    macOf(AIR),
];

const NOT_SEEN_SINCE = 'Not seen since yesterday, 18:40.';

export const STUDIO_MAC = macOf(
    {
        cpu: '',
        kind: '2 machines',
        measured: false,
        name: 'studio-mac',
        state: MAC_NOT_SEEN,
        total: 32,
    },
    [
        {
            identity: 'blue',
            name: 'render-1',
            purpose: 'Render the launch film',
            sentence: NOT_SEEN_SINCE,
            state: NOT_SEEN,
        },
        {
            identity: 'purple',
            name: 'render-2',
            purpose: 'Render its stills',
            sentence: NOT_SEEN_SINCE,
            state: NOT_SEEN,
        },
    ],
);

const DENSE_IDENTITIES: readonly Identity[] = [
    'blue',
    'purple',
    'teal',
    'indigo',
    'pink',
    'sky',
    'lilac',
];

export function denseMac(name: string, at: number): FleetMac {
    const drawn = Array.from({ length: 8 }, (_, index) => ({
        cpu: `${String(index + 2)} %`,
        gb: 12,
        identity: DENSE_IDENTITIES[(at * 8 + index) % DENSE_IDENTITIES.length] ?? 'blue',
        name: at === 1 && index === 3 ? LONG_NAME : `${name}-vm-${String(index + 1)}`,
        purpose: `Lot ${String(index + 1)} of the nightly run, its checks and its build`,
        used: 7,
    }));
    const mac = {
        cpu: '41 %',
        kind: '8 machines',
        name,
        state: ALL_RUNNING,
        thisMac: at === 0,
        total: 128,
    };
    return macOf(mac, drawn);
}
