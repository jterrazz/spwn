'use client';

import { usePathname, useRouter } from 'next/navigation';
import { createContext, useCallback, useContext, useEffect, useState } from 'react';

import { apiGet } from '../api/client';
import { DockerProvider, useDocker } from '../contexts/docker-context';
import type { World } from '../domain/model';
import { checkForUpdatesOnStartup } from '../tauri/updater';
import { AppSidebar } from './app-sidebar';
import { DockerLockScreen } from './docker-lock-screen';
import { ErrorBoundary } from './error-boundary';
import { GlossaryButton } from './glossary-button';
import { SidebarInset, SidebarProvider } from './ui/sidebar';

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

    const isWelcome = pathname === '/welcome';
    const dockerKnown = status !== null;
    const locked = dockerKnown && !ready && !isWelcome;

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
            .catch(() => {});
        // oxlint-disable-next-line react-hooks/exhaustive-deps -- the first-run gate fires once on mount; keying it on `isWelcome` would re-poll onboarding on every navigation
    }, []);

    const worldMatch = /^\/world\/(?<id>[^/]+)/.exec(pathname);
    const urlWorldId = worldMatch?.groups?.id;
    const [lastWorldId, setLastWorldId] = useState<string | undefined>(urlWorldId);

    useEffect(() => {
        if (urlWorldId !== undefined && urlWorldId !== '') {
            setLastWorldId(urlWorldId);
        }
    }, [urlWorldId]);

    const currentWorldId = urlWorldId ?? lastWorldId;

    const fetchWorlds = useCallback(() => {
        if (locked) {
            return;
        }
        void Promise.all([
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
        return () => {
            clearInterval(interval);
        };
    }, [fetchWorlds]);

    useEffect(() => {
        void checkForUpdatesOnStartup();
    }, []);

    return (
        <RefetchContext.Provider value={fetchWorlds}>
            <SidebarProvider>
                <div
                    className="tauri-drag-region fixed top-0 right-0 left-0 z-[100] h-[32px]"
                    data-tauri-drag-region="true"
                    style={{ WebkitAppRegion: 'drag' } as React.CSSProperties}
                />
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
