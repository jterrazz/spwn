"use client";

import { SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { SidebarInset } from "@/components/ui/sidebar";
import { MOCK_WORLDS, MOCK_DRIFTING } from "@/lib/mock-data";
import { useParams } from "next/navigation";

export default function WorldLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const params = useParams();
  const currentWorldId = params.id as string;

  return (
    <SidebarProvider>
      <AppSidebar
        worlds={MOCK_WORLDS}
        driftingAgents={MOCK_DRIFTING}
        currentWorldId={currentWorldId}
      />
      <SidebarInset>{children}</SidebarInset>
    </SidebarProvider>
  );
}
