'use client';

import { usePathname, useRouter } from 'next/navigation';
import { createContext, useCallback, useContext, useRef, useState } from 'react';

import { apiGet } from '../api/client';
import { DockerProvider, useDocker } from '../contexts/docker-context';
import type { World } from '../domain/model';
import { useBackgroundTask } from '../hooks/use-background-task';
import { checkForUpdatesOnStartup } from '../tauri/updater';
import { AppSidebar } from './app-sidebar';
import { DockerLockScreen } from './docker-lock-screen';
import { ErrorBoundary } from './error-boundary';
import { GlossaryButton } from './glossary-button';
import { SidebarInset, SidebarProvider } from './ui/sidebar';

// outside: Tauri reads -webkit-app-region to let the window be dragged by this strip, a key React's CSSProperties does not know
declare module 'react' {
    interface CSSProperties {
        WebkitAppRegion?: 'drag' | 'no-drag';
    }
}

const WORLD_ADDRESS = /^\/world\/(?<id>[^/]+)/u;

const RefetchContext = createContext<() => void>(() => undefined);

export function useRefetch() {
    return useContext(RefetchContext);
}

async function optional<Value>(promise: Promise<Value>): Promise<undefined | Value> {
    try {
        return await promise;
    } catch {
        return undefined;
    }
}

function useOnboardingGate(onWelcome: boolean): void {
    const router = useRouter();
    const openedOnWelcome = useRef(onWelcome);
    const gate = useCallback(async () => {
        const onboarding = await optional(apiGet<{ completed: boolean }>('/api/system/onboarding'));
        if (!openedOnWelcome.current && onboarding?.completed === false) {
            router.replace('/welcome');
        }
    }, [router]);
    useBackgroundTask(gate);
}

function useCurrentWorld(pathname: string): string | undefined {
    const named = WORLD_ADDRESS.exec(pathname)?.groups?.id;
    const [last, setLast] = useState(named);
    if (named !== undefined && named !== last) {
        setLast(named);
    }
    return named ?? last;
}

function useSidebarData(locked: boolean) {
    const [worlds, setWorlds] = useState<World[]>([]);
    const [loading, setLoading] = useState(true);

    const fetchWorlds = useCallback(async () => {
        if (locked) {
            return;
        }
        setWorlds((await optional(apiGet<World[]>('/api/worlds'))) ?? []);
        setLoading(false);
    }, [locked]);
    useBackgroundTask(fetchWorlds, 5000);

    const refetch = useCallback(() => {
        void fetchWorlds();
    }, [fetchWorlds]);
    return { loading, refetch, worlds };
}

async function checkUpdates(): Promise<void> {
    await checkForUpdatesOnStartup();
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
    const { ready, status } = useDocker();
    const onWelcome = pathname === '/welcome';
    const locked = status !== null && !ready && !onWelcome;
    const sidebar = useSidebarData(locked);
    const currentWorldId = useCurrentWorld(pathname);
    useOnboardingGate(onWelcome);
    useBackgroundTask(checkUpdates);

    return (
        <RefetchContext.Provider value={sidebar.refetch}>
            <SidebarProvider>
                <div
                    className="tauri-drag-region fixed top-0 right-0 left-0 z-[100] h-[32px]"
                    data-tauri-drag-region="true"
                    style={{ WebkitAppRegion: 'drag' }}
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
                        loading={sidebar.loading}
                        worlds={sidebar.worlds}
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
