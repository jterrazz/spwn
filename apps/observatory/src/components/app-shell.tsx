"use client";

import { usePathname } from "next/navigation";
import { useState, useEffect, useCallback, createContext, useContext } from "react";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { ErrorBoundary } from "@/components/error-boundary";
import type { World, LimboAgent } from "@/lib/types";
import { apiGet } from "@/lib/api-client";

// ── Refetch context: allows any child to trigger an immediate data refetch ──
const RefetchContext = createContext<() => void>(() => {});
export function useRefetch() { return useContext(RefetchContext); }

interface AgentListItem {
  name: string;
  path: string;
  layers: Record<string, string[]>;
}

interface StatusData {
  worlds: number;
  agents: number;
  running: number;
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [worlds, setWorlds] = useState<World[]>([]);
  const [limboAgents, setLimboAgents] = useState<LimboAgent[]>([]);
  const [sidebarLoading, setSidebarLoading] = useState(true);
  const [statusData, setStatusData] = useState<StatusData | null>(null);

  // Extract current world ID from URL if on a world page
  const worldMatch = pathname.match(/^\/world\/([^/]+)/);
  const currentWorldId = worldMatch?.[1];

  const fetchWorlds = useCallback(() => {
    Promise.all([
      apiGet<World[]>("/api/universes", "/api/worlds").catch(() => [] as World[]),
      apiGet<AgentListItem[]>("/api/agents", "/api/agents").catch(() => [] as AgentListItem[]),
      apiGet<StatusData>("/api/status", "/api/status").catch(() => null),
    ]).then(([worldData, agentData, sData]) => {
      setStatusData(sData);
      const w = worldData ?? [];
      setWorlds(w);
      setSidebarLoading(false);
      // Limbo agents = agents not in any active world
      const activeAgentNames = new Set<string>();
      for (const world of w) {
        for (const a of world.agents) {
          activeAgentNames.add(a.name);
        }
      }
      const limbo = (agentData ?? [])
        .filter((a) => !activeAgentNames.has(a.name))
        .map((a) => ({
          name: a.name,
          layers: Object.values(a.layers ?? {}).filter((f) => Array.isArray(f) && f.length > 0).length,
        }));
      setLimboAgents(limbo);
    });
  }, []);

  useEffect(() => {
    fetchWorlds();
    const interval = setInterval(fetchWorlds, 5000);
    return () => clearInterval(interval);
  }, [fetchWorlds]);

  return (
    <RefetchContext.Provider value={fetchWorlds}>
      <SidebarProvider>
        <AppSidebar
          worlds={worlds}
          limboAgents={limboAgents}
          currentWorldId={currentWorldId}
          loading={sidebarLoading}
          statusData={statusData}
        />
        <SidebarInset>
          {/* Thin drag strip for main content area — only the top 38px is draggable */}
          <div
            data-tauri-drag-region="true"
            className="absolute top-0 left-0 right-0 h-[38px] z-10"
          />
          {/* Breadcrumbs with top padding for macOS title bar overlay */}
          <Breadcrumbs />
          <ErrorBoundary>
            {children}
          </ErrorBoundary>
        </SidebarInset>
      </SidebarProvider>
    </RefetchContext.Provider>
  );
}
