"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  IconLayoutDashboardFilled,
  IconEyeFilled,
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
import type { World, LimboAgent } from "@/lib/types";
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
  running: "bg-green-500", idle: "bg-amber-400", stopped: "bg-zinc-400/30", creating: "bg-blue-400",
};

function extractName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2 ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1) : id;
}

export function AppSidebar({ worlds, limboAgents, currentWorldId, loading, statusData }: AppSidebarProps) {
  const pathname = usePathname();
  const router = useRouter();
  const [showNewAgent, setShowNewAgent] = useState(false);
  const [newAgentName, setNewAgentName] = useState("");
  const [creating, setCreating] = useState(false);

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
        <div className="flex items-center justify-between px-2">
          <div className="group/logo flex items-center w-full">
            <Link href="/" className="flex items-center gap-1.5">
              <span className={`text-base font-heading transition-colors ${connectionStatus === "connected" ? "text-green-500" : "text-red-400"}`}>⬡</span>
              <span className="text-base tracking-[0.12em] font-heading text-foreground">spwn</span>
            </Link>
            <span className={`ml-auto text-[10px] font-mono uppercase tracking-wider opacity-0 group-hover/logo:opacity-100 transition-opacity ${connectionStatus === "connected" ? "text-green-500/60" : "text-red-400/60"}`}>
              {connectionStatus}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-1 px-1.5 mt-4">
          <ThemeToggle />
          <a href="https://spwn.sh/docs" target="_blank" className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors" aria-label="Docs">
            <IconBookFilled size={14} />
          </a>
          <a href="https://github.com/jterrazz/spwn" target="_blank" className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors" aria-label="GitHub">
            <IconBrandGithubFilled size={14} />
          </a>
          <a href="https://github.com/jterrazz/spwn/issues/new" target="_blank" className="w-8 h-8 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground transition-colors" aria-label="Feedback">
            <IconAlertTriangleFilled size={14} />
          </a>
          <div className="flex-1" />
          <button
            className="w-8 h-8 flex items-center justify-center rounded-full bg-foreground/[0.06] dark:bg-white/[0.08] backdrop-blur-md border border-foreground/[0.08] dark:border-white/[0.1] text-muted-foreground/40 hover:text-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.12),0_1px_2px_rgba(0,0,0,0.05)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.06),0_1px_2px_rgba(0,0,0,0.2)] transition-colors"
            onClick={() => window.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true, bubbles: true }))}
            aria-label="Search (⌘K)"
          >
            <IconSearch size={15} />
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
          <SidebarGroupLabel className="text-[10px] uppercase tracking-widest text-sidebar-foreground/30">Worlds</SidebarGroupLabel>
          {worlds.length > 0 ? (
            <div className="flex flex-wrap gap-1 px-0 pb-1">
              {worlds.map((world) => {
                const name = extractName(world.id);
                const isSelected = selectedWorld?.id === world.id;
                const statusColor = WORLD_DOT[world.status]?.replace("bg-", "text-") ?? "text-white/10";
                return (
                  <button
                    key={world.id}
                    onClick={() => router.push(`/world/${world.id}`)}
                    className={`flex items-center gap-2 rounded-full h-8 px-2 text-sm transition-all ${
                      isSelected
                        ? "bg-foreground/[0.06] dark:bg-white/[0.08] backdrop-blur-md border border-foreground/[0.08] dark:border-white/[0.1] text-foreground/80 shadow-[inset_0_1px_0_rgba(255,255,255,0.12),0_1px_2px_rgba(0,0,0,0.05)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.06),0_1px_2px_rgba(0,0,0,0.2)]"
                        : "border border-transparent text-muted-foreground/40 hover:text-muted-foreground/60 hover:bg-sidebar-accent"
                    }`}
                  >
                    <IconCircleFilled size={8} className={statusColor} />
                    <span>{name}</span>
                  </button>
                );
              })}
            </div>
          ) : (
            <p className="px-2 py-1.5 text-xs text-muted-foreground/25">No worlds running</p>
          )}
          {selectedWorld && (
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={pathname === `/world/${selectedWorld.id}`} onClick={() => router.push(`/world/${selectedWorld.id}`)}>
                  <IconEyeFilled size={16} />
                  <span>Overview</span>
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
      </SidebarFooter>
    </Sidebar>
  );
}
