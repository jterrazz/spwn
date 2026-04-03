"use client";

import { useState, useRef, useEffect } from "react";
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
  IconBook2,
  IconPlus,
  IconCheck,
  IconX,
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
import { UpgradeBanner } from "@/components/upgrade-banner";
import { useVersion } from "@/hooks/use-version";
import type { World, LimboAgent } from "@/lib/types";
import { apiAction } from "@/lib/api-client";

interface StatusData {
  worlds: number;
  agents: number;
  running: number;
}

interface AppSidebarProps {
  worlds: World[];
  limboAgents: LimboAgent[];
  currentWorldId?: string;
  loading?: boolean;
  statusData?: StatusData | null;
}

// Agent status → Tabler filled icon + color
const AGENT_ICON: Record<string, { icon: typeof IconBoltFilled; color: string; dim: boolean }> = {
  running:  { icon: IconBoltFilled, color: "text-green-400", dim: false },
  waiting:  { icon: IconMessageFilled, color: "text-amber-400 animate-pulse", dim: false },
  sleeping: { icon: IconMoonFilled, color: "text-purple-400", dim: false },
  idle:     { icon: IconCircleFilled, color: "text-amber-400/50", dim: true },
  stopped:  { icon: IconCircleFilled, color: "text-zinc-500/30", dim: true },
};

const WORLD_DOT: Record<string, string> = {
  running: "text-green-500",
  idle: "text-amber-400",
  stopped: "text-zinc-400/30",
  creating: "text-blue-400",
};

function extractName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2
    ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1)
    : id;
}

