import type {
    AppsMatrix,
    AppsMatrixRow,
    AppsReport,
    CatalogLivesOn,
    CatalogSetting,
    MachinefileFileView,
} from '@jterrazz/station-api';

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

export type InstallLine = { fetches: string; method: string; systems: string[] };

export function installLinesOf(installs: null | readonly CatalogLivesOn[]): InstallLine[] {
    const lines: InstallLine[] = [];
    for (const one of installs ?? []) {
        const method = one.Method ?? '';
        const fetches = one.Fetches ?? '';
        const systems = one.System === undefined || one.System === '' ? [] : [one.System];
        const same = lines.find((line) => line.method === method && line.fetches === fetches);
        if (same) {
            same.systems.push(...systems.filter((system) => !same.systems.includes(system)));
        } else {
            lines.push({ fetches, method, systems });
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
