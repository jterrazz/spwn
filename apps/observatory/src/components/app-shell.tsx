"use client";

import { usePathname } from "next/navigation";
import { useState, useEffect, useCallback } from "react";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { Breadcrumbs } from "@/components/breadcrumbs";
import type { World, LimboAgent } from "@/lib/types";
import { apiGet } from "@/lib/api-client";

interface AgentListItem {
  name: string;
  path: string;
  layers: Record<string, string[]>;
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [worlds, setWorlds] = useState<World[]>([]);
  const [limboAgents, setLimboAgents] = useState<LimboAgent[]>([]);

  // Extract current world ID from URL if on a world page
  const worldMatch = pathname.match(/^\/world\/([^/]+)/);
  const currentWorldId = worldMatch?.[1];

  const fetchWorlds = useCallback(() => {
    Promise.all([
      apiGet<World[]>("/api/universes", "/api/worlds").catch(() => [] as World[]),
      apiGet<AgentListItem[]>("/api/agents", "/api/agents").catch(() => [] as AgentListItem[]),
    ]).then(([worldData, agentData]) => {
      const w = worldData ?? [];
      setWorlds(w);
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
          layers: Object.values(a.layers ?? {}).filter((f) => f.length > 0).length,
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
    <SidebarProvider>
      <AppSidebar
        worlds={worlds}
        limboAgents={limboAgents}
        currentWorldId={currentWorldId}
      />
      <SidebarInset>
        <Breadcrumbs />
        {children}
      </SidebarInset>
    </SidebarProvider>
  );
}
