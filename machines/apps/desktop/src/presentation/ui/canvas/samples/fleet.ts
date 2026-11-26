import type { NodeWords } from '../canvas-node';
import type { CanvasGraph, CanvasNode } from '../graph';

// value: a fleet as the canvas draws it, with no wiki in it: two Macs, their machines, each machine's apps and each
// value: app's settings; a setting that waits for the owner is marked, and each holder counts the marks under it.

type Setting = { name: string; value: string; waits?: boolean };
type App = { name: string; line: string; settings: readonly Setting[] };
type Machine = { name: string; kind: string; line: string; apps: readonly App[] };
type Mac = { name: string; band: number; machines: readonly Machine[] };

const EDITOR: App = {
    line: 'the editor and its extensions',
    name: 'Zed',
    settings: [
        { name: 'theme', value: 'One Dark' },
        { name: 'font-size', value: '14' },
        { name: 'vim-mode', value: 'not set yet', waits: true },
    ],
};

const SHELL: App = {
    line: 'the shell block and its prompt',
    name: 'zsh',
    settings: [
        { name: 'prompt', value: 'starship' },
        { name: 'history', value: '50,000 lines' },
    ],
};

const DOCKER: App = {
    line: 'the containers the agents run in',
    name: 'Docker',
    settings: [
        { name: 'memory', value: '8 GB' },
        { name: 'sign-in', value: 'expired', waits: true },
        { name: 'cpus', value: '4' },
    ],
};

const NODE: App = {
    line: 'the runtime of the window',
    name: 'Node',
    settings: [{ name: 'version', value: '24' }],
};

const FLEET: readonly Mac[] = [
    {
        band: 0,
        machines: [
            {
                apps: [EDITOR, SHELL, DOCKER],
                kind: 'Mac',
                line: 'macOS 26 · 64 GB',
                name: 'macbook-pro',
            },
            {
                apps: [SHELL, NODE],
                kind: 'VM',
                line: 'Ubuntu 24.04 · 16 GB',
                name: 'development-linux',
            },
            {
                apps: [EDITOR, SHELL],
                kind: 'VM',
                line: 'macOS 26 · 12 GB',
                name: 'development-macos',
            },
        ],
        name: 'MacBook Pro',
    },
    {
        band: 1,
        machines: [
            { apps: [SHELL, DOCKER], kind: 'Mac', line: 'macOS 26 · 32 GB', name: 'mac-mini' },
            { apps: [DOCKER, NODE], kind: 'VM', line: 'Ubuntu 24.04 · 8 GB', name: 'cluster' },
            { apps: [SHELL], kind: 'VM', line: 'Ubuntu 24.04 · 4 GB', name: 'gates' },
        ],
        name: 'Mac mini',
    },
];

type Built = { nodes: Map<string, CanvasNode>; words: Map<string, NodeWords> };

function plural(count: number, word: string): string {
    return `${String(count)} ${word}${count === 1 ? '' : 's'}`;
}

function added(
    built: Built,
    node: Omit<CanvasNode, 'marks' | 'needs'>,
    words: NodeWords,
    waits: boolean,
): number {
    const kids = node.kids.map((kid) => built.nodes.get(kid));
    const needs = waits ? 1 : kids.reduce((sum, kid) => sum + (kid?.needs ?? 0), 0);
    built.nodes.set(node.id, { ...node, marks: kids.map((kid) => (kid?.needs ?? 0) > 0), needs });
    built.words.set(node.id, words);
    return needs;
}

function appOf(built: Built, app: App, parent: string): string {
    const id = `${parent}/${app.name}`;
    const kids = app.settings.map((setting) => {
        const kid = `${id}/${setting.name}`;
        added(
            built,
            {
                band: 0,
                depth: 4,
                id: kid,
                kids: [],
                kind: 'setting',
                label: setting.name,
                parent: id,
            },
            { kicker: 'Setting', line: setting.value, name: setting.name },
            setting.waits === true,
        );
        return kid;
    });
    added(
        built,
        { band: 0, depth: 3, id, kids, kind: 'app', label: app.name, parent },
        {
            count: plural(kids.length, 'setting'),
            foot: plural(kids.length, 'setting'),
            kicker: 'App',
            line: app.line,
            name: app.name,
        },
        false,
    );
    return id;
}

function fleetOf(fleet: readonly Mac[]): {
    graph: CanvasGraph;
    words: ReadonlyMap<string, NodeWords>;
} {
    const built: Built = { nodes: new Map(), words: new Map() };
    const roots = fleet.map((mac) => {
        const id = `mac:${mac.name}`;
        const kids = mac.machines.map((machine) => {
            const at = `${id}/${machine.name}`;
            const apps = machine.apps.map((app) => appOf(built, app, at));
            added(
                built,
                {
                    band: 0,
                    depth: 2,
                    id: at,
                    kids: apps,
                    kind: 'machine',
                    label: machine.name,
                    parent: id,
                },
                {
                    foot: plural(apps.length, 'app'),
                    kicker: machine.kind,
                    lead: machine.line,
                    line: machine.line,
                    name: machine.name,
                },
                false,
            );
            return at;
        });
        added(
            built,
            { band: mac.band, depth: 1, id, kids, kind: 'mac', label: mac.name, parent: null },
            { count: plural(kids.length, 'machine'), name: mac.name },
            false,
        );
        return id;
    });
    return { graph: { nodes: built.nodes, roots }, words: built.words };
}

export const FLEET_SAMPLE = fleetOf(FLEET);

export const FLEET_BANDS: ReadonlyMap<number, string> = new Map([
    [0, 'Desk'],
    [1, 'Rack'],
]);
