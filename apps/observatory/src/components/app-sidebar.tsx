"use client";

import { usePathname } from "next/navigation";
import {
  IconWorldFilled,
  IconHexagonFilled,
  IconSearch,
  IconBoltFilled,
  IconMessageFilled,
  IconMoonFilled,
  IconCircleFilled,
  IconChevronDown,
  IconGhostFilled,
  IconPackage,
  IconPlus,
} from "@tabler/icons-react";
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
import { LiveStatus } from "@/components/live-status";
import type { World, LimboAgent } from "@/lib/types";
import { apiAction } from "@/lib/api-client";

interface AppSidebarProps {
  worlds: World[];
  limboAgents: LimboAgent[];
  currentWorldId?: string;
}

// Agent status → Tabler filled icon + color
const AGENT_ICON: Record<string, { icon: typeof IconBoltFilled; color: string; dim: boolean }> = {
  running:  { icon: IconBoltFilled, color: "text-green-400", dim: false },
  waiting:  { icon: IconMessageFilled, color: "text-amber-400 animate-pulse", dim: false },
  sleeping: { icon: IconMoonFilled, color: "text-purple-400", dim: false },
  idle:     { icon: IconCircleFilled, color: "text-white/20", dim: true },
  stopped:  { icon: IconCircleFilled, color: "text-white/10", dim: true },
};

const WORLD_DOT: Record<string, string> = {
  running: "text-green-500",
  idle: "text-white/25",
  stopped: "text-white/10",
  creating: "text-blue-400",
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
          <IconSearch size={13} className="opacity-40" />
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
                <IconWorldFilled size={16} className="opacity-50" />
                <span>Overview</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={pathname === "/architect"}
                onClick={() => window.location.href = "/architect"}
              >
                <IconHexagonFilled size={16} className="opacity-50" />
                <span>Architect</span>
                <div className="ml-auto w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_4px_rgba(34,197,94,0.5)]" />
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={pathname === "/marketplace"}
                onClick={() => window.location.href = "/marketplace"}
              >
                <IconPackage size={16} className="opacity-50" />
                <span>Marketplace</span>
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
                        <IconCircleFilled size={8} className={`shrink-0 ${WORLD_DOT[world.status] ?? "text-white/10"}`} />
                        <span className="font-medium">{name}</span>
                      </a>
                      <CollapsibleTrigger className="px-2 py-1.5 text-muted-foreground/25 hover:text-muted-foreground/50 transition-colors">
                        <IconChevronDown size={12} />
                      </CollapsibleTrigger>
                    </div>
                    <CollapsibleContent>
                      <SidebarMenuSub>
                        {world.agents.map((agent) => {
                          const s = AGENT_ICON[agent.status] ?? AGENT_ICON.stopped;
                          const StatusIcon = s.icon;
                          return (
                            <SidebarMenuSubItem key={agent.name}>
                              <SidebarMenuSubButton
                                isActive={pathname === `/world/${world.id}/${agent.name}`}
                                onClick={() => window.location.href = `/world/${world.id}/${agent.name}`}
                                className="text-[11px]"
                              >
                                <span className={s.dim ? "opacity-40" : ""}>{agent.name}</span>
                                <StatusIcon size={12} className={`ml-auto shrink-0 ${s.color}`} />
                              </SidebarMenuSubButton>
                            </SidebarMenuSubItem>
                          );
                        })}
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </SidebarMenuItem>
                </Collapsible>
              );
            })}
          </SidebarMenu>
        </SidebarGroup>

        {/* ── Limbo agents ── */}
        <SidebarGroup>
          <SidebarGroupLabel className="text-[9px] uppercase tracking-[0.15em] text-muted-foreground/30 font-mono">
            Limbo
          </SidebarGroupLabel>
          <SidebarMenu>
            {limboAgents.map((agent) => (
              <SidebarMenuItem key={agent.name}>
                <SidebarMenuButton
                  className="text-[11px] text-muted-foreground/35"
                  isActive={pathname === `/agents/${agent.name}`}
                  onClick={() => window.location.href = `/agents/${agent.name}`}
                >
                  <IconGhostFilled size={12} className="opacity-20 shrink-0" />
                  <span>{agent.name}</span>
                  <span className="ml-auto text-[9px] font-mono text-muted-foreground/20">
                    {agent.layers}/6
                  </span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
            {limboAgents.length === 0 && (
              <SidebarMenuItem>
                <span className="text-[10px] text-muted-foreground/20 px-2 py-1">No agents in limbo</span>
              </SidebarMenuItem>
            )}
            <SidebarMenuItem>
              <SidebarMenuButton
                className="text-[11px] text-muted-foreground/25 hover:text-muted-foreground/50"
                onClick={async () => {
                  const name = prompt("Agent name:");
                  if (!name?.trim()) return;
                  await apiAction("/api/agents", { name: name.trim() }, "/api/agents/create");
                  window.location.reload();
                }}
              >
                <IconPlus size={12} className="opacity-30 shrink-0" />
                <span>New Agent</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

      </SidebarContent>

      {/* ── Footer ── */}
      <SidebarFooter className="px-4 py-3 space-y-2">
        <div className="flex items-center justify-between">
          <LiveStatus />
          <ThemeToggle />
        </div>
        <div className="flex items-center gap-2">
          <div className="w-6 h-6 rounded-full bg-white/[0.06] flex items-center justify-center text-[10px] font-mono text-muted-foreground/40">
            J
          </div>
          <span className="text-[11px] text-muted-foreground/40">jterrazz</span>
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
