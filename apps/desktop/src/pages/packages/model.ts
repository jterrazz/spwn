import type {
    AppsReport,
    CatalogApp,
    MachinefileChapter,
    MachinefileChapterDrift,
    MachinefileDiffering,
    MachinefileDrift,
    MachinefilePlan,
    PageNoun,
} from '@jterrazz/station-api';

import { IDENTITIES } from '../../presentation/ui/index';
import type { Identity } from '../../presentation/ui/index';
import { countOf } from '../../presentation/words/engine-words';
import type { PlanLine } from '../../presentation/words/provisional';
import { shelfWord } from '../../presentation/words/shown';

export const SHELVES = ['Apps', 'Skills', 'Settings'] as const;

export type Shelf = (typeof SHELVES)[number];

export type Kinds = ReadonlyMap<string, CatalogApp['AppKind']>;

export type Layer = { name: string; identity: Identity; lines: readonly PlanLine[] };

const NO_LAYER: Identity = 'system';

export function byAction(lines: readonly PlanLine[]): readonly [PlanLine['Action'], PlanLine[]][] {
    const groups = new Map<PlanLine['Action'], PlanLine[]>();
    for (const line of lines) {
        groups.set(line.Action, [...(groups.get(line.Action) ?? []), line]);
    }
    return [...groups.entries()];
}

export function kindsOf(reports: readonly AppsReport[]): Kinds {
    return new Map(reports.map((report) => [report.App.Name, report.App.AppKind]));
}

export function shelfOf(line: PlanLine, kinds: Kinds): string {
    return kinds.get(line.Step.Package) === 'skill' ? 'Skills' : shelfWord(line.Step.Chapter);
}

export function layersOf(plan: MachinefilePlan): readonly Layer[] {
    const groups = new Map<string, PlanLine[]>();
    for (const line of plan.Lines ?? []) {
        const name = line.Step.File.Name;
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

export function onShelf(layers: readonly Layer[], shelf: Shelf, kinds: Kinds): readonly Layer[] {
    return layers
        .map((layer) => ({
            ...layer,
            lines: layer.lines.filter((line) => shelfOf(line, kinds) === shelf),
        }))
        .filter((layer) => layer.lines.length > 0);
}

export function countOn(layers: readonly Layer[], shelf: Shelf, kinds: Kinds): number {
    return onShelf(layers, shelf, kinds).reduce((sum, layer) => sum + layer.lines.length, 0);
}

const COUNTED_AS: Readonly<Record<Shelf, PageNoun>> = {
    Apps: 'app',
    Settings: 'setting',
    Skills: 'skill',
};

export function countWord(count: number, shelf: Shelf): string {
    return countOf(COUNTED_AS[shelf], count);
}

const PACKAGE_CHAPTERS: readonly MachinefileChapter[] = ['apps', 'settings'];

export function packageChapters(drift: MachinefileDrift): readonly MachinefileChapterDrift[] {
    const chapters: readonly MachinefileChapterDrift[] = drift.Chapters ?? [];
    return chapters.filter((chapter) => PACKAGE_CHAPTERS.some((one) => one === chapter.Chapter));
}

// value: the Packages tab heads with the apps chapter's verdict; the whole file's is Its file's, on the Overview.
const HEAD_CHAPTER: MachinefileChapter = 'apps';

export function headChapter(drift: MachinefileDrift): MachinefileChapterDrift | undefined {
    return (drift.Chapters ?? []).find((chapter) => chapter.Chapter === HEAD_CHAPTER);
}

function packageDifferences(drift: MachinefileDrift): readonly MachinefileDiffering[] {
    return packageChapters(drift).flatMap((chapter) => chapter.Differing ?? []);
}

export type Difference = { differing: MachinefileDiffering; line: PlanLine | undefined };

export function differencesOf(drift: MachinefileDrift): readonly Difference[] {
    const all = [...(drift.Plan.Lines ?? []), ...(drift.Plan.Extra ?? [])];
    return packageDifferences(drift).map((differing) => ({
        differing,
        line:
            differing.Block === ''
                ? all.find((line) => line.Step.Package === differing.Package)
                : undefined,
    }));
}

export function differingLinesOf(drift: MachinefileDrift): ReadonlySet<PlanLine> {
    return new Set(differencesOf(drift).flatMap(({ line }) => (line === undefined ? [] : [line])));
}
