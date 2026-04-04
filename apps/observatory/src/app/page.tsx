"use client";

import { useState, useEffect, useMemo, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Planet } from "@/components/planet";
import { AVAILABLE_CONFIGS } from "@/lib/types";
import type { World } from "@/lib/types";
import { IconPlus, IconRocket, IconX, IconPlanet, IconTrash, IconAlertTriangle, IconUser, IconBulb, IconWorld, IconCheck, IconArrowRight, IconSparkles, IconActivity, IconBoltFilled, IconMoonFilled, IconCircleFilled, IconMessageFilled } from "@tabler/icons-react";
import { Planet as PlanetGlobe } from "@/components/planet";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiAction, apiDelete, goApiUrl } from "@/lib/api-client";
import { useKeyboardShortcuts } from "@/hooks/use-keyboard-shortcuts";
import { RecentActivity } from "@/components/recent-activity";
import { useRefetch } from "@/components/app-shell";
import { usePageTitle } from "@/hooks/use-page-title";

interface AgentListItem {
  name: string;
  path: string;
  layers: Record<string, string[]>;
}

export default function UniverseMapPage() {
  const [worlds, setWorlds] = useState<World[]>([]);
  const [agents, setAgents] = useState<AgentListItem[]>([]);
  const [selected, setSelected] = useState<number | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const planetRefs = useRef<(HTMLDivElement | null)[]>([]);
  const [showSpawn, setShowSpawn] = useState(false);
  const [showDestroyAll, setShowDestroyAll] = useState(false);
  const [destroyingAll, setDestroyingAll] = useState(false);
  const [loading, setLoading] = useState(true);
  const [agentsLoading, setAgentsLoading] = useState(true);
  const router = useRouter();
  const refetchSidebar = useRefetch();
  usePageTitle("Dashboard");

  const fetchWorlds = () => {
    apiGet<World[]>("/api/universes")
      .then((data) => {
        setWorlds(data ?? []);
        setLoading(false);
      })
      .catch(() => {
        setWorlds([]);
        setLoading(false);
      });
  };

  const fetchAgents = () => {
    apiGet<AgentListItem[]>("/api/agents")
      .then((data) => {
        setAgents(data ?? []);
        setAgentsLoading(false);
      })
      .catch(() => {
        setAgents([]);
        setAgentsLoading(false);
      });
  };

  useEffect(() => {
    fetchWorlds();
    fetchAgents();
    const interval = setInterval(fetchWorlds, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (showSpawn) return;
      if (worlds.length === 0) return;
      if (e.key === "ArrowRight" || e.key === "d") {
        setSelected((s) => s === null ? 0 : (s + 1) % worlds.length);
      } else if (e.key === "ArrowLeft" || e.key === "a") {
        setSelected((s) => s === null ? worlds.length - 1 : (s - 1 + worlds.length) % worlds.length);
      } else if (e.key === "Enter" && selected !== null) {
        router.push(`/world/${worlds[selected].id}`);
      } else if (e.key === "Escape" && selected !== null) {
        setSelected(null);
      }
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [worlds, selected, router, showSpawn]);

  // ── Planet centering + drag system ──
  // Uses offsetLeft (static layout position, unaffected by transform) to avoid circular deps.
  // Single `tx` state = final translateX. Drag adds delta on top during gesture.
  const panelRef = useRef<HTMLDivElement>(null);
  const [, forceRender] = useState(0);
  const [tx, setTx] = useState(0);
  const [dragDelta, setDragDelta] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const dragRef = useRef({ startX: 0, startTx: 0, moved: false });
  const wasDragging = useRef(false);
  const lastFocusIdx = useRef<number | null>(null);

  // Get the visible width of the viewport area (minus panel if showing)
  const getVisibleWidth = useCallback((withPanel: boolean) => {
    const parent = scrollRef.current?.parentElement;
    if (!parent) return 800;
    if (!withPanel) return parent.clientWidth;
    const panelW = panelRef.current?.offsetWidth ?? 380;
    return parent.clientWidth - panelW - 24; // 24px gap between planet and panel
  }, []);

  // Get the static center X of a planet (its layout position, not affected by transform)
  const getPlanetCenter = useCallback((idx: number) => {
    const el = planetRefs.current[idx];
    if (!el) return 0;
    return el.offsetLeft + el.offsetWidth / 2;
  }, []);

  // Compute translateX to center a planet (or group) in the visible area
  const centerOn = useCallback((idx: number | null, withPanel: boolean) => {
    const vw = getVisibleWidth(withPanel);
    const target = vw / 2;

    if (idx !== null) {
      return target - getPlanetCenter(idx);
    }

    // Center the group of real planets
    const count = planetRefs.current.filter(Boolean).length;
    if (count === 0) return 0;
    const first = getPlanetCenter(0);
    const last = getPlanetCenter(Math.min(count - 1, planetRefs.current.length - 1));
    return target - (first + last) / 2;
  }, [getVisibleWidth, getPlanetCenter]);

  // Recenter when selection changes (double-RAF to ensure panel is mounted and measured)
  useEffect(() => {
    if (selected !== null) lastFocusIdx.current = selected;
    const focusIdx = selected ?? lastFocusIdx.current;

    // Double rAF: first lets React render the panel, second measures it
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        setTx(centerOn(focusIdx, selected !== null));
        setDragDelta(0);
      });
    });
  }, [selected, worlds.length, centerOn]);

  // Drag handlers
  const onDragStart = useCallback((x: number) => {
    dragRef.current = { startX: x, startTx: 0, moved: false };
    setIsDragging(true);
  }, []);

  const onDragMove = useCallback((x: number) => {
    if (!isDragging) return;
    const dx = x - dragRef.current.startX;
    if (Math.abs(dx) > 3) dragRef.current.moved = true;
    setDragDelta(dx);
  }, [isDragging]);

  const onDragEnd = useCallback(() => {
    setIsDragging(false);
    if (!dragRef.current.moved) { setDragDelta(0); return; }

    // Snap to nearest planet
    const vw = getVisibleWidth(selected !== null);
    const target = vw / 2;
    const currentTx = tx + dragDelta;

    let bestIdx = 0;
    let bestDist = Infinity;
    for (let i = 0; i < worlds.length; i++) {
      const screenX = getPlanetCenter(i) + currentTx;
      const dist = Math.abs(screenX - target);
      if (dist < bestDist) { bestDist = dist; bestIdx = i; }
    }

    setDragDelta(0);
    setTx(target - getPlanetCenter(bestIdx));
  }, [tx, dragDelta, selected, worlds.length, getVisibleWidth, getPlanetCenter]);

  // Global pointer listeners
  useEffect(() => {
    if (!isDragging) return;
    const move = (e: MouseEvent) => onDragMove(e.clientX);
    const up = () => onDragEnd();
    const tmove = (e: TouchEvent) => onDragMove(e.touches[0].clientX);
    const tend = () => onDragEnd();
    window.addEventListener("mousemove", move);
    window.addEventListener("mouseup", up);
    window.addEventListener("touchmove", tmove, { passive: true });
    window.addEventListener("touchend", tend);
    return () => {
      window.removeEventListener("mousemove", move);
      window.removeEventListener("mouseup", up);
      window.removeEventListener("touchmove", tmove);
      window.removeEventListener("touchend", tend);
    };
  }, [isDragging, onDragMove, onDragEnd]);

  const totalTx = tx + dragDelta;
  useEffect(() => { wasDragging.current = dragRef.current.moved; }, [isDragging]);

  // Recenter on window resize
  useEffect(() => {
    const onResize = () => {
      const focusIdx = selected ?? lastFocusIdx.current;
      setTx(centerOn(focusIdx, selected !== null));
    };
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, [selected, centerOn]);

  // Global keyboard shortcuts
  useKeyboardShortcuts({
    onSpawnWorld: () => setShowSpawn(true),
    onEscape: () => setShowSpawn(false),
  });

  const handleDestroyAll = async () => {
    setDestroyingAll(true);
    try {
      // Destroy each world sequentially (Go API uses DELETE method)
      for (const world of worlds) {
        await apiDelete(`/api/worlds/${world.id}`);
      }
      // Immediately refetch
      fetchWorlds();
      refetchSidebar();
      setShowDestroyAll(false);
    } catch {
      // ignore errors — worlds may already be gone
    } finally {
      setDestroyingAll(false);
      setShowDestroyAll(false);
    }
  };

  const handleSpawnComplete = () => {
    // Immediately refetch after spawn
    fetchWorlds();
    refetchSidebar();
  };

  const extractName = (id: string) => {
    const parts = id.split("-");
    return parts.length >= 2 ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1) : id;
  };

  const STATUS_DOT: Record<string, string> = {
    running: "bg-green-500", idle: "bg-amber-400", stopped: "bg-zinc-500/30", creating: "bg-blue-400",
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-heading tracking-wide text-foreground/90">Dashboard</h1>
          <p className="text-xs text-muted-foreground/30 mt-0.5">
            Orchestrate AI agents across your projects, persist their minds, scale at will — your AI matrix.
          </p>
        </div>
        <button
          onClick={() => setShowSpawn(true)}
          className="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs bg-primary/10 text-primary hover:bg-primary/20 border border-primary/20 transition-all"
        >
          <IconPlus size={14} />
          Spawn World
        </button>
      </div>

      {/* ── Quick Stats Bar ── */}
      {!loading && worlds.length > 0 && (() => {
        const totalAgents = worlds.reduce((n, w) => n + w.agents.length, 0);
        const runningWorlds = worlds.filter(w => w.status === "running" || w.status === "idle").length;
        const runningAgents = worlds.reduce((n, w) => n + w.agents.filter(a => a.status === "running").length, 0);
        const idleAgents = worlds.reduce((n, w) => n + w.agents.filter(a => a.status === "idle" || a.status === "waiting").length, 0);

        return (
          <div className="flex items-center gap-6 px-1">
            {[
              { label: "Worlds", value: worlds.length, sub: `${runningWorlds} active`, color: "text-foreground/70" },
              { label: "Agents", value: totalAgents, sub: `${runningAgents} running`, color: "text-foreground/70" },
              { label: "Running", value: runningAgents, icon: <IconBoltFilled size={12} className="text-green-400/60" />, color: "text-green-400/80" },
              { label: "Idle", value: idleAgents, icon: <IconMoonFilled size={12} className="text-amber-400/60" />, color: "text-amber-400/80" },
            ].map(({ label, value, sub, icon, color }) => (
              <div key={label} className="flex items-center gap-2.5">
                {icon}
                <div>
                  <p className={`text-sm font-mono font-medium ${color}`}>{value}</p>
                  <p className="text-[9px] uppercase tracking-[0.15em] text-muted-foreground/25">{sub ?? label}</p>
                </div>
              </div>
            ))}
          </div>
        );
      })()}

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32 rounded-lg" />
          ))}
        </div>
      ) : worlds.length === 0 && agents.length === 0 && !agentsLoading ? (
        <QuickStartWizard onComplete={() => { fetchWorlds(); fetchAgents(); refetchSidebar(); }} />
      ) : (
        <>
          {/* Worlds */}
          {worlds.length > 0 ? (
            <div className="relative overflow-hidden min-h-[320px] flex items-center" onClick={(e) => { if (e.target === e.currentTarget && selected !== null) setSelected(null); }}>
              {/* Planets — full width scrollable */}
              <div
                ref={scrollRef}
                className="flex gap-10 items-center will-change-transform select-none"
                style={{
                  transform: `translateX(${totalTx}px)`,
                  transition: isDragging ? "none" : "transform 0.8s cubic-bezier(0.16, 1, 0.3, 1)",
                  cursor: isDragging ? "grabbing" : "grab",
                }}
                onMouseDown={(e) => { e.preventDefault(); onDragStart(e.clientX); }}
                onTouchStart={(e) => onDragStart(e.touches[0].clientX)}
              >
                {worlds.map((world, i) => {
                  const isActive = selected === i;
                  const hasSelection = selected !== null;
                  return (
                    <div
                      key={world.id}
                      ref={(el) => { planetRefs.current[i] = el; }}
                      className="flex flex-col items-center shrink-0 cursor-pointer"
                      style={{
                        opacity: hasSelection && !isActive ? 0.35 : 1,
                        transform: hasSelection && !isActive ? "scale(0.9)" : "scale(1)",
                        filter: hasSelection && !isActive ? "blur(1px)" : "blur(0px)",
                        margin: isActive ? "0 36px" : "0",
                        transition: "opacity 0.7s ease-out, transform 0.7s ease-out, filter 0.7s ease-out, margin 0.9s cubic-bezier(0.16, 1, 0.3, 1)",
                      }}
                      onClick={() => { if (!wasDragging.current) setSelected(selected === i ? null : i); }}
                    >
                      <PlanetGlobe
                        world={world}
                        index={i}
                        isSelected={isActive}
                        onClick={() => { if (!wasDragging.current) setSelected(selected === i ? null : i); }}
                        onEnter={() => router.push(`/world/${world.id}`)}
                        compact
                      />
                    </div>
                  );
                })}
                {/* New world */}
                <div
                  className="flex flex-col items-center shrink-0 cursor-pointer"
                  style={{
                    opacity: selected !== null ? 0.2 : 0.5,
                    transform: selected !== null ? "scale(0.85)" : "scale(1)",
                    transition: "opacity 0.7s ease-out, transform 0.7s ease-out",
                  }}
                  onClick={() => setShowSpawn(true)}
                >
                  <PlanetGlobe
                    world={{ id: "w-new-00000", config: "default", agent: "", agents: [], status: "creating", created_at: "", container_id: "", workspace: "" } as World}
                    index={worlds.length}
                    isSelected={false}
                    onClick={() => setShowSpawn(true)}
                    compact
                    hideLabels
                  />
                </div>
              </div>

              {/* Floating world info panel */}
              {selected !== null && worlds[selected] && (() => {
                const w = worlds[selected];
                const name = extractName(w.id);
                const isRunning = w.status === "running" || w.status === "idle";
                return (
                  <div
                    ref={panelRef}
                    className="absolute top-1/2 -translate-y-1/2 right-4 w-[360px] z-10 rounded-3xl overflow-hidden animate-in fade-in slide-in-from-right-8 duration-600 ease-out"
                    style={{
                      background: "linear-gradient(135deg, rgba(255,255,255,0.06) 0%, rgba(255,255,255,0.02) 100%)",
                      backdropFilter: "blur(24px) saturate(1.2)",
                      border: "1px solid rgba(255,255,255,0.08)",
                      boxShadow: "0 24px 48px rgba(0,0,0,0.3), 0 4px 12px rgba(0,0,0,0.15), inset 0 1px 0 rgba(255,255,255,0.08)",
                    }}
                  >
                    {/* Header */}
                    <div className="px-6 pt-5 pb-4">
                      <div className="flex items-center justify-between mb-1">
                        <div className="flex items-center gap-3">
                          <div className="relative">
                            <div className={`w-3 h-3 rounded-full ${STATUS_DOT[w.status] ?? "bg-white/10"}`} style={{ boxShadow: `0 0 10px ${w.status === "running" ? "rgba(34,197,94,0.5)" : w.status === "idle" ? "rgba(234,179,8,0.4)" : "transparent"}` }} />
                            {w.status === "running" && (
                              <div className={`absolute inset-0 w-3 h-3 rounded-full animate-ping ${STATUS_DOT[w.status]}`} style={{ opacity: 0.4 }} />
                            )}
                          </div>
                          <h3 className="font-heading text-lg tracking-wide text-foreground/95">{name}</h3>
                        </div>
                        <button
                          onClick={() => setSelected(null)}
                          className="w-8 h-8 flex items-center justify-center rounded-xl text-muted-foreground/25 hover:text-foreground/60 hover:bg-white/[0.06] transition-all"
                        >
                          <IconX size={16} />
                        </button>
                      </div>
                      <p className="text-[11px] font-mono text-muted-foreground/30 pl-6">
                        {w.config ?? "default"} · {w.workspace ?? "/tmp"}
                      </p>
                    </div>

                    {/* Divider */}
                    <div className="h-px bg-gradient-to-r from-transparent via-white/[0.06] to-transparent" />

                    {/* Stats grid */}
                    <div className="px-6 py-4">
                      <div className="grid grid-cols-3 gap-4">
                        {[
                          { label: "Status", value: w.status, accent: w.status === "running" ? "text-green-400/80" : w.status === "idle" ? "text-amber-400/80" : "text-muted-foreground/50" },
                          { label: "Uptime", value: w.created_at ? (() => { const m = Math.floor((Date.now() - new Date(w.created_at).getTime()) / 60000); if (m < 60) return `${m}m`; const h = Math.floor(m / 60); if (h < 24) return `${h}h`; return `${Math.floor(h / 24)}d`; })() : "—", accent: "text-foreground/60" },
                          { label: "Agents", value: String(w.agents.length), accent: "text-foreground/60" },
                        ].map(({ label, value, accent }) => (
                          <div key={label} className="text-center">
                            <p className="text-[9px] uppercase tracking-[0.15em] text-muted-foreground/25 mb-1.5">{label}</p>
                            <p className={`text-sm font-mono font-medium ${accent}`}>{value}</p>
                          </div>
                        ))}
                      </div>
                    </div>

                    {/* Agents section */}
                    {w.agents.length > 0 && (
                      <>
                        <div className="h-px bg-gradient-to-r from-transparent via-white/[0.06] to-transparent" />
                        <div className="px-6 py-4">
                          <p className="text-[9px] uppercase tracking-[0.15em] text-muted-foreground/25 mb-3">Agents</p>
                          <div className="flex flex-wrap gap-2">
                            {w.agents.map((a) => (
                              <button
                                key={a.name}
                                onClick={(e) => { e.stopPropagation(); router.push(`/world/${w.id}/${a.name}`); }}
                                className="group/agent inline-flex items-center gap-2 text-xs font-mono pl-2.5 pr-3 py-1.5 rounded-xl bg-white/[0.04] text-muted-foreground/50 border border-white/[0.06] hover:bg-white/[0.08] hover:text-foreground/80 hover:border-white/[0.12] transition-all"
                              >
                                <span className={`w-2 h-2 rounded-full transition-shadow ${a.status === "running" ? "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.4)]" : a.status === "idle" ? "bg-amber-400 shadow-[0_0_6px_rgba(234,179,8,0.3)]" : "bg-zinc-500/30"}`} />
                                {a.name}
                                <IconArrowRight size={11} className="opacity-0 -ml-1 group-hover/agent:opacity-50 group-hover/agent:ml-0 transition-all" />
                              </button>
                            ))}
                          </div>
                        </div>
                      </>
                    )}

                    {/* Actions */}
                    <div className="h-px bg-gradient-to-r from-transparent via-white/[0.06] to-transparent" />
                    <div className="px-6 py-4 flex items-center gap-2">
                      <button
                        onClick={() => router.push(`/world/${w.id}`)}
                        className="flex-1 flex items-center justify-center gap-2 py-2.5 rounded-xl text-xs font-medium bg-white/[0.06] text-foreground/70 hover:text-foreground/95 hover:bg-white/[0.1] border border-white/[0.06] hover:border-white/[0.12] transition-all"
                      >
                        Enter World
                        <IconArrowRight size={14} />
                      </button>
                      {isRunning && (
                        <button
                          onClick={(e) => { e.stopPropagation(); apiDelete(`/api/worlds/${w.id}`).then(() => { fetchWorlds(); refetchSidebar(); setSelected(null); }); }}
                          className="py-2.5 px-4 rounded-xl text-xs text-muted-foreground/30 hover:text-red-400 hover:bg-red-500/[0.06] border border-transparent hover:border-red-500/15 transition-all"
                        >
                          Shutdown
                        </button>
                      )}
                    </div>
                  </div>
                );
              })()}
            </div>
          ) : (
            <div className="flex justify-center py-12">
              <div
                className="flex flex-col items-center cursor-pointer opacity-40 hover:opacity-70 transition-opacity"
                onClick={() => setShowSpawn(true)}
              >
                <PlanetGlobe
                  world={{ id: "w-new-00000", config: "default", agent: "", agents: [], status: "stopped", created_at: "", container_id: "", workspace: "" } as World}
                  index={0}
                  isSelected={false}
                  onClick={() => setShowSpawn(true)}
                />
                <p className="text-sm font-heading text-muted-foreground/30 mt-2">Spawn your first world</p>
              </div>
            </div>
          )}

          {/* ── Agent Activity Feed ── */}
          {worlds.length > 0 && (() => {
            const allAgents = worlds.flatMap(w => w.agents.map(a => ({ ...a, worldName: extractName(w.id), worldId: w.id })));
            if (allAgents.length === 0) return null;

            const statusIcon: Record<string, { icon: typeof IconBoltFilled; color: string; label: string }> = {
              running:  { icon: IconBoltFilled,    color: "text-green-400",  label: "Running" },
              waiting:  { icon: IconMessageFilled, color: "text-amber-400",  label: "Waiting" },
              idle:     { icon: IconCircleFilled,  color: "text-amber-400/50", label: "Idle" },
              sleeping: { icon: IconMoonFilled,    color: "text-purple-400", label: "Sleeping" },
              stopped:  { icon: IconCircleFilled,  color: "text-zinc-500/30",  label: "Stopped" },
            };

            return (
              <div>
                <h3 className="text-[10px] uppercase tracking-[0.15em] text-muted-foreground/30 mb-3 flex items-center gap-2">
                  <IconActivity size={12} className="opacity-50" />
                  Agent Activity
                </h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                  {allAgents.map((a) => {
                    const s = statusIcon[a.status] ?? statusIcon.stopped;
                    const Icon = s.icon;
                    return (
                      <button
                        key={`${a.worldId}-${a.name}`}
                        onClick={() => router.push(`/world/${a.worldId}/${a.name}`)}
                        className="group flex items-center gap-3 px-4 py-3 rounded-xl text-left transition-all hover:bg-white/[0.04] border border-transparent hover:border-white/[0.06]"
                      >
                        <div className="relative">
                          <Icon size={14} className={`${s.color} transition-colors`} />
                          {a.status === "running" && (
                            <div className="absolute -top-0.5 -right-0.5 w-2 h-2">
                              <div className="absolute inset-0 rounded-full bg-green-400 animate-ping opacity-40" />
                              <div className="absolute inset-0 rounded-full bg-green-400" />
                            </div>
                          )}
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-xs font-mono text-foreground/70 group-hover:text-foreground/90 truncate transition-colors">{a.name}</p>
                          <p className="text-[10px] text-muted-foreground/25">{a.worldName} · {s.label}</p>
                        </div>
                        <IconArrowRight size={12} className="text-muted-foreground/15 group-hover:text-muted-foreground/40 transition-colors shrink-0" />
                      </button>
                    );
                  })}
                </div>
              </div>
            );
          })()}

          {/* ── Recent Activity Timeline ── */}
          {worlds.length > 0 && (
            <div>
              <h3 className="text-[10px] uppercase tracking-[0.15em] text-muted-foreground/30 mb-3 flex items-center gap-2">
                <IconSparkles size={12} className="opacity-50" />
                Recent Activity
              </h3>
              <RecentActivity />
            </div>
          )}
        </>
      )}

      {/* Destroy All Confirmation Dialog */}
      {showDestroyAll && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => !destroyingAll && setShowDestroyAll(false)} />
          <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-red-500/30 shadow-2xl p-6">
            <div className="flex flex-col items-center text-center">
              <div className="w-14 h-14 rounded-2xl bg-red-500/10 border border-red-500/20 flex items-center justify-center mb-4">
                <IconAlertTriangle size={28} className="text-red-400" />
              </div>
              <h2 className="text-lg font-heading text-red-300 mb-2">Destroy All Worlds?</h2>
              <p className="text-xs text-red-300/60 mb-1">
                This will permanently destroy <span className="font-mono font-bold">{worlds.length}</span> world{worlds.length !== 1 ? "s" : ""} and all their agents.
              </p>
              <p className="text-xs text-red-300/40 mb-6">This action cannot be undone.</p>
              <div className="flex gap-3 w-full">
                <button
                  onClick={() => setShowDestroyAll(false)}
                  disabled={destroyingAll}
                  className="flex-1 px-4 py-2.5 rounded-xl text-sm text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
                >
                  Cancel
                </button>
                <button
                  onClick={handleDestroyAll}
                  disabled={destroyingAll}
                  className="flex-1 px-4 py-2.5 rounded-xl text-sm bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/30 transition-colors disabled:opacity-50"
                >
                  {destroyingAll ? (
                    <span className="flex items-center justify-center gap-2">
                      <span className="w-3.5 h-3.5 border-2 border-red-300/30 border-t-red-300/70 rounded-full animate-spin" />
                      Destroying...
                    </span>
                  ) : (
                    "Yes, destroy all"
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Spawn World Dialog */}
      {showSpawn && (
        <SpawnWorldDialog onClose={() => setShowSpawn(false)} onComplete={handleSpawnComplete} />
      )}
    </div>
  );
}

