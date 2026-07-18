'use client';

import { usePathname, useRouter } from 'next/navigation';
import { createContext, useCallback, useContext, useEffect, useState } from 'react';

import { AppSidebar } from '@/components/app-sidebar';
import { DockerLockScreen } from '@/components/docker-lock-screen';
import { ErrorBoundary } from '@/components/error-boundary';
import { GlossaryButton } from '@/components/glossary-button';
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { DockerProvider, useDocker } from '@/contexts/docker-context';
import { apiGet } from '@/lib/api-client';
import { checkForUpdatesOnStartup } from '@/lib/tauri-updater';
import type { World } from '@/lib/types';

// ── Refetch context: allows any child to trigger an immediate data refetch ──
const RefetchContext = createContext<() => void>(() => {});
export function useRefetch() {
    return useContext(RefetchContext);
}

interface StatusData {
    worlds: number;
    agents: number;
    running: number;
}

export function AppShell({ children }: { children: React.ReactNode }) {
    return (
        <DockerProvider>
            <AppShellInner>{children}</AppShellInner>
        </DockerProvider>
    );
}

function AppShellInner({ children }: { children: React.ReactNode }) {
    const pathname = usePathname();
    const router = useRouter();
    const { ready, status } = useDocker();
    const [worlds, setWorlds] = useState<World[]>([]);
    const [sidebarLoading, setSidebarLoading] = useState(true);
    const [statusData, setStatusData] = useState<null | StatusData>(null);

    // The /welcome wizard is the user's path back to a working state, so it
    // Is the one screen the lock must NOT cover. Same goes for redirect-to-
    // Welcome on first run.
    const isWelcome = pathname === '/welcome';
    const dockerKnown = status !== null;
    const locked = dockerKnown && !ready && !isWelcome;

    // First-run gate: redirect to /welcome if onboarding hasn't been
    // Completed yet. Runs once on mount; the wizard itself polls for state.
    useEffect(() => {
        if (isWelcome) {
            return;
        }
        apiGet<{ completed: boolean }>('/api/system/onboarding')
            .then((s) => {
                if (!s.completed) {
                    router.replace('/welcome');
                }
            })
            .catch(() => {
                /* API not reachable - render normally */
            });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Track the last-visited world so the sidebar remembers the selection
    // Across page navigations (e.g. going from /world/w-x to /agents).
    const worldMatch = pathname.match(/^\/world\/(?<id>[^/]+)/);
    const urlWorldId = worldMatch?.groups?.id;
    const [lastWorldId, setLastWorldId] = useState<string | undefined>(urlWorldId);

    useEffect(() => {
        if (urlWorldId) {
            setLastWorldId(urlWorldId);
        }
    }, [urlWorldId]);

    const currentWorldId = urlWorldId ?? lastWorldId;

    const fetchWorlds = useCallback(() => {
        // Don't hammer the API while Docker is down - every world endpoint
        // Depends on it and will just return an empty list anyway.
        if (locked) {
            return;
        }
        Promise.all([
            apiGet<World[]>('/api/worlds').catch(() => [] as World[]),
            apiGet<StatusData>('/api/status').catch(() => null),
        ]).then(([worldData, sData]) => {
            setStatusData(sData);
            setWorlds(worldData ?? []);
            setSidebarLoading(false);
        });
    }, [locked]);

    useEffect(() => {
        fetchWorlds();
        const interval = setInterval(fetchWorlds, 5000);
        return () => clearInterval(interval);
    }, [fetchWorlds]);

    // One-shot update check when the native Tauri app boots.
    useEffect(() => {
        void checkForUpdatesOnStartup();
    }, []);

    return (
        <RefetchContext.Provider value={fetchWorlds}>
            <SidebarProvider>
                {/* Drag region for window movement - stays interactive even
            while the rest of the app is locked. */}
                <div
                    className="tauri-drag-region fixed top-0 left-0 right-0 h-[32px] z-[100]"
                    data-tauri-drag-region="true"
                    style={{ WebkitAppRegion: 'drag' } as React.CSSProperties}
                />
                {/* Sidebar dims and becomes non-interactive while locked, but
            stays visible so the app's identity is preserved. */}
                <div
                    aria-hidden={locked || undefined}
                    className={
                        locked
                            ? 'pointer-events-none opacity-40 transition-opacity duration-200'
                            : 'transition-opacity duration-200'
                    }
                >
                    <AppSidebar
                        currentWorldId={currentWorldId}
                        loading={sidebarLoading}
                        statusData={statusData}
                        worlds={worlds}
                    />
                </div>
                <SidebarInset>
                    {locked ? <DockerLockScreen /> : <ErrorBoundary>{children}</ErrorBoundary>}
                </SidebarInset>
                <GlossaryButton />
            </SidebarProvider>
        </RefetchContext.Provider>
    );
}