export function AppSidebar({ worlds, limboAgents, currentWorldId, loading, statusData }: AppSidebarProps) {
  const pathname = usePathname();
  const [showNewAgent, setShowNewAgent] = useState(false);
  const [newAgentName, setNewAgentName] = useState("");
  const [creating, setCreating] = useState(false);
  const newAgentInputRef = useRef<HTMLInputElement>(null);
  const { version } = useVersion();

  useEffect(() => {
    if (showNewAgent) {
      newAgentInputRef.current?.focus();
    }
  }, [showNewAgent]);

  const handleCreateAgent = async () => {
    if (!newAgentName.trim() || creating) return;
    setCreating(true);
    try {
      const result = await apiAction("/api/agents", { name: newAgentName.trim() }, "/api/agents/create");
      if (result.ok) {
        const name = newAgentName.trim();
        setNewAgentName("");
        setShowNewAgent(false);
        // Navigate to the new agent's profile page
        window.location.href = `/agents/${name}`;
      }
    } catch {
      // silently fail
    } finally {
      setCreating(false);
    }
  };

  return (
    <Sidebar>
      {/* ── Spacer: pushes content below macOS traffic lights in Tauri native app ── */}
      <div className="h-[52px] w-full shrink-0 tauri-only" />
      <SidebarHeader className="px-4 pb-2 -mt-1">
        <a href="/" className="flex items-center gap-2">
          <span className="text-sm tracking-[0.15em] font-heading text-foreground/90">
            ⬡ spwn
          </span>
        </a>
        <p className="text-[10px] font-mono text-muted-foreground/30 mt-1 leading-relaxed">
          <span>orbstack</span>
          <span className="text-muted-foreground/15"> · </span>
          <span>docker</span>
          <br />
          <span>{statusData?.worlds ?? worlds.length} worlds</span>
          <span className="text-muted-foreground/15"> · </span>
          <span>{statusData?.agents ?? limboAgents.length + worlds.reduce((n, w) => n + w.agents.length, 0)} agents</span>
        </p>
        <button
          className="mt-3 w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-[11px] text-muted-foreground/30 hover:text-muted-foreground/50 hover:bg-white/[0.03] transition-colors"
          onClick={() => {
            // Trigger Cmd+K to open command palette
            window.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true, bubbles: true }));
          }}
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
                isActive={pathname === "/blueprint"}
                onClick={() => window.location.href = "/blueprint"}
              >
                <IconBook2 size={16} className="opacity-50" />
                <span>Blueprint</span>
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
            {worlds.length > 0 && (
              <span className="ml-1.5 text-[9px] font-mono text-muted-foreground/20">{worlds.length}</span>
            )}
          </SidebarGroupLabel>
          <SidebarMenu>
            {loading && worlds.length === 0 && (
              <>
                {[1, 2].map((i) => (
                  <SidebarMenuItem key={i}>
                    <div className="flex items-center gap-2.5 px-2 py-1.5">
                      <div className="w-2 h-2 rounded-full bg-white/[0.06] animate-pulse" />
                      <div className="h-3 w-20 rounded bg-white/[0.06] animate-pulse" />
                    </div>
                  </SidebarMenuItem>
                ))}
              </>
            )}
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
            {limboAgents.length > 0 && (
              <span className="ml-1.5 text-[9px] font-mono text-muted-foreground/20">{limboAgents.length}</span>
            )}
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
                <span className="text-[10px] text-muted-foreground/20 px-2 py-1">No agents yet</span>
              </SidebarMenuItem>
            )}
            <SidebarMenuItem>
              {showNewAgent ? (
                <div className="flex items-center gap-1 px-2 py-1">
                  <input
                    ref={newAgentInputRef}
                    value={newAgentName}
                    onChange={(e) => setNewAgentName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleCreateAgent();
                      if (e.key === "Escape") { setShowNewAgent(false); setNewAgentName(""); }
                    }}
                    placeholder="agent name..."
                    className="flex-1 min-w-0 bg-transparent text-[11px] text-foreground/70 placeholder:text-muted-foreground/25 border-b border-white/[0.08] pb-0.5 focus:outline-none focus:border-white/[0.2]"
                    disabled={creating}
                  />
                  <button
                    onClick={handleCreateAgent}
                    disabled={!newAgentName.trim() || creating}
                    className="p-0.5 text-green-400/60 hover:text-green-400 transition-colors disabled:opacity-30"
                  >
                    {creating ? (
                      <div className="w-3 h-3 border border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                    ) : (
                      <IconCheck size={12} />
                    )}
                  </button>
                  <button
                    onClick={() => { setShowNewAgent(false); setNewAgentName(""); }}
                    className="p-0.5 text-muted-foreground/30 hover:text-muted-foreground/60 transition-colors"
                  >
                    <IconX size={12} />
                  </button>
                </div>
              ) : (
                <SidebarMenuButton
                  className="text-[11px] text-muted-foreground/25 hover:text-muted-foreground/50"
                  onClick={() => setShowNewAgent(true)}
                >
                  <IconPlus size={12} className="opacity-30 shrink-0" />
                  <span>New Agent</span>
                </SidebarMenuButton>
              )}
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

      </SidebarContent>

      {/* ── Footer ── */}
      <SidebarFooter className="px-4 py-3 space-y-3">
        {version?.updateAvailable && <UpgradeBanner version={version} />}
        <div className="flex items-center justify-between">
          <LiveStatus />
          <ThemeToggle />
        </div>
        <div className="flex items-center gap-3 text-[10px] font-mono text-muted-foreground/25">
          <a href="https://spwn.sh/docs" target="_blank" rel="noopener noreferrer" className="hover:text-muted-foreground/50 transition-colors">
            Docs
          </a>
          <span className="text-muted-foreground/10">·</span>
          <a href="https://github.com/jterrazz/spwn" target="_blank" rel="noopener noreferrer" className="hover:text-muted-foreground/50 transition-colors">
            GitHub
          </a>
          <span className="text-muted-foreground/10">·</span>
          <a href="https://github.com/jterrazz/spwn/issues/new" target="_blank" rel="noopener noreferrer" className="hover:text-muted-foreground/50 transition-colors">
            Report Bug
          </a>
          <span className="ml-auto flex items-center gap-1.5 text-muted-foreground/15">
            {version && (
              <span className={`inline-block w-1.5 h-1.5 rounded-full ${version.updateAvailable ? "bg-amber-400" : "bg-green-500"}`} />
            )}
            v{version?.current ?? "…"}
            {version?.updateAvailable && (
              <span className="text-amber-400/60">⬆</span>
            )}
          </span>
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