/* ── Quick Start Wizard ── */

function QuickStartWizard({ onComplete }: { onComplete: () => void }) {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [agentName, setAgentName] = useState("");
  const [purpose, setPurpose] = useState("");
  const [workspace, setWorkspace] = useState("");
  const [error, setError] = useState("");
  const [working, setWorking] = useState(false);

  const handleCreateAgent = async () => {
    if (!agentName.trim()) return;
    setWorking(true);
    setError("");
    try {
      const result = await apiAction("/api/agents", { name: agentName.trim() });
      if (!result.ok) {
        setError(result.error || "Failed to create agent");
        setWorking(false);
        return;
      }
      setStep(2);
    } catch {
      setError("Failed to connect to API");
    } finally {
      setWorking(false);
    }
  };

  const handleSetPurpose = async () => {
    // Purpose is optional, proceed to step 3
    setStep(3);
  };

  const handleSpawnWorld = async () => {
    setWorking(true);
    setError("");
    const effectiveWorkspace = workspace.trim() || `/tmp/spwn-${agentName.trim()}-${Math.random().toString(36).substring(2, 6)}`;
    try {
      const res = await fetch(goApiUrl("/api/worlds"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agent: agentName.trim(), workspace: effectiveWorkspace, config: "default", tier: "citizen" }),
        signal: AbortSignal.timeout(30000),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Failed to spawn world");
        setWorking(false);
        return;
      }
      onComplete();
      if (data.id) {
        router.push(`/world/${data.id}`);
      }
    } catch {
      setError("Failed to connect to API");
      setWorking(false);
    }
  };

  const steps = [
    { num: 1, label: "Create Agent", icon: <IconUser size={14} /> },
    { num: 2, label: "Set Purpose", icon: <IconBulb size={14} /> },
    { num: 3, label: "Spawn World", icon: <IconWorld size={14} /> },
  ];

  return (
    <div className="w-full max-w-lg mx-auto px-4">
      {/* Header */}
      <div className="text-center mb-8">
        <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-blue-500/20 to-purple-500/20 border border-white/[0.08] flex items-center justify-center mx-auto mb-4">
          <IconSparkles size={28} className="text-blue-400/60" />
        </div>
        <h2 className="text-xl font-heading text-foreground/90">Welcome to SPWN</h2>
        <p className="text-xs text-muted-foreground/40 mt-1 font-mono">Let&apos;s set up your first agent and world</p>
      </div>

      {/* Step indicators */}
      <div className="flex items-center justify-center gap-2 mb-8">
        {steps.map((s, i) => (
          <div key={s.num} className="flex items-center gap-2">
            <div className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-[10px] font-mono transition-all ${
              step > s.num
                ? "bg-green-500/15 text-green-400/80 border border-green-500/20"
                : step === s.num
                  ? "bg-white/[0.08] text-foreground/70 border border-white/[0.12]"
                  : "bg-white/[0.02] text-muted-foreground/25 border border-white/[0.04]"
            }`}>
              {step > s.num ? <IconCheck size={10} /> : s.icon}
              {s.label}
            </div>
            {i < steps.length - 1 && (
              <IconArrowRight size={10} className="text-muted-foreground/15" />
            )}
          </div>
        ))}
      </div>

      {/* Step content */}
      <div className="glass-subtle rounded-2xl p-6 space-y-4">
        {step === 1 && (
          <>
            <div>
              <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-2">
                Name your first agent
              </label>
              <input
                value={agentName}
                onChange={(e) => setAgentName(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") handleCreateAgent(); }}
                placeholder="e.g. atlas, neo, morpheus..."
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-4 py-3 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                autoFocus
              />
              <p className="text-[10px] text-muted-foreground/25 mt-2">
                Agents are autonomous AI entities that work inside worlds
              </p>
            </div>
            <button
              onClick={handleCreateAgent}
              disabled={!agentName.trim() || working}
              className="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed"
            >
              {working ? (
                <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
              ) : (
                <IconArrowRight size={16} />
              )}
              {working ? "Creating..." : "Create Agent"}
            </button>
          </>
        )}

        {step === 2 && (
          <>
            <div>
              <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-2">
                What should {agentName} do?
              </label>
              <textarea
                value={purpose}
                onChange={(e) => setPurpose(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); handleSetPurpose(); } }}
                placeholder="e.g. Build a REST API, Manage my infrastructure, Write documentation..."
                rows={3}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-4 py-3 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors resize-none"
                autoFocus
              />
              <p className="text-[10px] text-muted-foreground/25 mt-2">
                Optional — you can always change this later
              </p>
            </div>
            <button
              onClick={handleSetPurpose}
              className="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all"
            >
              <IconArrowRight size={16} />
              {purpose.trim() ? "Continue" : "Skip for now"}
            </button>
          </>
        )}

        {step === 3 && (
          <>
            <div>
              <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-2">
                Workspace path
              </label>
              <input
                value={workspace}
                onChange={(e) => setWorkspace(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") handleSpawnWorld(); }}
                placeholder={`/tmp/spwn-${agentName.trim() || "agent"}`}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-4 py-3 text-sm font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                autoFocus
              />
              <p className="text-[10px] text-muted-foreground/25 mt-2">
                The directory where {agentName} will work — leave empty for default
              </p>
            </div>

            {/* Preview */}
            <div className="rounded-lg bg-white/[0.02] border border-white/[0.05] px-3 py-3">
              <p className="text-[10px] uppercase tracking-widest text-muted-foreground/30 mb-1">Summary</p>
              <div className="text-[11px] text-muted-foreground/40 space-y-0.5">
                <p>→ Agent: <span className="text-foreground/60 font-mono">{agentName}</span></p>
                {purpose && <p>→ Purpose: <span className="text-foreground/60">{purpose}</span></p>}
                <p>→ Workspace: <span className="font-mono text-foreground/60">{workspace || `/tmp/spwn-${agentName.trim()}`}</span></p>
              </div>
            </div>

            <button
              onClick={handleSpawnWorld}
              disabled={working}
              className="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed"
            >
              {working ? (
                <>
                  <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                  Spawning...
                </>
              ) : (
                <>
                  <IconRocket size={16} />
                  Spawn World
                </>
              )}
            </button>
          </>
        )}

        {error && (
          <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 text-xs text-red-400 font-mono">
            {error}
          </div>
        )}
      </div>
    </div>
  );
}

/* ── Spawn World Dialog ── */

interface SpawnAgentListItem {
  name: string;
  path: string;
  layers: Record<string, string[]>;
}

function SpawnWorldDialog({ onClose, onComplete }: { onClose: () => void; onComplete: () => void }) {
  const router = useRouter();
  const [agentName, setAgentName] = useState("");
  const [workspace, setWorkspace] = useState("");
  const [config, setConfig] = useState("default");
  const [tier, setTier] = useState("citizen");
  const [spawning, setSpawning] = useState(false);
  const [availableAgents, setAvailableAgents] = useState<SpawnAgentListItem[]>([]);
  const [error, setError] = useState("");
  const [creatingAgent, setCreatingAgent] = useState(false);
  const [newAgentName, setNewAgentName] = useState("");

  // Generate a sensible default workspace when agent is selected
  const defaultWorkspace = useMemo(() => {
    if (!agentName) return "/tmp/spwn-world";
    const rand = Math.random().toString(36).substring(2, 6);
    return `/tmp/spwn-${agentName}-${rand}`;
  }, [agentName]);

  // Fetch available agents for dropdown
  useEffect(() => {
    apiGet<SpawnAgentListItem[]>("/api/agents")
      .then((agents) => {
        setAvailableAgents(agents ?? []);
        if (agents && agents.length > 0 && !agentName) {
          setAgentName(agents[0].name);
        }
      })
      .catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleCreateInlineAgent = async () => {
    if (!newAgentName.trim()) return;
    setCreatingAgent(true);
    setError("");
    try {
      const result = await apiAction("/api/agents", { name: newAgentName.trim() });
      if (!result.ok) {
        setError(result.error || "Failed to create agent");
        return;
      }
      const created = { name: newAgentName.trim(), path: "", layers: {} };
      setAvailableAgents((prev) => [...prev, created]);
      setAgentName(newAgentName.trim());
      setNewAgentName("");
    } catch {
      setError("Failed to connect to API");
    } finally {
      setCreatingAgent(false);
    }
  };

  const effectiveWorkspace = workspace || defaultWorkspace;

  const handleSpawn = async () => {
    setSpawning(true);
    setError("");
    try {
      const res = await fetch(goApiUrl("/api/worlds"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agent: agentName.trim(), workspace: effectiveWorkspace, config, tier }),
        signal: AbortSignal.timeout(30000),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || "Failed to spawn world");
        setSpawning(false);
        return;
      }
      onComplete();
      onClose();
      // Redirect to the new world if we got an ID back
      if (data.id) {
        router.push(`/world/${data.id}`);
      }
    } catch {
      setError("Failed to connect to API");
      setSpawning(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />

      {/* Dialog */}
      <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl">
        {/* Header */}
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-heading text-foreground/90">Spawn World</h2>
            <p className="text-[11px] text-muted-foreground/40 mt-0.5">Create a new isolated world for your agent</p>
          </div>
          <button
            onClick={onClose}
            className="text-muted-foreground/40 hover:text-foreground/60 transition-colors"
          >
            <IconX size={18} />
          </button>
        </div>

        {/* Form */}
        <div className="px-6 pb-6 space-y-4">
          {/* Agent name — dropdown if agents exist, inline creation if not */}
          <div>
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
              Agent
            </label>
            {availableAgents.length > 0 ? (
              <select
                value={agentName}
                onChange={(e) => setAgentName(e.target.value)}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
                autoFocus
              >
                {availableAgents.map((a) => (
                  <option key={a.name} value={a.name}>{a.name}</option>
                ))}
              </select>
            ) : (
              <div className="space-y-2">
                <div className="rounded-lg bg-yellow-500/5 border border-yellow-500/15 px-3 py-2">
                  <p className="text-[11px] text-yellow-400/60">No agents yet. Create one first:</p>
                </div>
                <div className="flex gap-2">
                  <input
                    value={newAgentName}
                    onChange={(e) => setNewAgentName(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") handleCreateInlineAgent(); }}
                    placeholder="Agent name..."
                    className="flex-1 bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                    autoFocus
                  />
                  <button
                    onClick={handleCreateInlineAgent}
                    disabled={!newAgentName.trim() || creatingAgent}
                    className="px-3 py-2.5 rounded-lg text-xs bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed whitespace-nowrap"
                  >
                    {creatingAgent ? "Creating..." : "Create"}
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Workspace */}
          <div>
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
              Workspace Path
            </label>
            <input
              value={workspace}
              onChange={(e) => setWorkspace(e.target.value)}
              placeholder={defaultWorkspace}
              className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
            />
          </div>

          {/* Config + Tier row */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                Config
              </label>
              <select
                value={config}
                onChange={(e) => setConfig(e.target.value)}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
              >
                {AVAILABLE_CONFIGS.map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                Agent Tier
              </label>
              <select
                value={tier}
                onChange={(e) => setTier(e.target.value)}
                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
              >
                <option value="governor">Governor</option>
                <option value="citizen">Citizen</option>
                <option value="npc">NPC</option>
              </select>
            </div>
          </div>

          {/* Preview of what will happen */}
          <div className="rounded-lg bg-white/[0.02] border border-white/[0.05] px-3 py-3 space-y-2">
            <p className="text-[10px] uppercase tracking-widest text-muted-foreground/30 mb-1">Preview</p>
            <div className="font-mono text-[11px] text-muted-foreground/35">
              spwn up --agent {agentName || "‹name›"} --tier {tier} --config {config} -w {effectiveWorkspace}
            </div>
            <div className="text-[10px] text-muted-foreground/25 space-y-0.5">
              <p>→ Creates isolated Docker container</p>
              <p>→ Mounts agent mind from <span className="font-mono">~/.spwn/agents/{agentName || "‹name›"}</span></p>
              <p>→ Workspace: <span className="font-mono">{effectiveWorkspace}</span></p>
            </div>
          </div>

          {/* Error display */}
          {error && (
            <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 text-xs text-red-400 font-mono">
              {error}
            </div>
          )}

          {/* Spawn button */}
          <button
            onClick={handleSpawn}
            disabled={!agentName.trim() || spawning}
            className="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] hover:text-foreground/90 border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed"
          >
            {spawning ? (
              <>
                <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                Spawning...
              </>
            ) : (
              <>
                <IconRocket size={16} />
                Spawn World
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
