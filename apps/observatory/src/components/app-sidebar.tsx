"use client";

import { usePathname } from "next/navigation";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarFooter,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ThemeToggle } from "@/components/theme-toggle";
import type { World, LimboAgent } from "@/lib/mock-data";

interface AppSidebarProps {
  worlds: World[];
  limboAgents: LimboAgent[];
  currentWorldId?: string;
}

const STATUS_DOT: Record<string, string> = {
  running: "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]",
  idle: "bg-yellow-500 shadow-[0_0_6px_rgba(234,179,8,0.5)]",
  stopped: "bg-white/20",
  creating: "bg-blue-400 shadow-[0_0_6px_rgba(96,165,250,0.5)]",
};

function extractName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2
    ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1)
    : id;
}

export function AppSidebar({ worlds, limboAgents, currentWorldId }: AppSidebarProps) {
  const pathname = usePathname();

  return (
    <Sidebar>
      {/* ── Header: brand + search hint ── */}
      <SidebarHeader className="px-4 pt-4 pb-2">
        <a href="/" className="flex items-center gap-2">
          <span className="text-sm tracking-[0.15em] font-heading text-foreground/90">
            ⬡ spwn
          </span>
        </a>
        <button
          className="mt-3 w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-[11px] text-muted-foreground/30 hover:text-muted-foreground/50 hover:bg-white/[0.03] transition-colors"
          onClick={() => {/* TODO: command palette */}}
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="opacity-50">
            <circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <span className="flex-1 text-left">Search...</span>
          <kbd className="font-mono text-[9px] px-1.5 py-0.5 rounded bg-white/[0.04] border border-white/[0.06]">⌘K</kbd>
        </button>
      </SidebarHeader>

      <SidebarContent>
        {/* ── Navigation ── */}
        <SidebarGroup>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={pathname === "/"}
                onClick={() => window.location.href = "/"}
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="opacity-60"><circle cx="12" cy="12" r="10" /><circle cx="12" cy="12" r="3" /></svg>
                <span>Overview</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={pathname === "/architect"}
                onClick={() => window.location.href = "/architect"}
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="opacity-60"><rect x="2" y="2" width="20" height="20" rx="2" /><line x1="12" y1="2" x2="12" y2="22" /><line x1="2" y1="12" x2="22" y2="12" /></svg>
                <span>Architect</span>
                <div className="ml-auto w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_4px_rgba(34,197,94,0.5)]" />
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

        <SidebarSeparator />

        {/* ── Worlds ── */}
        <SidebarGroup>
          <SidebarGroupLabel className="text-[9px] uppercase tracking-[0.15em] text-muted-foreground/30 font-mono">
            Worlds
          </SidebarGroupLabel>
          <SidebarMenu>
            {worlds.map((world) => {
              const name = extractName(world.id);
              const isWorldActive = pathname.startsWith(`/world/${world.id}`);

              return (
                <Collapsible key={world.id} defaultOpen={isWorldActive || worlds.length <= 4}>
                  <SidebarMenuItem>
                    <div className="flex w-full items-center">
                      <a
                        href={`/world/${world.id}`}
                        className={`flex flex-1 items-center gap-2.5 rounded-md px-2 py-1.5 text-xs transition-colors ${isWorldActive ? "bg-white/[0.06] text-foreground" : "text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.03]"}`}
                      >
                        <div className={`w-1.5 h-1.5 rounded-full shrink-0 ${STATUS_DOT[world.status]}`} />
                        <span className="font-medium">{name}</span>
                      </a>
                      <CollapsibleTrigger className="px-2 py-1.5 text-muted-foreground/25 hover:text-muted-foreground/50 transition-colors">
                        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="6 9 12 15 18 9" /></svg>
                      </CollapsibleTrigger>
                    </div>
                    <CollapsibleContent>
                      <SidebarMenuSub>
                        {world.agents.map((agent) => (
                          <SidebarMenuSubItem key={agent.name}>
                            <SidebarMenuSubButton
                              isActive={pathname === `/world/${world.id}/${agent.name}`}
                              onClick={() => window.location.href = `/world/${world.id}/${agent.name}`}
                              className="text-[11px]"
                            >
                              <div className={`w-1 h-1 rounded-full shrink-0 ${STATUS_DOT[agent.status]}`} />
                              <span>{agent.name}</span>
                              <span className="ml-auto text-[9px] text-muted-foreground/25 capitalize">{agent.tier}</span>
                            </SidebarMenuSubButton>
                          </SidebarMenuSubItem>
                        ))}
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </SidebarMenuItem>
                </Collapsible>
              );
            })}
          </SidebarMenu>
        </SidebarGroup>

        {/* ── Limbo agents ── */}
        {limboAgents.length > 0 && (
          <SidebarGroup>
            <SidebarGroupLabel className="text-[9px] uppercase tracking-[0.15em] text-muted-foreground/30 font-mono">
              Limbo
            </SidebarGroupLabel>
            <SidebarMenu>
              {limboAgents.map((agent) => (
                <SidebarMenuItem key={agent.name}>
                  <SidebarMenuButton className="text-[11px] text-muted-foreground/35">
                    <div className="w-1 h-1 rounded-full bg-white/10 shrink-0" />
                    <span>{agent.name}</span>
                    <span className="ml-auto text-[9px] font-mono text-muted-foreground/20">
                      {agent.layers}/6
                    </span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroup>
        )}

      </SidebarContent>

      {/* ── Footer ── */}
      <SidebarFooter className="px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded-full bg-white/[0.06] flex items-center justify-center text-[10px] font-mono text-muted-foreground/40">
              J
            </div>
            <span className="text-[11px] text-muted-foreground/40">jterrazz</span>
          </div>
          <ThemeToggle />
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
