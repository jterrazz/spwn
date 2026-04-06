"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  IconAssemblyFilled,
  IconHomeFilled,
  IconHexagonFilled,
  IconUserFilled,
  IconSearch,
  IconBoltFilled,
  IconMessageFilled,
  IconMoonFilled,
  IconCircleFilled,
  IconBookFilled as IconKnowledgeFilled,
  IconSettingsFilled,
  IconBookFilled,
  IconBrandGithubFilled,
  IconAlertTriangleFilled,
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
  SidebarFooter,
  SidebarSeparator,
} from "@/components/ui/sidebar";

import { ThemeToggle } from "@/components/theme-toggle";
import { UpgradeBanner } from "@/components/upgrade-banner";
import { useVersion } from "@/hooks/use-version";
import { getWorldName, type World, type Team } from "@/lib/types";
import { WorldPlanet } from "@/components/world-planet";
import { apiGet, isGoApiAvailable, onConnectionStatusChange, getConnectionStatus, type ConnectionStatus } from "@/lib/api-client";

interface StatusData { worlds: number; agents: number; running: number; }
interface AppSidebarProps {
  worlds: World[];
  currentWorldId?: string;
  loading?: boolean;
  statusData?: StatusData | null;
}

const AGENT_ICON: Record<string, { icon: typeof IconBoltFilled; color: string; dim: boolean }> = {
  running:  { icon: IconBoltFilled,    color: "text-green-400",                dim: false },
  waiting:  { icon: IconMessageFilled, color: "text-amber-400 animate-pulse",  dim: false },
  sleeping: { icon: IconMoonFilled,    color: "text-purple-400",               dim: false },
  idle:     { icon: IconCircleFilled,  color: "text-amber-400/50",             dim: true },
  stopped:  { icon: IconCircleFilled,  color: "text-zinc-500/30",              dim: true },
};

const WORLD_DOT: Record<string, string> = {
  running: "bg-green-500", idle: "bg-amber-400", stopped: "bg-zinc-400/30", creating: "bg-white/80",
};

// Thin wrapper for legacy callers that only have an id. When a full World is available,
// prefer getWorldName(world) so a custom display name is respected.
function extractName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2 ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1) : id;
}

