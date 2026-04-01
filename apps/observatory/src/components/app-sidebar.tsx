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

interface Agent {
  name: string;
  tier: string;
  status: string;
}

interface World {
  id: string;
  config: string;
  agent: string;
  agents: Agent[];
  status: string;
}

interface DriftingAgent {
  name: string;
  layers: number;
}

interface AppSidebarProps {
  worlds: World[];
  driftingAgents: DriftingAgent[];
  currentWorldId?: string;
}

const STATUS_DOT: Record<string, string> = {
  running: "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]",
  idle: "bg-yellow-500 shadow-[0_0_6px_rgba(234,179,8,0.5)]",
  stopped: "bg-white/20",
  creating: "bg-blue-400 shadow-[0_0_6px_rgba(96,165,250,0.5)]",
};

const TIER_ICON: Record<string, string> = {
  governor: "♛",
  citizen: "◉",
  npc: "◌",
};

function extractName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2
    ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1)
    : id;
}

export function AppSidebar({ worlds, driftingAgents, currentWorldId }: AppSidebarProps) {
  const pathname = usePathname();
  const currentWorld = worlds.find((w) => w.id === currentWorldId);

  return (
    <Sidebar>
      <SidebarHeader className="p-4 space-y-3">
        <a href="/" className="flex items-center gap-2.5">
          <span className="text-sm tracking-[0.2em] font-heading text-foreground/90">
            ⬡ observatory
          </span>
        </a>

        {/* World selector dropdown */}
        {currentWorld && (
          <div className="glass-subtle px-3 py-2 flex items-center gap-2">
            <div className={`w-1.5 h-1.5 rounded-full shrink-0 ${STATUS_DOT[currentWorld.status]}`} />
            <span className="text-xs font-heading text-foreground/70 flex-1">
              {extractName(currentWorld.id)}
            </span>
            <span className="text-[10px] font-mono text-muted-foreground/30">
              ⌄
            </span>
          </div>
        )}
      </SidebarHeader>

      <SidebarSeparator />

      <SidebarContent>
        {/* Universe overview */}
        <SidebarGroup>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                isActive={pathname === "/"}
                onClick={() => window.location.href = "/"}
              >
                <span className="text-sm">◉</span>
                <span>Universe</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

        <SidebarSeparator />

        {/* Worlds */}
        <SidebarGroup>
          <SidebarGroupLabel className="text-[10px] uppercase tracking-widest text-muted-foreground/40">
            Worlds
          </SidebarGroupLabel>
          <SidebarMenu>
            {worlds.map((world) => (
              <Collapsible key={world.id} defaultOpen>
                <SidebarMenuItem>
                  <div className="flex w-full items-center">
                    <a
                      href={`/world/${world.id}`}
                      className={`flex flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-xs font-mono hover:bg-sidebar-accent transition-colors ${pathname === `/world/${world.id}` ? "bg-sidebar-accent text-foreground" : "text-muted-foreground/70"}`}
                    >
                      <div className={`w-1.5 h-1.5 rounded-full shrink-0 ${STATUS_DOT[world.status]}`} />
                      <span>{extractName(world.id)}</span>
                    </a>
                    <CollapsibleTrigger className="px-1.5 py-1 text-[10px] text-muted-foreground/40 hover:text-muted-foreground transition-colors">
                      ⌄
                    </CollapsibleTrigger>
                  </div>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      {world.agents.map((agent) => (
                        <SidebarMenuSubItem key={agent.name}>
                          <SidebarMenuSubButton
                            isActive={pathname === `/world/${world.id}/${agent.name}`}
                            onClick={() => window.location.href = `/world/${world.id}/${agent.name}`}
                          >
                            <span className="text-[10px] text-muted-foreground/50">
                              {TIER_ICON[agent.tier] ?? "◌"}
                            </span>
                            <span className="text-xs">{agent.name}</span>
                            <div className={`w-1 h-1 rounded-full ml-auto shrink-0 ${STATUS_DOT[agent.status]}`} />
                          </SidebarMenuSubButton>
                        </SidebarMenuSubItem>
                      ))}
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>
            ))}
          </SidebarMenu>
        </SidebarGroup>

        {/* Drifting agents */}
        {driftingAgents.length > 0 && (
          <>
            <SidebarSeparator />
            <SidebarGroup>
              <SidebarGroupLabel className="text-[10px] uppercase tracking-widest text-muted-foreground/40">
                Drifting
              </SidebarGroupLabel>
              <SidebarMenu>
                {driftingAgents.map((agent) => (
                  <SidebarMenuItem key={agent.name}>
                    <SidebarMenuButton className="text-xs text-muted-foreground/50">
                      <span className="text-[10px]">◌</span>
                      <span>{agent.name}</span>
                      <span className="ml-auto text-[10px] text-muted-foreground/30 font-mono">
                        {agent.layers}/6
                      </span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroup>
          </>
        )}
      </SidebarContent>

      <SidebarFooter className="p-3">
        <div className="flex items-center justify-between">
          <span className="text-[10px] font-mono text-muted-foreground/30">spwn vdev</span>
          <ThemeToggle />
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
