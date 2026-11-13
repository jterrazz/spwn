import type { AppsReport, CatalogApp, MachinefilePlan } from '@jterrazz/station-api';

import { IDENTITIES } from '../../design/index';
import type { Identity } from '../../design/index';
import { isSkill } from '../../engine/provisional';
import type { PlanLine } from '../../engine/provisional';
import { CATALOG_SHELF_WORDS } from '../../engine/words';

export const SHELVES = ['Apps', 'Skills', 'Settings'] as const;

export type Shelf = (typeof SHELVES)[number];

export type Categories = ReadonlyMap<string, CatalogApp['Category']>;

export type Layer = { name: string; identity: Identity; lines: readonly PlanLine[] };

const NO_LAYER: Identity = 'system';

export function byAction(lines: readonly PlanLine[]): readonly [PlanLine['Action'], PlanLine[]][] {
    const groups = new Map<PlanLine['Action'], PlanLine[]>();
    for (const line of lines) {
        groups.set(line.Action, [...(groups.get(line.Action) ?? []), line]);
    }
    return [...groups.entries()];
}

export function categoriesOf(reports: readonly AppsReport[]): Categories {
    return new Map(reports.map((report) => [report.App.Name, report.App.Category]));
}

export function shelfOf(line: PlanLine, categories: Categories): string {
    const category = categories.get(line.Step.Package);
    return category !== undefined && isSkill(category)
        ? 'Skills'
        : CATALOG_SHELF_WORDS[line.Step.Kind];
}

export function layersOf(plan: MachinefilePlan): readonly Layer[] {
    const groups = new Map<string, PlanLine[]>();
    for (const line of plan.Lines ?? []) {
        const name = line.Step.Origin.Name;
        groups.set(name, [...(groups.get(name) ?? []), line]);
    }
    const layers = [...groups].map(([name, lines], index) => ({
        identity: IDENTITIES[index % IDENTITIES.length] ?? NO_LAYER,
        lines,
        name,
    }));
    const extra = plan.Extra ?? [];
    return extra.length === 0
        ? layers
        : [...layers, { identity: NO_LAYER, lines: extra, name: '' }];
}

export function onShelf(
    layers: readonly Layer[],
    shelf: Shelf,
    categories: Categories,
): readonly Layer[] {
    return layers
        .map((layer) => ({
            ...layer,
            lines: layer.lines.filter((line) => shelfOf(line, categories) === shelf),
        }))
        .filter((layer) => layer.lines.length > 0);
}

export function countOn(layers: readonly Layer[], shelf: Shelf, categories: Categories): number {
    return onShelf(layers, shelf, categories).reduce((sum, layer) => sum + layer.lines.length, 0);
}

const SINGULAR: Readonly<Record<Shelf, string>> = {
    Apps: 'app',
    Settings: 'setting',
    Skills: 'skill',
};

export function countWord(count: number, shelf: Shelf): string {
    return `${count} ${SINGULAR[shelf]}${count === 1 ? '' : 's'}`;
}
