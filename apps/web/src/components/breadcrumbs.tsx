'use client';

import { IconChevronRight } from '@tabler/icons-react';
import { usePathname } from 'next/navigation';

function extractWorldName(id: string): string {
    const [, middle] = id.split('-');
    return middle ? middle.charAt(0).toUpperCase() + middle.slice(1) : id;
}

interface Crumb {
    label: string;
    href: string;
}

export function Breadcrumbs() {
    const pathname = usePathname();

    const crumbs: Crumb[] = [{ label: 'Meson', href: '/' }];

    if (pathname === '/' || pathname === '') {
        return null; // No breadcrumbs on root
    }

    if (pathname === '/architect') {
        crumbs.push({ label: 'Architect', href: '/architect' });
    } else if (pathname === '/knowledge') {
        crumbs.push({ label: 'Knowledge', href: '/knowledge' });
    } else if (pathname.startsWith('/world/')) {
        const parts = pathname.split('/').filter(Boolean);
        const worldId = parts[1];
        if (worldId) {
            crumbs.push({
                label: extractWorldName(worldId),
                href: `/world/${worldId}`,
            });
        }
        const agentName = parts[2];
        if (agentName) {
            crumbs.push({
                label: agentName,
                href: `/world/${worldId}/${agentName}`,
            });
        }
    } else if (pathname.startsWith('/agents/')) {
        const agentName = decodeURIComponent(pathname.split('/')[2] ?? '');
        const worldIdParam =
            typeof globalThis === 'undefined'
                ? null
                : new URLSearchParams(globalThis.location.search).get('world');
        if (worldIdParam) {
            crumbs.push({ label: extractWorldName(worldIdParam), href: `/world/${worldIdParam}` });
        } else {
            crumbs.push({ label: 'Agents', href: '/agents' });
        }
        if (agentName) {
            crumbs.push({
                label: agentName,
                href: pathname + (worldIdParam ? `?world=${worldIdParam}` : ''),
            });
        }
    }

    if (crumbs.length <= 1) {
        return null;
    }

    return (
        <nav className="flex items-center gap-1 px-6 pt-4 pb-0">
            {crumbs.map((crumb, i) => (
                <span className="flex items-center gap-1" key={crumb.href}>
                    {i > 0 && <IconChevronRight className="text-muted-foreground/20" size={12} />}
                    {i < crumbs.length - 1 ? (
                        <a
                            className="text-muted-foreground/40 hover:text-muted-foreground/70 font-mono text-[11px] transition-colors"
                            href={crumb.href}
                        >
                            {crumb.label}
                        </a>
                    ) : (
                        <span className="text-muted-foreground/60 font-mono text-[11px]">
                            {crumb.label}
                        </span>
                    )}
                </span>
            ))}
        </nav>
    );
}
