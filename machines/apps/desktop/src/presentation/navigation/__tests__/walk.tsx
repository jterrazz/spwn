import { component } from '@jterrazz/test';

import { settled, stagedOver } from '../../../composition/__tests__/page-over';
import { isRecord } from '../../../domain/request-key';
import { recordingFor } from '../../../infrastructure/outbound/engine-replay/recordings';
import type { Recording } from '../../../infrastructure/outbound/engine-replay/replay';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { carriesCommand } from '../../ui/command';
import { machineRoute, PLAIN_PAGES, TABS, tabsOf } from '../route';
import type { Route } from '../route';
import { Window } from '../window';

export const WORLDS = Object.keys(
    import.meta.glob('../../../fixtures/worlds/*/engine.read.json'),
).map((path) => path.split('/').at(-2) ?? '');

const EVERY_WORLD = new Map(
    await Promise.all(WORLDS.map(async (world) => [world, await recordingFor(world)] as const)),
);

// value: what a world holds as data, not as a sentence of the window's: a VM's own log, and the disk row Machines names
// value: after its own package, `station`, until the product's rename reaches its files.
const DATA: readonly RegExp[] = [
    /^\s*- log "Log lines"/u,
    /\.local\/state\/station\/logs/u,
    /^\s*- button "Free station"/u,
];

function findingsIn(text: string): string[] {
    return text
        .split('\n')
        .filter((line) => carriesCommand(line) && !DATA.some((data) => data.test(line)))
        .map((line) => line.trim());
}

function namesOf(value: unknown): string[] {
    return isRecord(value) && typeof value.Name === 'string' ? [value.Name] : [];
}

function listedIn(recording: Recording): { macs: string[]; vms: string[] } {
    const listed = recording.exchanges.find((one) => one.request.watch === 'machines.list');
    const answer = listed?.answers.find((one) => isRecord(one) && isRecord(one.out));
    const all = isRecord(answer) && isRecord(answer.out) ? answer.out.All : undefined;
    const macs: unknown[] = isRecord(all) && Array.isArray(all.Macs) ? all.Macs : [];
    return {
        macs: macs.flatMap(namesOf),
        vms: macs.flatMap((mac) =>
            isRecord(mac) && Array.isArray(mac.VMs) ? mac.VMs.flatMap(namesOf) : [],
        ),
    };
}

function routesOf(recording: Recording): Route[] {
    const { macs, vms } = listedIn(recording);
    return [
        ...PLAIN_PAGES.filter((page) => page !== 'gallery').map((page): Route => ({ page })),
        ...macs.flatMap((mac) => tabsOf(undefined).map((tab) => machineRoute(mac, tab))),
        ...vms.flatMap((vm) => TABS.map((tab) => machineRoute(vm, tab))),
    ];
}

function nameOf(route: Route): string {
    return route.page === 'machine' ? `${route.machine} ${route.tab}` : route.page;
}

// value: every line of every page and tab of one world that carries a command, the old name or a config path.
export async function commandsDrawnOver(world: string): Promise<string[]> {
    const recording = EVERY_WORLD.get(world);
    if (recording === undefined) {
        throw new Error(`${world} is not loaded`);
    }
    const found: string[] = [];
    for (const route of routesOf(recording)) {
        const { dependencies, stage } = stagedOver(recording);
        const ui = (
            <EngineProvider dependencies={dependencies}>
                <Window initial={route} pages={PAGES} />
            </EngineProvider>
        );
        const result = await component.render(ui, async () => {
            await settled(stage);
        });
        found.push(...findingsIn(result.tree.text).map((line) => `${nameOf(route)} | ${line}`));
    }
    return [...new Set(found)];
}
