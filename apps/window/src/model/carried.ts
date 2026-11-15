import type {
    AppsMatrixRow,
    AppsReport,
    SettingsMatrixRow,
    SettingsRow,
} from '@jterrazz/station-api';

import { isSkill } from './apps';

export function appsOn(shelf: 'apps' | 'skills', reports: readonly AppsReport[]): AppsReport[] {
    return reports.filter((report) => isSkill(report.App) === (shelf === 'skills'));
}

export function settingsInCatalog(rows: readonly SettingsRow[]): SettingsRow[] {
    return rows.filter((row) => row.Package.Name !== '');
}

// value: the apps whose matrix row has a cell the engine says its machine holds, none judged here.
export function carriedApps(
    shelf: 'apps' | 'skills',
    reports: readonly AppsReport[],
    matrix: readonly Pick<AppsMatrixRow, 'Cells' | 'Name'>[],
): AppsReport[] {
    const names = new Set(
        matrix.filter((row) => (row.Cells ?? []).some((cell) => cell.Holds)).map((row) => row.Name),
    );
    return appsOn(shelf, reports).filter((report) => names.has(report.App.Name));
}

export function carriedSettings(
    rows: readonly SettingsRow[],
    matrix: readonly Pick<SettingsMatrixRow, 'Name'>[],
): SettingsRow[] {
    const names = new Set(matrix.map((row) => row.Name));
    return settingsInCatalog(rows).filter((row) => names.has(row.Name));
}
