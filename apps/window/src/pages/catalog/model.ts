import type { AppsReport, SettingsRow } from '@jterrazz/station-api';

import { isSkill } from '../../engine/provisional';
import { matchesQuery } from '../../model/search';

export type Shelf = 'apps' | 'settings' | 'skills';

export const EVERY_CATEGORY = '\u0000every';

export type Package = {
    name: string;
    category: string;
    description: string;
};

export function categoriesOf(packages: readonly Package[]): readonly string[] {
    const seen: string[] = [];
    for (const { category } of packages) {
        if (!seen.includes(category)) {
            seen.push(category);
        }
    }
    return seen;
}

export function appsOn(shelf: 'apps' | 'skills', reports: readonly AppsReport[]): AppsReport[] {
    return reports.filter((report) => isSkill(report.App.Category) === (shelf === 'skills'));
}

export function packageOfApp(report: AppsReport): Package {
    return {
        category: report.App.Category,
        description: report.App.Description,
        name: report.App.Name,
    };
}

export function settingsInCatalog(rows: readonly SettingsRow[]): SettingsRow[] {
    return rows.filter((row) => row.Package.Name !== '');
}

export function packageOfSetting(row: SettingsRow): Package {
    return { category: row.Package.Category, description: row.Package.Description, name: row.Name };
}

export function matches(item: Package, category: string, query: string): boolean {
    if (category !== EVERY_CATEGORY && item.category !== category) {
        return false;
    }
    return matchesQuery(query, [item.name, item.description, item.category]);
}

export function countIn(packages: readonly Package[], category: string): number {
    return packages.filter((item) => item.category === category).length;
}
