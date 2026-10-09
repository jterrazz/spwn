'use client';

import { usePathname } from 'next/navigation';

import type { World } from '../domain/model';
import { GettingStarted } from './sidebar/getting-started';
import { Navigation } from './sidebar/navigation';
import { SpwnSidebarFooter } from './sidebar/sidebar-footer';
import { SpwnSidebarHeader } from './sidebar/sidebar-header';
import { WorldRoster } from './sidebar/world-roster';
import { WorldSwitcher } from './sidebar/world-switcher';
import { Sidebar, SidebarContent, SidebarGroup } from './ui/sidebar';

interface AppSidebarProps {
    worlds: World[];
    currentWorldId?: string | undefined;
    loading?: boolean | undefined;
}

/** The app's sidebar: its destinations, the world it follows with that world's agents, and its foot. */
export function AppSidebar({ worlds, currentWorldId, loading }: AppSidebarProps) {
    const pathname = usePathname();
    const followed =
        currentWorldId ??
        worlds.find((world) => pathname.startsWith(`/world/${world.id}`))?.id ??
        worlds[0]?.id;
    const selected = worlds.find((world) => world.id === followed);

    return (
        <Sidebar>
            <SpwnSidebarHeader />
            <SidebarContent>
                <Navigation />
                {loading !== true && worlds.length === 0 && <GettingStarted />}
                <SidebarGroup>
                    <WorldSwitcher selected={selected} worlds={worlds} />
                    {selected !== undefined && <WorldRoster world={selected} />}
                </SidebarGroup>
            </SidebarContent>
            <SpwnSidebarFooter />
        </Sidebar>
    );
}
