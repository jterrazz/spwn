"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  IconLayoutDashboardFilled,
  IconHomeFilled,
  IconHexagonFilled,
  IconSearch,
  IconBoltFilled,
  IconMessageFilled,
  IconMoonFilled,
  IconCircleFilled,
  IconGhostFilled,
  IconBookFilled as IconKnowledgeFilled,
  IconPlus,
  IconCheck,
  IconX,
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
import { getWorldName, type World, type LimboAgent } from "@/lib/types";
import { apiAction, apiDelete, isGoApiAvailable, onConnectionStatusChange, getConnectionStatus, type ConnectionStatus } from "@/lib/api-client";

interface StatusData { worlds: number; agents: number; running: number; }
interface AppSidebarProps {
  worlds: World[];
  limboAgents: LimboAgent[];
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

function hashHue(id: string): number {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0;
  return h % 360;
}

export function AppSidebar({ worlds, limboAgents, currentWorldId, loading, statusData }: AppSidebarProps) {
  const pathname = usePathname();
  const router = useRouter();
  const [showNewAgent, setShowNewAgent] = useState(false);
  const [newAgentName, setNewAgentName] = useState("");
  const [creating, setCreating] = useState(false);
  const [worldsExpanded, setWorldsExpanded] = useState(false);

  const newAgentInputRef = useRef<HTMLInputElement>(null);
  const { version } = useVersion();
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

  useEffect(() => { if (showNewAgent) newAgentInputRef.current?.focus(); }, [showNewAgent]);

  const handleCreateAgent = async () => {
    if (!newAgentName.trim() || creating) return;
    setCreating(true);
    try {
      const result = await apiAction("/api/agents", { name: newAgentName.trim() });
      if (result.ok) { setNewAgentName(""); setShowNewAgent(false); router.push(`/agents/${newAgentName.trim()}`); }
    } catch { /* */ } finally { setCreating(false); }
  };

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

        {/* ── Navigation ── */}
        <SidebarGroup className="pt-0">
          <SidebarGroupLabel className="text-[10px] uppercase tracking-widest text-sidebar-foreground/30">Universe</SidebarGroupLabel>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton isActive={pathname === "/"} onClick={() => router.push("/")}>
                <IconLayoutDashboardFilled size={16} />
                <span>Dashboard</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
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

        {/* ── Worlds ── */}
        <SidebarGroup>
          {worlds.length > 0 && selectedWorld ? (
            <div className="px-1 space-y-1.5">
              {/* Hero: current world */}
              {(() => {
                const hue = hashHue(selectedWorld.id);
                const isActive =
                  selectedWorld.status === "running" || selectedWorld.status === "creating";
                const sat = isActive ? 70 : 15;
                return (
                  <button
                    onClick={() => router.push(`/world/${selectedWorld.id}`)}
                    className="w-full flex items-center gap-3 px-2 py-2 rounded-md bg-sidebar-accent/50 hover:bg-sidebar-accent transition-colors text-left"
                  >
                    <span className="relative shrink-0 w-7 h-7 flex items-center justify-center">
                      <span
                        className="absolute inset-[-3px] rounded-full blur-md pointer-events-none"
                        style={{ background: `hsl(${hue} ${sat}% 60% / 0.45)` }}
                      />
                      <span
                        className="relative block w-6 h-6 rounded-full"
                        style={{
                          background: `radial-gradient(circle at 32% 30%, hsl(${hue} ${sat}% 78%), hsl(${hue} ${sat}% 48%) 55%, hsl(${hue} ${sat}% 22%))`,
                          boxShadow: `0 0 0 1.5px hsl(${hue} ${sat}% 72% / 0.9), inset 0 -1px 2px rgba(0,0,0,0.35)`,
                        }}
                      />
                    </span>
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
                );
              })()}

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

                const renderPill = (world: (typeof others)[number]) => {
                  const hue = hashHue(world.id);
                  const isActive =
                    world.status === "running" || world.status === "creating";
                  const sat = isActive ? 70 : 15;
                  return (
                    <button
                      key={world.id}
                      onClick={() => router.push(`/world/${world.id}`)}
                      className="group/switch shrink-0 flex items-center gap-1.5 h-6 pl-1 pr-2 rounded-md text-xs text-muted-foreground/50 hover:text-foreground hover:bg-sidebar-accent/40 transition-colors"
                    >
                      <span
                        className="block w-3 h-3 rounded-full opacity-80 group-hover/switch:opacity-100 transition-opacity"
                        style={{
                          background: `radial-gradient(circle at 32% 30%, hsl(${hue} ${sat}% 78%), hsl(${hue} ${sat}% 48%) 55%, hsl(${hue} ${sat}% 22%))`,
                          boxShadow: "inset 0 -1px 1px rgba(0,0,0,0.35)",
                        }}
                      />
                      <span>{getWorldName(world)}</span>
                    </button>
                  );
                };

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

              {selectedWorld.agents.map((agent) => {
                const s = AGENT_ICON[agent.status] ?? AGENT_ICON.stopped;
                const StatusIcon = s.icon;
                return (
                  <SidebarMenuItem key={agent.name}>
                    <SidebarMenuButton
                      isActive={pathname === `/world/${selectedWorld.id}/${agent.name}`}
                      onClick={() => router.push(`/world/${selectedWorld.id}/${agent.name}`)}
                    >
                      <StatusIcon size={16} className={s.color} />
                      <span className={s.dim ? "opacity-50" : ""}>{agent.name}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}

              {selectedWorld.agents.length === 0 && (
                <p className="px-2 py-1.5 text-xs text-muted-foreground/25">No agents deployed</p>
              )}
            </SidebarMenu>
          )}
        </SidebarGroup>

        {/* ── Limbo ── */}
        <SidebarGroup>
          <SidebarGroupLabel className="text-[10px] uppercase tracking-widest text-sidebar-foreground/30">
            Limbo
            {limboAgents.length > 0 && (
              <span className="ml-auto text-sidebar-foreground/20">{limboAgents.length}</span>
            )}
          </SidebarGroupLabel>
          <SidebarMenu>
            {limboAgents.map((agent) => (
              <SidebarMenuItem key={agent.name} className="group/agent">
                <SidebarMenuButton
                  isActive={pathname === `/agents/${agent.name}`}
                  onClick={() => router.push(`/agents/${agent.name}`)}
                >
                  <IconGhostFilled size={16} className="opacity-30" />
                  <span className="text-muted-foreground/40">{agent.name}</span>
                  <span
                    role="button"
                    tabIndex={0}
                    className="ml-auto hidden group-hover/agent:flex items-center justify-center w-5 h-5 rounded-md hover:bg-destructive/10 text-muted-foreground/20 hover:text-destructive transition-colors cursor-pointer"
                    title={`Delete ${agent.name}`}
                    onClick={async (e) => {
                      e.stopPropagation();
                      if (!confirm(`Delete agent "${agent.name}"? This cannot be undone.`)) return;
                      try { await apiDelete(`/api/agents/${agent.name}`); router.refresh(); } catch { /* */ }
                    }}
                    onKeyDown={(e) => { if (e.key === "Enter") e.currentTarget.click(); }}
                  >
                    <IconX size={12} />
                  </span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
            {limboAgents.length === 0 && (
              <p className="px-2 py-1.5 text-xs text-muted-foreground/25">No agents in limbo</p>
            )}
            <SidebarMenuItem>
              {showNewAgent ? (
                <div className="flex items-center gap-1.5 px-2 py-1">
                  <input
                    ref={newAgentInputRef}
                    value={newAgentName}
                    onChange={(e) => setNewAgentName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleCreateAgent();
                      if (e.key === "Escape") { setShowNewAgent(false); setNewAgentName(""); }
                    }}
                    placeholder="agent name..."
                    className="flex-1 min-w-0 bg-transparent text-xs text-foreground/60 placeholder:text-muted-foreground/25 border-b border-sidebar-border focus:outline-none focus:border-sidebar-ring"
                    disabled={creating}
                  />
                  <button onClick={handleCreateAgent} disabled={!newAgentName.trim() || creating} className="p-0.5 text-green-500/60 hover:text-green-500 disabled:opacity-30">
                    {creating ? <div className="w-3.5 h-3.5 border-2 border-foreground/20 border-t-foreground/60 rounded-full animate-spin" /> : <IconCheck size={14} />}
                  </button>
                  <button onClick={() => { setShowNewAgent(false); setNewAgentName(""); }} className="p-0.5 text-muted-foreground/25 hover:text-muted-foreground/50">
                    <IconX size={14} />
                  </button>
                </div>
              ) : (
                <SidebarMenuButton onClick={() => setShowNewAgent(true)} className="text-muted-foreground/30">
                  <IconPlus size={16} />
                  <span>New Agent</span>
                </SidebarMenuButton>
              )}
            </SidebarMenuItem>
          </SidebarMenu>
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