export function AppSidebar({ worlds, currentWorldId, loading, statusData }: AppSidebarProps) {
  const pathname = usePathname();
  const router = useRouter();
  const [worldsExpanded, setWorldsExpanded] = useState(false);
  const [teams, setTeams] = useState<Team[]>([]);

  const { version } = useVersion();

  // Fetch teams to group agents in the selected world by team
  useEffect(() => {
    apiGet<Team[]>("/api/teams").then((t) => setTeams(t ?? [])).catch(() => {});
  }, []);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>(getConnectionStatus());

  useEffect(() => {
    const unsub = onConnectionStatusChange(setConnectionStatus);
    const check = async () => {
      const goUp = await isGoApiAvailable();
      setConnectionStatus(goUp ? "connected" : "disconnected");
    };
    check();
    const interval = setInterval(check, 10000);
    return () => { clearInterval(interval); unsub(); };
  }, []);

  // Always show a world context (first world as default), but only highlight when on a world page
  const activeWorldId = currentWorldId || worlds.find(w => pathname.startsWith(`/world/${w.id}`))?.id;
  const selectedWorldId = activeWorldId || worlds[0]?.id;
  const selectedWorld = selectedWorldId ? worlds.find(w => w.id === selectedWorldId) : undefined;

  return (
    <Sidebar>
      {/* ── Header ── */}
      <SidebarHeader className="gap-0 pt-4 pb-4">
        <div className="flex items-center justify-between gap-2 px-2">
          <div className="group/logo flex items-center flex-1 min-w-0">
            <Link href="/" className="flex items-center gap-1.5">
              <span className={`text-base font-heading transition-colors ${connectionStatus === "connected" ? "text-green-500" : "text-red-400"}`}>⬡</span>
              <span className="text-base tracking-[0.12em] font-heading text-foreground">spwn</span>
            </Link>
            <span className={`ml-auto text-[10px] font-mono uppercase tracking-wider opacity-0 group-hover/logo:opacity-100 transition-opacity ${connectionStatus === "connected" ? "text-green-500/60" : "text-red-400/60"}`}>
              {connectionStatus}
            </span>
          </div>
          <button
            className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors shrink-0"
            onClick={() => window.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true, bubbles: true }))}
            aria-label="Search (⌘K)"
          >
            <IconSearch size={16} stroke={2.5} />
          </button>
        </div>
      </SidebarHeader>

      <SidebarContent>

        {/* ── Global ── */}
        <SidebarGroup className="pt-0">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton isActive={pathname === "/architect"} onClick={() => router.push("/architect")}>
                <IconHexagonFilled size={16} />
                <span>Architect</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton isActive={pathname === "/providers"} onClick={() => router.push("/providers")}>
                <IconSettingsFilled size={16} />
                <span>Settings</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

        {/* ── Universe ── */}
        <SidebarGroup>
          <SidebarGroupLabel className="text-[10px] uppercase tracking-widest text-sidebar-foreground/30">Universe</SidebarGroupLabel>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton isActive={pathname === "/"} onClick={() => router.push("/")}>
                <IconAssemblyFilled size={16} />
                <span>Worlds</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton isActive={pathname === "/agents" || pathname.startsWith("/agents/")} onClick={() => router.push("/agents")}>
                <IconUserFilled size={16} />
                <span>Agents</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

        {/* ── Worlds ── */}
        <SidebarGroup>
          {worlds.length > 0 && selectedWorld ? (
            <div className="px-1 space-y-1.5">
              {/* Hero: current world */}
              <button
                onClick={() => router.push(`/world/${selectedWorld.id}`)}
                className="w-full flex items-center gap-3 px-2 py-2 rounded-md bg-sidebar-accent/50 hover:bg-sidebar-accent transition-colors text-left"
              >
                <WorldPlanet world={selectedWorld} size="md" />
                <span className="min-w-0 flex-1">
                  <span className="block text-sm font-medium text-foreground truncate">
                    {getWorldName(selectedWorld)}
                  </span>
                  <span className="block text-[10px] uppercase tracking-widest text-muted-foreground/40">
                    {selectedWorld.status} · {selectedWorld.agents.length} agent
                    {selectedWorld.agents.length === 1 ? "" : "s"}
                  </span>
                </span>
              </button>

              {/* Quick-switch: other worlds. Condensed to one line with "+N" overflow; click to expand into a masonry wrap. */}
              {(() => {
                const others = worlds.filter((w) => w.id !== selectedWorld.id);
                if (others.length === 0) return null;

                // Character-budget heuristic: fit as many pills on one line as reasonably possible.
                // Each pill costs name.length + 3 (planet + padding/gap). Adjust budget if sidebar width changes.
                const INLINE_CHAR_BUDGET = 22;
                const inlineItems: typeof others = [];
                const overflowItems: typeof others = [];
                let budget = INLINE_CHAR_BUDGET;
                for (const w of others) {
                  const cost = getWorldName(w).length + 3;
                  if (!worldsExpanded && budget - cost < 0 && inlineItems.length > 0) {
                    overflowItems.push(w);
                  } else {
                    inlineItems.push(w);
                    budget -= cost;
                  }
                }
                if (worldsExpanded) {
                  overflowItems.length = 0; // all visible when expanded
                }

                const renderPill = (world: (typeof others)[number]) => (
                  <button
                    key={world.id}
                    onClick={() => router.push(`/world/${world.id}`)}
                    className="group/switch shrink-0 flex items-center gap-1.5 h-6 pl-1 pr-2 rounded-md text-xs text-muted-foreground/50 hover:text-foreground hover:bg-sidebar-accent/40 transition-colors"
                  >
                    <WorldPlanet world={world} size="sm" className="opacity-80 group-hover/switch:opacity-100 transition-opacity" />
                    <span>{getWorldName(world)}</span>
                  </button>
                );

                const toggleBtn = (label: string) => (
                  <button
                    onClick={() => setWorldsExpanded((v) => !v)}
                    className="shrink-0 flex items-center justify-center h-6 min-w-6 px-1.5 rounded-md text-[11px] font-medium text-muted-foreground/40 hover:text-foreground hover:bg-sidebar-accent/40 transition-colors"
                    aria-label={worldsExpanded ? "Collapse worlds" : "Show all worlds"}
                  >
                    {label}
                  </button>
                );

                return (
                  <div
                    className={`${
                      worldsExpanded ? "flex flex-wrap" : "flex items-center overflow-hidden"
                    } gap-1 -mx-1 px-1`}
                  >
                    {inlineItems.map(renderPill)}
                    {!worldsExpanded && overflowItems.length > 0 && toggleBtn(`+${overflowItems.length}`)}
                    {worldsExpanded && others.length > 0 && toggleBtn("Hide")}
                  </div>
                );
              })()}
            </div>
          ) : (
            <p className="px-2 py-1.5 text-xs text-muted-foreground/25">No worlds running</p>
          )}
          {selectedWorld && (
            <SidebarMenu className="mt-3">
              <SidebarMenuItem>
                <SidebarMenuButton isActive={pathname === `/world/${selectedWorld.id}`} onClick={() => router.push(`/world/${selectedWorld.id}`)}>
                  <IconHomeFilled size={16} />
                  <span>Home</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={pathname === `/world/${selectedWorld.id}/knowledge`} onClick={() => router.push(`/world/${selectedWorld.id}/knowledge`)}>
                  <IconKnowledgeFilled size={16} />
                  <span>Knowledge</span>
                </SidebarMenuButton>
              </SidebarMenuItem>

              {(() => {
                // Build agent-name → team lookup from teams data
                const agentTeamMap = new Map<string, Team>();
                for (const t of teams) {
                  for (const m of t.members ?? []) {
                    agentTeamMap.set(m, t);
                  }
                }

                // Group world agents by team
                const grouped = new Map<string, { team: Team | null; agents: typeof selectedWorld.agents }>();
                const soloAgents: typeof selectedWorld.agents = [];

                for (const agent of selectedWorld.agents) {
                  const team = agentTeamMap.get(agent.name);
                  if (team) {
                    const group = grouped.get(team.slug) ?? { team, agents: [] };
                    group.agents.push(agent);
                    grouped.set(team.slug, group);
                  } else {
                    soloAgents.push(agent);
                  }
                }

                const renderAgent = (agent: typeof selectedWorld.agents[number]) => {
                  const s = AGENT_ICON[agent.status] ?? AGENT_ICON.stopped;
                  const StatusIcon = s.icon;
                  return (
                    <SidebarMenuItem key={agent.name}>
                      <SidebarMenuButton
                        isActive={pathname === `/world/${selectedWorld.id}/${agent.name}`}
                        onClick={() => router.push(`/world/${selectedWorld.id}/${agent.name}`)}
                      >
                        <span className="w-[20px] h-[20px] -mx-[2px] -translate-x-[0.5px] rounded-full flex items-center justify-center shrink-0 bg-white/[0.15]">
                          <StatusIcon className={`!size-[12px] ${s.color}`} />
                        </span>
                        <span className={s.dim ? "opacity-50" : ""}>{agent.name}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                };

                if (selectedWorld.agents.length === 0) {
                  return <p className="px-2 py-1.5 text-xs text-muted-foreground/25">No agents deployed</p>;
                }

                // If no teams, render flat list (no headers)
                if (grouped.size === 0) {
                  return soloAgents.map(renderAgent);
                }

                // Render grouped: team sections + solo at bottom
                return (
                  <>
                    {Array.from(grouped.values()).map(({ team, agents }) => (
                      <div key={team!.slug} className="mt-1">
                        <p className="px-2 py-1 text-[9px] uppercase tracking-[0.12em] text-sidebar-foreground/25 flex items-center gap-1.5">
                          {team!.icon && <span className="text-[10px]">{team!.icon}</span>}
                          <span style={team!.color ? { color: team!.color } : undefined}>{team!.name}</span>
                        </p>
                        {agents.map(renderAgent)}
                      </div>
                    ))}
                    {soloAgents.length > 0 && (
                      <div className="mt-1">
                        {grouped.size > 0 && (
                          <p className="px-2 py-1 text-[9px] uppercase tracking-[0.12em] text-sidebar-foreground/20">No team</p>
                        )}
                        {soloAgents.map(renderAgent)}
                      </div>
                    )}
                  </>
                );
              })()}
            </SidebarMenu>
          )}
        </SidebarGroup>

      </SidebarContent>

      {/* ── Footer ── */}
      <SidebarFooter>
        {version?.updateAvailable && <UpgradeBanner version={version} />}
        <div className="flex items-center gap-1 px-1.5 pb-1">
          <a href="https://spwn.sh/docs" target="_blank" className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors" aria-label="Docs">
            <IconBookFilled size={15} />
          </a>
          <a href="https://github.com/jterrazz/spwn" target="_blank" className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors" aria-label="GitHub">
            <IconBrandGithubFilled size={15} />
          </a>
          <a href="https://github.com/jterrazz/spwn/issues/new" target="_blank" className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors" aria-label="Feedback">
            <IconAlertTriangleFilled size={15} />
          </a>
          <div className="flex-1" />
          <ThemeToggle />
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
