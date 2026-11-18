import type { CatalogApp } from '@jterrazz/station-api';

export function isSkill({ AppKind }: Pick<CatalogApp, 'AppKind'>): boolean {
    return AppKind === 'skill';
}
