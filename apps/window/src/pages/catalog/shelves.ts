import type { PlainPage } from '../../shell/route';
import type { Shelf } from './model';

export const SHELVES = {
    apps: {
        description: 'Programs, command-line tools, runtimes and the managers that install them',
        icon: 'apps',
        page: 'catalog-apps',
        title: 'Apps',
    },
    settings: {
        description:
            'What station writes or switches on a machine, each one undoable where it can be',
        icon: 'settings',
        page: 'catalog-settings',
        title: 'Settings',
    },
    skills: {
        description: 'What an agent knows how to do, added to it on a machine',
        icon: 'skills',
        page: 'catalog-skills',
        title: 'Skills',
    },
} as const satisfies Record<
    Shelf,
    {
        description: string;
        icon: 'apps' | 'settings' | 'skills';
        page: PlainPage;
        title: string;
    }
>;

export function categoryWord(category: string): string {
    return category === '' ? 'No category given' : category;
}
