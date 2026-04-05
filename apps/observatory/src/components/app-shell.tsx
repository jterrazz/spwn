"use client";

import { usePathname } from "next/navigation";
import { useState, useEffect, useCallback, createContext, useContext } from "react";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { ErrorBoundary } from "@/components/error-boundary";
import type { World } from "@/lib/types";
import { apiGet } from "@/lib/api-client";
import { checkForUpdatesOnStartup } from "@/lib/tauri-updater";

// ── Refetch context: allows any child to trigger an immediate data refetch ──
const RefetchContext = createContext<() => void>(() => {});
export function useRefetch() { return useContext(RefetchContext); }

interface StatusData {
  worlds: number;
  agents: number;
  running: number;
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [worlds, setWorlds] = useState<World[]>([]);
  const [sidebarLoading, setSidebarLoading] = useState(true);
  const [statusData, setStatusData] = useState<StatusData | null>(null);

  // Extract current world ID from URL if on a world page
  const worldMatch = pathname.match(/^\/world\/([^/]+)/);
  const currentWorldId = worldMatch?.[1];

  const fetchWorlds = useCallback(() => {
    Promise.all([
      apiGet<World[]>("/api/universes").catch(() => [] as World[]),
      apiGet<StatusData>("/api/status").catch(() => null),
    ]).then(([worldData, sData]) => {
      setStatusData(sData);
      setWorlds(worldData ?? []);
      setSidebarLoading(false);
    });
  }, []);

  useEffect(() => {
    fetchWorlds();
    const interval = setInterval(fetchWorlds, 5000);
    return () => clearInterval(interval);
  }, [fetchWorlds]);

  // One-shot update check when the native Tauri app boots.
  useEffect(() => { void checkForUpdatesOnStartup(); }, []);

  return (
    <RefetchContext.Provider value={fetchWorlds}>
      <SidebarProvider>
        {/* Drag region for window movement */}
        <div
          data-tauri-drag-region="true"
          className="tauri-drag-region fixed top-0 left-0 right-0 h-[32px] z-[100]"
          style={{ WebkitAppRegion: "drag" } as React.CSSProperties}
        />
        <AppSidebar
          worlds={worlds}
          currentWorldId={currentWorldId}
          loading={sidebarLoading}
          statusData={statusData}
        />
        <SidebarInset>
          <ErrorBoundary>
            {children}
          </ErrorBoundary>
        </SidebarInset>
      </SidebarProvider>
    </RefetchContext.Provider>
  );
}
