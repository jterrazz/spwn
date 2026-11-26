import { isRecord } from './request-key';

// value: the engine's views the window mirrors (ADR-070), each named as the call whose answer it is: This Mac's by the
// value: call's name, another machine's by `<call>:<machine>`, asked for every machine as `<call>:*`.
export const VIEW_KEYS = [
    'machines.list',
    'overview.read',
    'overview.read+elsewhere',
    'overview.read:*',
    'ports.list+all-machines',
    'ports.list:*',
    'accounts.list',
    'accounts.list:*',
    'awake.read',
    'awake.read:*',
    'engine.guides',
] as const;

const PER_MACHINE = ['overview.read:', 'ports.list:', 'accounts.list:', 'awake.read:'] as const;

export type View =
    | `${(typeof PER_MACHINE)[number]}${string}`
    | Exclude<(typeof VIEW_KEYS)[number], `${string}:*`>;

type Aim = undefined | { name?: string | undefined; on?: string | undefined };

function fieldsOf(input: unknown): Record<string, unknown> | undefined {
    return isRecord(input) ? input : undefined;
}

function only(input: Record<string, unknown>, key: string): string | undefined {
    const keys = Object.keys(input);
    const value = input[key];
    return keys.length === 1 && keys[0] === key && typeof value === 'string' ? value : undefined;
}

function aMac(target: Aim): string | undefined {
    return target?.name !== undefined && target.on === undefined ? target.name : undefined;
}

function ownOrOf(call: 'accounts.list' | 'awake.read', machine: string | undefined): View {
    return machine === undefined ? call : `${call}:${machine}`;
}

function sole(fields: Record<string, unknown>, key: string, value: unknown): boolean {
    return Object.keys(fields).length === 1 && fields[key] === value;
}

function isPlain(fields: Record<string, unknown>): boolean {
    return Object.keys(fields).length === 0;
}

function overviewOf(fields: Record<string, unknown>): undefined | View {
    if (isPlain(fields)) {
        return 'overview.read';
    }
    return sole(fields, 'Elsewhere', true) ? 'overview.read+elsewhere' : undefined;
}

function portsOf(fields: Record<string, unknown>): undefined | View {
    const of = only(fields, 'Of');
    if (of !== undefined) {
        return `ports.list:${of}`;
    }
    return sole(fields, 'AllMachines', true) ? 'ports.list+all-machines' : undefined;
}

type Named = (fields: Record<string, unknown>, mac: string | undefined) => undefined | View;

const NAMED: Readonly<Record<string, Named>> = {
    'accounts.list': (fields, mac) => {
        const vm = only(fields, 'Machine');
        if (vm !== undefined && mac === undefined) {
            return `accounts.list:${vm}`;
        }
        return isPlain(fields) ? ownOrOf('accounts.list', mac) : undefined;
    },
    'awake.read': (fields, mac) => (isPlain(fields) ? ownOrOf('awake.read', mac) : undefined),
    'engine.guides': (fields, mac) =>
        isPlain(fields) && mac === undefined ? 'engine.guides' : undefined,
    'machines.list': (fields, mac) =>
        isPlain(fields) && mac === undefined ? 'machines.list' : undefined,
    'overview.read': (fields, mac) => {
        if (mac === undefined) {
            return overviewOf(fields);
        }
        return isPlain(fields) ? `overview.read:${mac}` : undefined;
    },
    'ports.list': (fields, mac) => (mac === undefined ? portsOf(fields) : undefined),
};

// invariant: a read is a view when it asks This Mac's engine for an answer the engine keeps: the fleet, This Mac's
// invariant: Overview with the other Macs or without, another machine's Overview, every Mac's ports or one machine's,
// invariant: a machine's sign-ins, a Mac's Awake and the acts the engine guides; any other input or target is still asked by its call.
export function viewNamed(name: string, input: unknown, target: Aim): undefined | View {
    const fields = fieldsOf(input);
    const mac = aMac(target);
    const named = NAMED[name];
    if (
        fields === undefined ||
        named === undefined ||
        (target !== undefined && mac === undefined)
    ) {
        return undefined;
    }
    return named(fields, mac);
}

// value: whether a view is one machine's, which the engine keeps only for the machines it reads.
export function isOfMachine(view: View): boolean {
    return PER_MACHINE.some((kind) => view.startsWith(kind));
}

export function isView(key: string): key is View {
    if (PER_MACHINE.some((kind) => key.startsWith(kind))) {
        return !key.endsWith(':*');
    }
    return (VIEW_KEYS as readonly string[]).includes(key);
}

// value: the call a view is the answer of.
export function callOfView(view: View): string {
    return view.split(/[+:]/u)[0] ?? view;
}

// value: whether a key asked of engine.state names the view: by its own name, or by `<call>:*`.
export function askedBy(key: string, view: View): boolean {
    if (key === view) {
        return true;
    }
    const kind = key.endsWith(':*') ? key.slice(0, -1) : undefined;
    return kind !== undefined && view.startsWith(kind);
}
