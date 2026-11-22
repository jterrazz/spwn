import type { MenuBarRead, MenuBarRefused } from '../../domain/menu-bar';

export interface MenuBarPort {
    read: () => Promise<MenuBarRead | MenuBarRefused>;
    show: (shown: boolean) => Promise<MenuBarRead | MenuBarRefused>;
}
