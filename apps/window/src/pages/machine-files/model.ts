import type {
    AppsReport,
    MachinefileFileView,
    MachinefileHead,
    MachinefileShelf,
} from '@jterrazz/station-api';

import type { CodeLine } from '../../design/code-block';
import { IDENTITIES } from '../../design/index';
import type { Identity } from '../../design/index';
import { wordOf, wordOfSent } from '../../engine/engine-words';
import { SEPARATOR } from '../../separator';
import type { PlainPage, Route } from '../../shell/route';
import { SHELVES } from '../packages/model';
import type { Shelf } from '../packages/model';

export const CHAPTERS = [...SHELVES, 'Access'] as const;

export type Chapter = (typeof CHAPTERS)[number];

const CATALOG_PAGES: Readonly<Record<Shelf, PlainPage>> = {
    Apps: 'catalog-apps',
    Settings: 'catalog-settings',
    Skills: 'catalog-skills',
};

export function catalogRouteOf(shelf: Shelf, name: string): Route {
    return { item: name, page: CATALOG_PAGES[shelf] };
}

const SHELF_ORDER: readonly MachinefileShelf[] = ['yours', 'mixin', 'shipped'];

export type Shelved = { shelf: MachinefileShelf; files: readonly MachinefileFileView[] };

export function shelvedOf(files: readonly MachinefileFileView[]): readonly Shelved[] {
    return SHELF_ORDER.map((shelf) => ({
        files: files.filter((file) => file.Shelf === shelf),
        shelf,
    })).filter((shelved) => shelved.files.length > 0);
}

export function descriptionsOf(reports: readonly AppsReport[]): ReadonlyMap<string, string> {
    return new Map(reports.map((report) => [report.App.Name, report.App.Description]));
}

export function systemOf({ System }: MachinefileHead): string {
    const named = [
        System.OS === undefined ? '' : wordOfSent('CatalogOS', System.OS),
        System.Version ?? '',
    ]
        .filter((part) => part !== '')
        .join(' ');
    return [named, System.Arch === undefined ? '' : wordOfSent('CatalogArch', System.Arch)]
        .filter((part) => part !== '')
        .join(SEPARATOR);
}

export function makesOf({ Kind }: MachinefileHead): string {
    return wordOf('CatalogMachineRole', Kind).Word;
}

export function sizeOf({ VM }: MachinefileHead): string {
    if (VM === undefined) {
        return '';
    }
    return [VM.CPUs === 0 ? '' : `${String(VM.CPUs)} CPU`, VM.Memory, VM.Disk]
        .filter((part) => part !== '')
        .join(SEPARATOR);
}

export function lifetimeOf({ VM }: MachinefileHead): string {
    return VM === undefined ? '' : wordOfSent('CatalogLifetime', VM.Lifetime);
}

export function extendedBy(
    files: readonly MachinefileFileView[],
    name: string,
): readonly MachinefileFileView[] {
    return files.filter((file) => (file.Extends ?? []).includes(name));
}

export function identityAt(index: number): Identity {
    return IDENTITIES[index % IDENTITIES.length] ?? 'system';
}

export function linesOf(text: string): readonly CodeLine[] {
    const lines = text.split('\n');
    const kept = lines.at(-1) === '' ? lines.slice(0, -1) : lines;
    return kept.map((line, index) => ({ number: index + 1, text: line }));
}
