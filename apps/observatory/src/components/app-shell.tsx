"use client";

import { usePathname } from "next/navigation";
import { useState, useEffect, useCallback } from "react";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { MOCK_WORLDS, MOCK_LIMBO } from "@/lib/mock-data";
import type { World, LimboAgent } from "@/lib/mock-data";
import { apiGet } from "@/lib/api-client";

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [worlds, setWorlds] = useState<World[]>(MOCK_WORLDS);
  const [limboAgents, setLimboAgents] = useState<LimboAgent[]>(MOCK_LIMBO);

  // Extract current world ID from URL if on a world page
  const worldMatch = pathname.match(/^\/world\/([^/]+)/);
  const currentWorldId = worldMatch?.[1];

  const fetchWorlds = useCallback(() => {
    apiGet<World[]>("/api/universes", "/api/worlds")
      .then((data) => {
        if (data && data.length > 0) setWorlds(data);
      })
      .catch(() => {
        // keep mock data on error
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
