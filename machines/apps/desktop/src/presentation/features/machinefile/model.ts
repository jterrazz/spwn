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

import { IDENTITIES } from '../../ui/index';
import type { Identity } from '../../ui/index';
import { countOf } from '../../words/engine-words';
import type { PlanLine } from '../../words/provisional';
import { shelfWord } from '../../words/shown';

export const SHELVES = ['Apps', 'Skills', 'Settings'] as const;

export type Shelf = (typeof SHELVES)[number];

export type Kinds = ReadonlyMap<string, CatalogApp['AppKind']>;

export type PlanFile = { name: string; identity: Identity; lines: readonly PlanLine[] };

const NO_FILE: Identity = 'system';

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

export function planFilesOf(plan: MachinefilePlan): readonly PlanFile[] {
    const groups = new Map<string, PlanLine[]>();
    for (const line of plan.Lines ?? []) {
        const name = line.Step.File.Name;
        groups.set(name, [...(groups.get(name) ?? []), line]);
    }
    const planFiles = [...groups].map(([name, lines], index) => ({
        identity: IDENTITIES[index % IDENTITIES.length] ?? NO_FILE,
        lines,
        name,
    }));
    const extra = plan.Extra ?? [];
    return extra.length === 0
        ? planFiles
        : [...planFiles, { identity: NO_FILE, lines: extra, name: '' }];
}

export function onShelf(
    planFiles: readonly PlanFile[],
    shelf: Shelf,
    kinds: Kinds,
): readonly PlanFile[] {
    return planFiles
        .map((planFile) => ({
            ...planFile,
            lines: planFile.lines.filter((line) => shelfOf(line, kinds) === shelf),
        }))
        .filter((planFile) => planFile.lines.length > 0);
}

export function countOn(planFiles: readonly PlanFile[], shelf: Shelf, kinds: Kinds): number {
    return onShelf(planFiles, shelf, kinds).reduce(
        (sum, planFile) => sum + planFile.lines.length,
        0,
    );
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
