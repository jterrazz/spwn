import type {
    AppsMatrix,
    AppsMatrixRow,
    AppsReport,
    CatalogLivesOn,
    CatalogSetting,
    MachinefileFileView,
} from '@jterrazz/station-api';

import { SEPARATOR } from '../../../domain/separator';

export function rowOf(matrix: AppsMatrix | undefined, name: string): AppsMatrixRow | undefined {
    return (matrix?.Rows ?? []).find((row) => row.Name === name);
}

export function carriedBy(matrix: AppsMatrix | undefined, name: string): string[] | undefined {
    const row = rowOf(matrix, name);
    if (matrix === undefined) {
        return undefined;
    }
    const columns = matrix.Columns ?? [];
    return (row?.Cells ?? []).flatMap((cell, index) => {
        const column = columns[index];
        return column !== undefined && cell.Holds ? [column.Name] : [];
    });
}

export function declaredIn(
    files: readonly MachinefileFileView[],
    chapter: 'Apps' | 'Settings',
    name: string,
): string[] {
    return files
        .filter((file) => ((chapter === 'Apps' ? file.Apps : file.Settings) ?? []).includes(name))
        .map((file) => file.Name);
}

export type System = NonNullable<CatalogLivesOn['Kinds']>[number];

export type InstallLine = { systems: System[]; method: string; target: string };

function textOf(value: unknown): string {
    if (typeof value === 'string') {
        return value;
    }
    return Array.isArray(value)
        ? value.filter((one): one is string => typeof one === 'string').join(' ')
        : '';
}

// outside: CatalogLivesOn.Install is typed unknown, its shape the method's: its text values are drawn as sent, in
// outside: the order sent, until the engine words what an install fetches.
export function targetOf(install: unknown): string {
    if (typeof install !== 'object' || install === null) {
        return textOf(install);
    }
    const values: unknown[] = Object.values(install);
    return values
        .map(textOf)
        .filter((text) => text !== '')
        .join(SEPARATOR);
}

export function installLinesOf(installs: null | readonly CatalogLivesOn[]): InstallLine[] {
    const lines: InstallLine[] = [];
    for (const one of installs ?? []) {
        const method = one.Method ?? '';
        const target = targetOf(one.Install);
        const systems = one.Kinds ?? [];
        const same = lines.find((line) => line.method === method && line.target === target);
        if (same) {
            same.systems.push(...systems.filter((system) => !same.systems.includes(system)));
        } else {
            lines.push({ method, systems: [...systems], target });
        }
    }
    return lines;
}

export function usedBy(name: string, reports: readonly AppsReport[]): string[] {
    return reports
        .filter((report) => (report.App.Requires ?? []).includes(name))
        .map((report) => report.App.Name);
}

export function needsOf(setting: CatalogSetting): { oneOf: boolean; names: string[] } {
    const names = [setting.Requires, ...(setting.OrRequires ?? [])].filter((name) => name !== '');
    return { names, oneOf: names.length > 1 };
}
