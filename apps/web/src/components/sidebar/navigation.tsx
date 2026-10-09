'use client';

import {
    IconAssemblyFilled,
    IconBinaryTreeFilled,
    IconBoltFilled,
    IconHexagonFilled,
    IconSettingsFilled,
    IconUserFilled,
} from '@tabler/icons-react';
import { usePathname, useRouter } from 'next/navigation';

import {
    SidebarGroup,
    SidebarGroupLabel,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
} from '../ui/sidebar';

interface Destination {
    href: string;
    label: string;
    icon: React.ReactNode;
    active: (pathname: string) => boolean;
}

const SYSTEM: Destination[] = [
    {
        active: (path) => path === '/architect',
        href: '/architect',
        icon: <IconHexagonFilled size={16} />,
        label: 'Architect',
    },
    {
        active: (path) => path === '/providers',
        href: '/providers',
        icon: <IconSettingsFilled size={16} />,
        label: 'Settings',
    },
];

const UNIVERSE: Destination[] = [
    {
        active: (path) => path === '/',
        href: '/',
        icon: <IconAssemblyFilled size={16} />,
        label: 'Worlds',
    },
    {
        active: (path) => path === '/agents' || path.startsWith('/agents/'),
        href: '/agents',
        icon: <IconUserFilled size={16} />,
        label: 'Agents',
    },
    {
        active: (path) => path === '/tools',
        href: '/tools',
        icon: <IconBoltFilled size={16} />,
        label: 'Tools',
    },
    {
        active: (path) => path === '/organizations',
        href: '/organizations',
        icon: <IconBinaryTreeFilled size={16} />,
        label: 'Organizations',
    },
];

function Destinations({ destinations }: { destinations: Destination[] }) {
    const pathname = usePathname();
    const router = useRouter();
    return (
        <SidebarMenu>
            {destinations.map((destination) => (
                <SidebarMenuItem key={destination.href}>
                    <SidebarMenuButton
                        isActive={destination.active(pathname)}
                        onClick={() => {
                            router.push(destination.href);
                        }}
                    >
                        {destination.icon}
                        <span>{destination.label}</span>
                    </SidebarMenuButton>
                </SidebarMenuItem>
            ))}
        </SidebarMenu>
    );
}

/** The app's destinations: the architect and the settings, then the universe — worlds, agents, tools, organizations. */
export function Navigation() {
    return (
        <>
            <SidebarGroup className="pt-0">
                <Destinations destinations={SYSTEM} />
            </SidebarGroup>
            <SidebarGroup>
                <SidebarGroupLabel className="text-sidebar-foreground/30 text-[10px] tracking-widest uppercase">
                    Universe
                </SidebarGroupLabel>
                <Destinations destinations={UNIVERSE} />
            </SidebarGroup>
        </>
    );
}
