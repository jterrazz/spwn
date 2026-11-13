import type { PlainPage } from '../../shell/route';
import type { Shelf } from './model';

export const SHELVES = {
    apps: { page: 'catalog-apps', title: 'Apps' },
    settings: { page: 'catalog-settings', title: 'Settings' },
    skills: { page: 'catalog-skills', title: 'Skills' },
} as const satisfies Record<Shelf, { page: PlainPage; title: string }>;

export function categoryWord(category: string): string {
    return category === '' ? 'No category given' : category;
}
