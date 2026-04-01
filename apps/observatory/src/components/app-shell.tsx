"use client";

import { usePathname } from "next/navigation";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { MOCK_WORLDS, MOCK_LIMBO } from "@/lib/mock-data";

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  // Extract current world ID from URL if on a world page
  const worldMatch = pathname.match(/^\/world\/([^/]+)/);
  const currentWorldId = worldMatch?.[1];

  return (
    <SidebarProvider>
      <AppSidebar
        worlds={MOCK_WORLDS}
        limboAgents={MOCK_LIMBO}
        currentWorldId={currentWorldId}
      />
      <SidebarInset>{children}</SidebarInset>
    </SidebarProvider>
  );
}
