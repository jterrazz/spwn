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

function heldNames(
    matrix: readonly (
        | Pick<AppsMatrixRow, 'Cells' | 'Name'>
        | Pick<SettingsMatrixRow, 'Cells' | 'Name'>
    )[],
): Set<string> {
    return new Set(
        matrix.filter((row) => (row.Cells ?? []).some((cell) => cell.Holds)).map((row) => row.Name),
    );
}

// value: the apps whose matrix row has a cell the engine says its machine holds, none judged here.
export function carriedApps(
    shelf: 'apps' | 'skills',
    reports: readonly AppsReport[],
    matrix: readonly Pick<AppsMatrixRow, 'Cells' | 'Name'>[],
): AppsReport[] {
    const names = heldNames(matrix);
    return appsOn(shelf, reports).filter((report) => names.has(report.App.Name));
}

// value: the settings whose matrix row has a cell the engine says its machine holds, none judged here.
export function carriedSettings(
    rows: readonly SettingsRow[],
    matrix: readonly Pick<SettingsMatrixRow, 'Cells' | 'Name'>[],
): SettingsRow[] {
    const names = heldNames(matrix);
    return settingsInCatalog(rows).filter((row) => names.has(row.Name));
}
