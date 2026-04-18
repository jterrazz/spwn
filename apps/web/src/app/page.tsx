'use client';

import {
    IconActivity,
    IconAlertTriangle,
    IconArrowRight,
    IconBriefcase,
    IconBuildingFactory2,
    IconBulb,
    IconCheck,
    IconFlask,
    IconLoader2,
    IconMoonFilled,
    IconPlus,
    IconRobot,
    IconRocket,
    IconSparkles,
    IconTerminal2,
    IconUser,
    IconUsers,
    IconWorld,
    IconWorldFilled,
    IconX,
} from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { ActionButton } from '@/components/action-button';
import { useRefetch } from '@/components/app-shell';
import {
    ItemList,
    MetricGrid,
    ProgressBar,
    SectionLabel,
    Separator,
    StatusDot,
} from '@/components/ds';
import { GLASS_PILL_CLASS } from '@/components/glass-pill';
import { NewWorldCard } from '@/components/new-world-card';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { Planet as PlanetGlobe } from '@/components/planet';
import { ProgressShimmer } from '@/components/progress-shimmer';
import { Skeleton } from '@/components/ui/skeleton';
import { WorldPlanet } from '@/components/world-planet';
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts';
import { usePageTitle } from '@/hooks/use-page-title';
import { useProgressMessages } from '@/hooks/use-progress-messages';
import { apiAction, apiDelete, apiGet, apiPost, goApiUrl } from '@/lib/api-client';
import { AVAILABLE_CONFIGS, getWorldName, type World } from '@/lib/types';

interface AgentListItem {
    name: string;
    path: string;
    layers: Record<string, string[]>;
}

export default function UniverseMapPage() {
    const [worlds, setWorlds] = useState<World[]>([]);
    const [agents, setAgents] = useState<AgentListItem[]>([]);
    const [selected, setSelected] = useState<null | number>(null);
    const scrollRef = useRef<HTMLDivElement>(null);
    const planetRefs = useRef<(HTMLDivElement | null)[]>([]);
    const [showSpawn, setShowSpawn] = useState(false);
    const [showDestroyAll, setShowDestroyAll] = useState(false);
    const [destroyingAll, setDestroyingAll] = useState(false);
    const [loading, setLoading] = useState(true);
    const [agentsLoading, setAgentsLoading] = useState(true);
    const router = useRouter();
    const refetchSidebar = useRefetch();
    usePageTitle('Worlds');

    const fetchWorlds = () => {
        apiGet<World[]>('/api/worlds')
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
        apiGet<AgentListItem[]>('/api/agents')
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
            if (showSpawn) {
                return;
            }
            if (worlds.length === 0) {
                return;
            }
            if (e.key === 'ArrowRight' || e.key === 'd') {
                setSelected((s) => (s === null ? 0 : (s + 1) % worlds.length));
            } else if (e.key === 'ArrowLeft' || e.key === 'a') {
                setSelected((s) =>
                    s === null ? worlds.length - 1 : (s - 1 + worlds.length) % worlds.length,
                );
            } else if (e.key === 'Enter' && selected !== null) {
                router.push(`/world/${worlds[selected].id}`);
            } else if (e.key === 'Escape' && selected !== null) {
                setSelected(null);
            }
        };
        globalThis.addEventListener('keydown', handleKey);
        return () => globalThis.removeEventListener('keydown', handleKey);
    }, [worlds, selected, router, showSpawn]);

    // ── Planet centering + drag system ──
    // Uses offsetLeft (static layout position, unaffected by transform) to avoid circular deps.
    // Single `tx` state = final translateX. Drag adds delta on top during gesture.
    const panelRef = useRef<HTMLDivElement>(null);
    const [, _forceRender] = useState(0);
    const [tx, setTx] = useState(0);
    const [dragDelta, setDragDelta] = useState(0);
    const [isDragging, setIsDragging] = useState(false);
    const dragRef = useRef({ startX: 0, startTx: 0, moved: false });
    const wasDragging = useRef(false);
    const lastFocusIdx = useRef<null | number>(null);

    // Get the visible width of the viewport area (minus panel if showing)
    const getVisibleWidth = useCallback((withPanel: boolean) => {
        const parent = scrollRef.current?.parentElement;
        if (!parent) {
            return 800;
        }
        if (!withPanel) {
            return parent.clientWidth;
        }
        const panelW = panelRef.current?.offsetWidth ?? 380;
        return parent.clientWidth - panelW - 24; // 24px gap between planet and panel
    }, []);

    // Get the static center X of a planet (its layout position, not affected by transform)
    const getPlanetCenter = useCallback((idx: number) => {
        const el = planetRefs.current[idx];
        if (!el) {
            return 0;
        }
        return el.offsetLeft + el.offsetWidth / 2;
    }, []);

    // Compute translateX to center a planet (or group) in the visible area
    const centerOn = useCallback(
        (idx: null | number, withPanel: boolean) => {
            const vw = getVisibleWidth(withPanel);
            const target = vw / 2;

            if (idx !== null) {
                return target - getPlanetCenter(idx);
            }

            // Center the group of real planets
            const count = planetRefs.current.filter(Boolean).length;
            if (count === 0) {
                return 0;
            }
            const first = getPlanetCenter(0);
            const last = getPlanetCenter(Math.min(count - 1, planetRefs.current.length - 1));
            return target - (first + last) / 2;
        },
        [getVisibleWidth, getPlanetCenter],
    );

    // Recenter when selection changes (double-RAF to ensure panel is mounted and measured)
    useEffect(() => {
        if (selected !== null) {
            lastFocusIdx.current = selected;
        }
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

    const onDragMove = useCallback(
        (x: number) => {
            if (!isDragging) {
                return;
            }
            const dx = x - dragRef.current.startX;
            if (Math.abs(dx) > 3) {
                dragRef.current.moved = true;
            }
            setDragDelta(dx);
        },
        [isDragging],
    );

    const onDragEnd = useCallback(() => {
        setIsDragging(false);
        if (!dragRef.current.moved) {
            setDragDelta(0);
            return;
        }

        // Snap to nearest planet
        const vw = getVisibleWidth(selected !== null);
        const target = vw / 2;
        const currentTx = tx + dragDelta;

        let bestIdx = 0;
        let bestDist = Infinity;
        for (let i = 0; i < worlds.length; i++) {
            const screenX = getPlanetCenter(i) + currentTx;
            const dist = Math.abs(screenX - target);
            if (dist < bestDist) {
                bestDist = dist;
                bestIdx = i;
            }
        }

        setDragDelta(0);
        setTx(target - getPlanetCenter(bestIdx));
    }, [tx, dragDelta, selected, worlds.length, getVisibleWidth, getPlanetCenter]);

    // Global pointer listeners
    useEffect(() => {
        if (!isDragging) {
            return;
        }
        const move = (e: MouseEvent) => onDragMove(e.clientX);
        const up = () => onDragEnd();
        const tmove = (e: TouchEvent) => onDragMove(e.touches[0].clientX);
        const tend = () => onDragEnd();
        globalThis.addEventListener('mousemove', move);
        globalThis.addEventListener('mouseup', up);
        globalThis.addEventListener('touchmove', tmove, { passive: true });
        globalThis.addEventListener('touchend', tend);
        return () => {
            globalThis.removeEventListener('mousemove', move);
            globalThis.removeEventListener('mouseup', up);
            globalThis.removeEventListener('touchmove', tmove);
            globalThis.removeEventListener('touchend', tend);
        };
    }, [isDragging, onDragMove, onDragEnd]);

    const totalTx = tx + dragDelta;
    useEffect(() => {
        wasDragging.current = dragRef.current.moved;
    }, [isDragging]);

    // Recenter on globalThis resize
    useEffect(() => {
        const onResize = () => {
            const focusIdx = selected ?? lastFocusIdx.current;
            setTx(centerOn(focusIdx, selected !== null));
        };
        globalThis.addEventListener('resize', onResize);
        return () => globalThis.removeEventListener('resize', onResize);
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
            // Ignore errors - worlds may already be gone
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

    const runningAgents = worlds.reduce(
        (n, w) => n + w.agents.filter((a) => a.status === 'running').length,
        0,
    );
    const idleAgents = worlds.reduce(
        (n, w) => n + w.agents.filter((a) => a.status === 'idle' || a.status === 'waiting').length,
        0,
    );

    return (
        <Page className="flex flex-col h-full">
            <PageHeader
                actions={
                    <DashboardHeaderStats
                        idleAgents={idleAgents}
                        onSpawn={() => setShowSpawn(true)}
                        runningAgents={runningAgents}
                        worldsCount={worlds.length}
                    />
                }
                description="Isolated environments where your agents live and work."
                title="Worlds"
            />

            {loading && (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {[1, 2, 3].map((i) => (
                        <Skeleton className="h-32 rounded-lg" key={i} />
                    ))}
                </div>
            )}
            {!loading && worlds.length === 0 && agents.length === 0 && !agentsLoading && (
                <QuickStartWizard
                    onComplete={() => {
                        fetchWorlds();
                        fetchAgents();
                        refetchSidebar();
                    }}
                />
            )}
            {!loading && !(worlds.length === 0 && agents.length === 0 && !agentsLoading) && (
                <>
                    {/* Worlds */}
                    {worlds.length > 0 ? (
                        <div
                            className="relative flex-1 min-h-[320px] -mx-6 md:-mx-8 overflow-hidden"
                            onClick={(e) => {
                                if (e.target === e.currentTarget && selected !== null) {
                                    setSelected(null);
                                }
                            }}
                        >
                            <div className="h-full flex items-center pb-24">
                                {/* Planets - full width scrollable */}
                                <div
                                    className="flex gap-10 items-center will-change-transform select-none"
                                    onMouseDown={(e) => {
                                        e.preventDefault();
                                        onDragStart(e.clientX);
                                    }}
                                    onTouchStart={(e) => onDragStart(e.touches[0].clientX)}
                                    ref={scrollRef}
                                    style={{
                                        transform: `translateX(${totalTx}px)`,
                                        transition: isDragging
                                            ? 'none'
                                            : 'transform 0.8s cubic-bezier(0.16, 1, 0.3, 1)',
                                        cursor: isDragging ? 'grabbing' : 'grab',
                                    }}
                                >
                                    {worlds.map((world, i) => {
                                        const isActive = selected === i;
                                        const hasSelection = selected !== null;
                                        return (
                                            <div
                                                className="flex flex-col items-center shrink-0 cursor-pointer"
                                                key={world.id}
                                                onClick={() => {
                                                    if (!wasDragging.current) {
                                                        setSelected(selected === i ? null : i);
                                                    }
                                                }}
                                                ref={(el) => {
                                                    planetRefs.current[i] = el;
                                                }}
                                                style={{
                                                    opacity: hasSelection && !isActive ? 0.35 : 1,
                                                    transform:
                                                        hasSelection && !isActive
                                                            ? 'scale(0.9)'
                                                            : 'scale(1)',
                                                    filter:
                                                        hasSelection && !isActive
                                                            ? 'blur(1px)'
                                                            : 'blur(0px)',
                                                    margin: isActive ? '0 36px' : '0',
                                                    transition:
                                                        'opacity 0.7s ease-out, transform 0.7s ease-out, filter 0.7s ease-out, margin 0.9s cubic-bezier(0.16, 1, 0.3, 1)',
                                                }}
                                            >
                                                <PlanetGlobe
                                                    compact
                                                    index={i}
                                                    isSelected={isActive}
                                                    onClick={() => {
                                                        if (!wasDragging.current) {
                                                            setSelected(selected === i ? null : i);
                                                        }
                                                    }}
                                                    onEnter={() =>
                                                        router.push(`/world/${world.id}`)
                                                    }
                                                    world={world}
                                                />
                                            </div>
                                        );
                                    })}
                                    {/* New world - same card, same animations */}
                                    <NewWorldCard
                                        onClick={() => setShowSpawn(true)}
                                        opacity={selected !== null ? 0.2 : 0.5}
                                        scale={selected !== null ? 0.85 : 1}
                                        tint="creating"
                                    />
                                </div>
                            </div>

                            {/* Floating world info panel - lives OUTSIDE the negative-margin
                carousel so it doesn't cause horizontal overflow. Positioned
                absolutely within the flex-1 parent that wraps the carousel. */}
                            {selected !== null &&
                                worlds[selected] &&
                                (() => {
                                    const w = worlds[selected];
                                    const name = getWorldName(w);
                                    const isRunning = w.status === 'running' || w.status === 'idle';
                                    return (
                                        <div className="absolute inset-y-0 right-6 md:right-8 w-[340px] z-10 flex items-center pb-24 pointer-events-none">
                                            <div
                                                className="w-full rounded-2xl overflow-hidden border border-foreground/[0.08] dark:border-white/[0.1] bg-foreground/[0.04] dark:bg-white/[0.05] backdrop-blur-md shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)] pointer-events-auto animate-in fade-in slide-in-from-right-12 duration-500 ease-out"
                                                ref={panelRef}
                                            >
                                                <div className="p-5 space-y-5">
                                                    {/* Header */}
                                                    <div className="flex items-center justify-between">
                                                        <div className="flex items-center gap-3 min-w-0">
                                                            <WorldPlanet size="md" world={w} />
                                                            <div className="min-w-0">
                                                                <h3 className="text-sm font-mono font-bold text-foreground/95 truncate">
                                                                    {name}
                                                                </h3>
                                                                <p className="text-[10px] font-mono text-muted-foreground/35 truncate">
                                                                    {w.config} ·{' '}
                                                                    <StatusDot
                                                                        className="inline-block align-middle"
                                                                        status={w.status}
                                                                    />{' '}
                                                                    {w.status}
                                                                </p>
                                                            </div>
                                                        </div>
                                                        <button
                                                            className="w-7 h-7 flex items-center justify-center rounded-full text-muted-foreground/30 hover:text-foreground/70 transition-colors shrink-0"
                                                            onClick={() => setSelected(null)}
                                                        >
                                                            <IconX size={14} />
                                                        </button>
                                                    </div>

                                                    {/* Metrics */}
                                                    <MetricGrid
                                                        columns={3}
                                                        items={[
                                                            {
                                                                label: 'Uptime',
                                                                value: w.created_at
                                                                    ? (() => {
                                                                          const m = Math.floor(
                                                                              (Date.now() -
                                                                                  new Date(
                                                                                      w.created_at,
                                                                                  ).getTime()) /
                                                                                  60_000,
                                                                          );
                                                                          if (m < 60) {
                                                                              return `${m}m`;
                                                                          }
                                                                          const h = Math.floor(
                                                                              m / 60,
                                                                          );
                                                                          if (h < 24) {
                                                                              return `${h}h`;
                                                                          }
                                                                          return `${Math.floor(h / 24)}d`;
                                                                      })()
                                                                    : '-',
                                                            },
                                                            {
                                                                label: 'Agents',
                                                                value: w.agents.length,
                                                            },
                                                            {
                                                                label: 'Workspaces',
                                                                value: w.workspaces?.length ?? 0,
                                                            },
                                                        ]}
                                                    />

                                                    <Separator />

                                                    {/* Agents alive */}
                                                    {w.agents.length > 0 && (
                                                        <ProgressBar
                                                            label="Alive"
                                                            value={
                                                                w.agents.length === 0
                                                                    ? 0
                                                                    : Math.round(
                                                                          (w.agents.filter(
                                                                              (a) =>
                                                                                  a.status ===
                                                                                      'running' ||
                                                                                  a.status ===
                                                                                      'idle' ||
                                                                                  a.status ===
                                                                                      'waiting',
                                                                          ).length /
                                                                              w.agents.length) *
                                                                              100,
                                                                      )
                                                            }
                                                        />
                                                    )}

                                                    {/* Agents */}
                                                    {w.agents.length > 0 && (
                                                        <div>
                                                            <SectionLabel>Agents</SectionLabel>
                                                            <ItemList
                                                                items={w.agents.map((a) => ({
                                                                    name: a.name,
                                                                    detail: a.status,
                                                                    href: `/agents/${encodeURIComponent(a.name)}?world=${w.id}`,
                                                                }))}
                                                            />
                                                        </div>
                                                    )}

                                                    {/* Workspaces */}
                                                    {w.workspaces && w.workspaces.length > 0 && (
                                                        <div>
                                                            <SectionLabel>Workspaces</SectionLabel>
                                                            <ItemList
                                                                items={w.workspaces.map((ws) => ({
                                                                    name: ws.name,
                                                                    detail: ws.readonly
                                                                        ? `${ws.path} (ro)`
                                                                        : ws.path,
                                                                }))}
                                                            />
                                                        </div>
                                                    )}

                                                    {/* Actions */}
                                                    <div className="flex items-center gap-2">
                                                        <button
                                                            className="flex-1 flex items-center justify-center gap-2 h-9 rounded-full text-xs font-mono font-medium bg-white/[0.06] text-foreground/70 hover:text-foreground/95 hover:bg-white/[0.1] border border-white/[0.06] hover:border-white/[0.12] transition-all"
                                                            onClick={() =>
                                                                router.push(`/world/${w.id}`)
                                                            }
                                                        >
                                                            Enter World
                                                            <IconArrowRight size={13} />
                                                        </button>
                                                        {isRunning && (
                                                            <button
                                                                className="h-9 px-3.5 rounded-full text-[11px] font-mono text-muted-foreground/30 hover:text-red-400 hover:bg-red-500/[0.06] border border-transparent hover:border-red-500/15 transition-all"
                                                                onClick={(e) => {
                                                                    e.stopPropagation();
                                                                    apiDelete(
                                                                        `/api/worlds/${w.id}`,
                                                                    ).then(() => {
                                                                        fetchWorlds();
                                                                        refetchSidebar();
                                                                        setSelected(null);
                                                                    });
                                                                }}
                                                            >
                                                                Shutdown
                                                            </button>
                                                        )}
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    );
                                })()}
                        </div>
                    ) : (
                        <EmptyWorldsView
                            agents={agents}
                            onRefetch={() => {
                                fetchWorlds();
                                refetchSidebar();
                            }}
                            onSpawn={() => setShowSpawn(true)}
                        />
                    )}
                </>
            )}

            {/* Destroy All Confirmation Dialog */}
            {showDestroyAll && (
                <div className="fixed inset-0 z-50 flex items-center justify-center">
                    <div
                        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                        onClick={() => !destroyingAll && setShowDestroyAll(false)}
                    />
                    <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-red-500/30 shadow-2xl p-6">
                        <div className="flex flex-col items-center text-center">
                            <div className="w-14 h-14 rounded-2xl bg-red-500/10 border border-red-500/20 flex items-center justify-center mb-4">
                                <IconAlertTriangle className="text-red-400" size={28} />
                            </div>
                            <h2 className="text-lg font-heading text-red-300 mb-2">
                                Destroy All Worlds?
                            </h2>
                            <p className="text-xs text-red-300/60 mb-1">
                                This will permanently destroy{' '}
                                <span className="font-mono font-bold">{worlds.length}</span> world
                                {worlds.length !== 1 ? 's' : ''} and all their agents.
                            </p>
                            <p className="text-xs text-red-300/40 mb-6">
                                This action cannot be undone.
                            </p>
                            <div className="flex gap-3 w-full">
                                <button
                                    className="flex-1 px-4 py-2.5 rounded-xl text-sm text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
                                    disabled={destroyingAll}
                                    onClick={() => setShowDestroyAll(false)}
                                >
                                    Cancel
                                </button>
                                <button
                                    className="flex-1 px-4 py-2.5 rounded-xl text-sm bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/30 transition-colors disabled:opacity-50"
                                    disabled={destroyingAll}
                                    onClick={handleDestroyAll}
                                >
                                    {destroyingAll ? (
                                        <span className="flex items-center justify-center gap-2">
                                            <span className="w-3.5 h-3.5 border-2 border-red-300/30 border-t-red-300/70 rounded-full animate-spin" />
                                            Destroying...
                                        </span>
                                    ) : (
                                        'Yes, destroy all'
                                    )}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* New World Dialog */}
            {showSpawn && (
                <SpawnWorldDialog
                    onClose={() => setShowSpawn(false)}
                    onComplete={handleSpawnComplete}
                />
            )}
        </Page>
    );
}

/* ── Quick Start Wizard ── */

function DashboardHeaderStats({
    worldsCount,
    runningAgents,
    idleAgents,
    onSpawn,
}: {
    worldsCount: number;
    runningAgents: number;
    idleAgents: number;
    onSpawn: () => void;
}) {
    const [expanded, setExpanded] = useState(false);

    const items = [
        {
            key: 'worlds',
            icon: <IconWorldFilled size={15} />,
            value: worldsCount,
            label: 'worlds',
            pillClass: 'text-foreground/78',
            iconWrapClass: '',
            labelClass: 'tracking-[0.13em]',
            widthCollapsed: 52,
            widthExpanded: 102,
        },
        {
            key: 'running',
            icon: <IconActivity size={14} stroke={2.2} />,
            value: runningAgents,
            label: 'alive',
            pillClass: 'text-emerald-100/95',
            iconWrapClass: '',
            labelClass: 'tracking-[0.13em]',
            widthCollapsed: 52,
            widthExpanded: 92,
        },
        {
            key: 'idle',
            icon: <IconMoonFilled size={14} />,
            value: idleAgents,
            label: 'sleeping',
            pillClass: 'text-amber-100/95',
            iconWrapClass: '',
            labelClass: 'tracking-[0.11em]',
            widthCollapsed: 52,
            widthExpanded: 112,
        },
    ];

    return (
        <div className="flex flex-wrap items-center justify-end gap-2 md:max-w-[620px]">
            <div
                className={`${GLASS_PILL_CLASS} flex h-[42px] flex-nowrap items-center justify-end gap-1 px-2.5 transition-all duration-300 ease-out`}
                onBlur={() => setExpanded(false)}
                onFocus={() => setExpanded(true)}
                onMouseEnter={() => setExpanded(true)}
                onMouseLeave={() => setExpanded(false)}
            >
                {items.map((item) => (
                    <button
                        className={`flex h-[30px] items-center gap-1.5 overflow-hidden rounded-full border border-transparent px-2 ${item.pillClass}`}
                        key={item.key}
                        style={{
                            width: expanded ? item.widthExpanded : item.widthCollapsed,
                            transition: 'width 280ms cubic-bezier(0.16, 1, 0.3, 1)',
                        }}
                        type="button"
                    >
                        <span
                            className={`flex h-[18px] w-[18px] shrink-0 items-center justify-center self-center ${item.iconWrapClass}`}
                        >
                            <span className="block leading-none translate-y-[0.5px]">
                                {item.icon}
                            </span>
                        </span>
                        <span className="flex items-baseline gap-1.5 self-center whitespace-nowrap">
                            <span className="text-[12px] font-mono font-medium leading-none">
                                {item.value}
                            </span>
                            <span
                                className={`text-[9px] uppercase font-medium leading-none ${item.labelClass}`}
                                style={{
                                    opacity: expanded ? 1 : 0,
                                    transform: expanded ? 'translateX(0)' : 'translateX(-6px)',
                                    transition:
                                        'opacity 180ms ease, transform 280ms cubic-bezier(0.16, 1, 0.3, 1)',
                                }}
                            >
                                {item.label}
                            </span>
                        </span>
                    </button>
                ))}
            </div>

            <ActionButton
                compact
                icon={<IconPlus size={18} stroke={2.4} />}
                label="New World"
                onClick={onSpawn}
            />
        </div>
    );
}

interface GalleryExample {
    slug: string;
    name: string;
    tagline: string;
    description: string;
    agents: string[];
    worlds: string[];
    command?: string;
}

function EmptyWorldsView({
    agents,
    onSpawn,
    onRefetch,
}: {
    agents: AgentListItem[];
    onSpawn: () => void;
    onRefetch: () => void;
}) {
    const hasAgents = agents.length > 0;
    const router = useRouter();
    const [gallery, setGallery] = useState<GalleryExample[] | null>(null);
    const [installing, setInstalling] = useState<null | string>(null);
    const [installError, setInstallError] = useState<null | string>(null);

    useEffect(() => {
        apiGet<{ examples: GalleryExample[] }>('/api/examples')
            .then((data) => setGallery(data.examples ?? []))
            .catch(() => setGallery([]));
    }, []);

    const handleInstallAndSpawn = async (ex: GalleryExample) => {
        setInstalling(ex.slug);
        setInstallError(null);
        try {
            // 1. Copy template files into ~/.spwn/ (idempotent - skips
            //    Existing agents/worlds so users don't lose local edits).
            await apiPost(`/api/examples/${ex.slug}/install`);

            // 2. Immediately spawn the first world with its canonical
            //    Agent set so the user lands in a live container on click.
            const primaryWorld = ex.worlds[0];
            const body: Record<string, unknown> = { config: primaryWorld };
            if (ex.agents.length === 1) {
                body.agent = ex.agents[0];
            } else if (ex.agents.length > 1) {
                body.agents = ex.agents.map((name) => ({ name, role: 'worker' }));
            }
            const res = await fetch(goApiUrl('/api/worlds'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({ error: 'Unknown error' }));
                throw new Error(err.error || `spawn failed (${res.status})`);
            }
            const world = await res.json();
            onRefetch();
            if (world?.id) {
                router.push(`/world/${world.id}`);
            }
        } catch (error) {
            setInstallError(error instanceof Error ? error.message : 'install failed');
            setInstalling(null);
        }
    };

    return (
        <div className="flex-1 min-h-[400px] flex items-start justify-center pt-12 pb-16 px-4">
            <div className="w-full max-w-5xl">
                <div className="text-center mb-10">
                    <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 text-[10px] uppercase tracking-wider text-muted-foreground/70">
                        <IconSparkles size={11} />
                        Start from a template
                    </div>
                    <h2 className="font-heading text-2xl tracking-wide text-foreground/90">
                        {hasAgents
                            ? 'Give your agents a world to work in'
                            : 'Pick a template and spawn in one click'}
                    </h2>
                    <p className="mx-auto mt-2 max-w-lg text-sm text-muted-foreground/60">
                        {hasAgents
                            ? `You have ${agents.length} agent${agents.length > 1 ? 's' : ''} installed. Pick a template to put one to work, or build your own world from scratch.`
                            : 'Each template ships a full world config + pre-written agents with profiles. Clicking Install & spawn copies the files into ~/.spwn, creates a container and drops you straight into a conversation.'}
                    </p>
                </div>

                {gallery === null && (
                    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                        {[0, 1, 2, 3, 4].map((i) => (
                            <Skeleton className="h-52 rounded-2xl" key={i} />
                        ))}
                    </div>
                )}
                {gallery !== null && gallery.length === 0 && (
                    <p className="text-center text-sm text-muted-foreground/60">
                        No examples bundled in this build.
                    </p>
                )}
                {gallery !== null && gallery.length > 0 && (
                    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                        {gallery.map((ex, i) => (
                            <GalleryCard
                                busy={installing === ex.slug}
                                disabled={installing !== null && installing !== ex.slug}
                                example={ex}
                                featured={i === 0}
                                key={ex.slug}
                                onInstall={() => handleInstallAndSpawn(ex)}
                            />
                        ))}
                    </div>
                )}

                {installError && (
                    <p className="mt-4 text-center text-xs text-red-300/80">{installError}</p>
                )}

                <div className="mt-10 flex flex-col items-center gap-2">
                    <button
                        className="inline-flex items-center gap-2 text-[11px] uppercase tracking-wider text-muted-foreground/50 hover:text-foreground/80 transition-colors"
                        onClick={onSpawn}
                    >
                        <IconRocket size={12} />
                        Or build your own world from scratch
                    </button>
                    <div className="flex items-center gap-2 text-[10px] text-muted-foreground/30 font-mono">
                        <IconTerminal2 size={11} />
                        <span>spwn example list</span>
                    </div>
                </div>
            </div>
        </div>
    );
}

const EXAMPLE_THEMES: Record<string, { icon: React.ReactNode; accent: string; gradient: string }> =
    {
        startup: {
            icon: <IconBriefcase size={18} />,
            accent: 'text-amber-400/80',
            gradient: 'from-amber-500/15 to-orange-500/10',
        },
        matrix: {
            icon: <IconRobot size={18} />,
            accent: 'text-green-400/80',
            gradient: 'from-green-500/15 to-emerald-500/10',
        },
        'paperclip-factory': {
            icon: <IconBuildingFactory2 size={18} />,
            accent: 'text-blue-400/80',
            gradient: 'from-blue-500/15 to-cyan-500/10',
        },
        'research-lab': {
            icon: <IconFlask size={18} />,
            accent: 'text-purple-400/80',
            gradient: 'from-purple-500/15 to-pink-500/10',
        },
        macrohard: {
            icon: <IconUsers size={18} />,
            accent: 'text-sky-400/80',
            gradient: 'from-sky-500/15 to-indigo-500/10',
        },
    };

function GalleryCard({
    example,
    featured,
    busy,
    disabled,
    onInstall,
}: {
    example: GalleryExample;
    featured?: boolean;
    busy: boolean;
    disabled: boolean;
    onInstall: () => void;
}) {
    const theme = EXAMPLE_THEMES[example.slug] ?? {
        icon: <IconWorld size={18} />,
        accent: 'text-blue-400/80',
        gradient: 'from-blue-500/10 to-purple-500/10',
    };
    const firstParagraph = example.description.split('\n\n')[0] ?? example.description;

    return (
        <div
            className={`group relative flex h-full flex-col rounded-2xl border border-white/[0.08] bg-white/[0.02] p-5 transition-all duration-200 ${
                featured ? 'sm:col-span-2 lg:col-span-2' : ''
            } ${disabled ? 'opacity-50' : 'hover:border-white/[0.15] hover:bg-white/[0.04] hover:shadow-lg hover:shadow-white/[0.02]'}`}
        >
            <div className="flex items-start gap-3">
                <div
                    className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-white/[0.08] bg-gradient-to-br ${theme.gradient}`}
                >
                    <span className={theme.accent}>{theme.icon}</span>
                </div>
                <div className="min-w-0 flex-1">
                    <h3 className="font-heading text-sm tracking-wide text-foreground/95">
                        {example.name}
                    </h3>
                    <p className="text-[11px] text-muted-foreground/60">{example.tagline}</p>
                </div>
            </div>

            <p
                className={`mt-3 text-[11px] leading-relaxed text-muted-foreground/70 ${featured ? 'line-clamp-4' : 'line-clamp-3'}`}
            >
                {firstParagraph}
            </p>

            <div className="mt-3 flex flex-wrap gap-1.5">
                {example.agents.map((a) => (
                    <span
                        className="inline-flex items-center gap-1 rounded-full border border-white/[0.08] bg-white/[0.04] px-2 py-0.5 text-[10px] font-mono text-muted-foreground/70"
                        key={a}
                    >
                        <IconUser className="opacity-50" size={9} />
                        {a}
                    </span>
                ))}
            </div>

            {example.command && (
                <div className="mt-3 rounded-lg bg-white/[0.03] border border-white/[0.06] px-3 py-1.5">
                    <code className="text-[10px] font-mono text-muted-foreground/50 leading-relaxed">
                        $ {example.command.split('\n')[0]}
                    </code>
                </div>
            )}

            <div className="flex-1" />

            <button
                className={`mt-4 inline-flex items-center justify-center gap-1.5 rounded-lg border px-3 py-2 text-xs font-medium transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-60 ${
                    featured
                        ? 'border-white/[0.15] bg-white/[0.08] text-foreground/95 hover:border-white/[0.25] hover:bg-white/[0.14]'
                        : 'border-white/[0.10] bg-white/[0.06] text-foreground/90 hover:border-white/[0.18] hover:bg-white/[0.10]'
                }`}
                disabled={disabled || busy}
                onClick={onInstall}
                type="button"
            >
                {busy ? (
                    <>
                        <IconLoader2 className="animate-spin" size={13} />
                        Spawning…
                    </>
                ) : (
                    <>
                        <IconRocket size={13} />
                        Install &amp; spawn
                        <IconArrowRight className="ml-0.5 opacity-60" size={12} />
                    </>
                )}
            </button>
        </div>
    );
}

function QuickStartWizard({ onComplete }: { onComplete: () => void }) {
    const router = useRouter();
    const [step, setStep] = useState(1);
    const [agentName, setAgentName] = useState('');
    const [purpose, setPurpose] = useState('');
    const [workspace, setWorkspace] = useState('');
    const [error, setError] = useState('');
    const [working, setWorking] = useState(false);

    const spawnProgressMessage = useProgressMessages(working && step === 3, [
        { after: 0, text: 'Creating world...' },
        { after: 5, text: 'Building Docker image (first run could take a few minutes)...' },
        { after: 30, text: 'Still building... installing dependencies...' },
        { after: 60, text: 'Almost there...' },
    ]);

    const handleCreateAgent = async () => {
        if (!agentName.trim()) {
            return;
        }
        setWorking(true);
        setError('');
        try {
            const result = await apiAction('/api/agents', { name: agentName.trim() });
            if (!result.ok) {
                setError(result.error || 'Failed to create agent');
                setWorking(false);
                return;
            }
            setStep(2);
        } catch {
            setError('Failed to connect to API');
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
        setError('');
        const effectiveWorkspace =
            workspace.trim() ||
            `/tmp/spwn-${agentName.trim()}-${Math.random().toString(36).substring(2, 6)}`;
        try {
            const res = await fetch(goApiUrl('/api/worlds'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    agent: agentName.trim(),
                    workspaces: [{ name: 'default', path: effectiveWorkspace }],
                    config: 'default',
                    role: 'worker',
                }),
                signal: AbortSignal.timeout(600_000), // 10 min - first run may build Docker images
            });
            const data = await res.json().catch(() => ({}));
            if (!res.ok) {
                setError(data.error || 'Failed to spawn world');
                setWorking(false);
                return;
            }
            onComplete();
            if (data.id) {
                router.push(`/world/${data.id}`);
            }
        } catch {
            setError('Failed to connect to API');
            setWorking(false);
        }
    };

    const steps = [
        { num: 1, label: 'Create Agent', icon: <IconUser size={14} /> },
        { num: 2, label: 'Set Purpose', icon: <IconBulb size={14} /> },
        { num: 3, label: 'New World', icon: <IconWorld size={14} /> },
    ];

    return (
        <div className="w-full max-w-lg mx-auto px-4">
            {/* Header */}
            <div className="text-center mb-8">
                <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-blue-500/20 to-purple-500/20 border border-white/[0.08] flex items-center justify-center mx-auto mb-4">
                    <IconSparkles className="text-blue-400/60" size={28} />
                </div>
                <h2 className="text-xl font-heading text-foreground/90">Get started</h2>
                <p className="text-xs text-muted-foreground/40 mt-1 font-mono">
                    Create an agent, give it a purpose, and spawn a world.
                </p>
            </div>

            {/* Step indicators */}
            <div className="flex items-center justify-center gap-2 mb-8">
                {steps.map((s, i) => {
                    let stepClass: string;
                    if (step > s.num) {
                        stepClass = 'bg-green-500/15 text-green-400/80 border border-green-500/20';
                    } else if (step === s.num) {
                        stepClass = 'bg-white/[0.08] text-foreground/70 border border-white/[0.12]';
                    } else {
                        stepClass =
                            'bg-white/[0.02] text-muted-foreground/25 border border-white/[0.04]';
                    }
                    return (
                        <div className="flex items-center gap-2" key={s.num}>
                            <div
                                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-[10px] font-mono transition-all ${stepClass}`}
                            >
                                {step > s.num ? <IconCheck size={10} /> : s.icon}
                                {s.label}
                            </div>
                            {i < steps.length - 1 && (
                                <IconArrowRight className="text-muted-foreground/15" size={10} />
                            )}
                        </div>
                    );
                })}
            </div>

            {/* Step content */}
            <div className="glass-subtle rounded-2xl p-6 space-y-4">
                {step === 1 && (
                    <>
                        <div>
                            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-2">
                                Agent name
                            </label>
                            <input
                                autoFocus
                                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-4 py-3 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                                onChange={(e) => setAgentName(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleCreateAgent();
                                    }
                                }}
                                placeholder="e.g. atlas, neo, morpheus..."
                                value={agentName}
                            />
                            <p className="text-[10px] text-muted-foreground/25 mt-2">
                                Agents are autonomous AI entities that work inside worlds
                            </p>
                        </div>
                        <button
                            className="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed"
                            disabled={!agentName.trim() || working}
                            onClick={handleCreateAgent}
                        >
                            {working ? (
                                <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                            ) : (
                                <IconArrowRight size={16} />
                            )}
                            {working ? 'Creating...' : 'Create Agent'}
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
                                autoFocus
                                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-4 py-3 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors resize-none"
                                onChange={(e) => setPurpose(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' && !e.shiftKey) {
                                        e.preventDefault();
                                        handleSetPurpose();
                                    }
                                }}
                                placeholder="e.g. Build a REST API, Manage my infrastructure, Write documentation..."
                                rows={3}
                                value={purpose}
                            />
                            <p className="text-[10px] text-muted-foreground/25 mt-2">
                                Optional - you can always change this later
                            </p>
                        </div>
                        <button
                            className="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all"
                            onClick={handleSetPurpose}
                        >
                            <IconArrowRight size={16} />
                            {purpose.trim() ? 'Continue' : 'Skip for now'}
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
                                autoFocus
                                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-4 py-3 text-sm font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                                onChange={(e) => setWorkspace(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleSpawnWorld();
                                    }
                                }}
                                placeholder={`/tmp/spwn-${agentName.trim() || 'agent'}`}
                                value={workspace}
                            />
                            <p className="text-[10px] text-muted-foreground/25 mt-2">
                                The directory where {agentName} will work - leave empty for default
                            </p>
                        </div>

                        {/* Preview */}
                        <div className="rounded-lg bg-white/[0.02] border border-white/[0.05] px-3 py-3">
                            <p className="text-[10px] uppercase tracking-widest text-muted-foreground/30 mb-1">
                                Summary
                            </p>
                            <div className="text-[11px] text-muted-foreground/40 space-y-0.5">
                                <p>
                                    → Agent:{' '}
                                    <span className="text-foreground/60 font-mono">
                                        {agentName}
                                    </span>
                                </p>
                                {purpose && (
                                    <p>
                                        → Purpose:{' '}
                                        <span className="text-foreground/60">{purpose}</span>
                                    </p>
                                )}
                                <p>
                                    → Workspace:{' '}
                                    <span className="font-mono text-foreground/60">
                                        {workspace || `/tmp/spwn-${agentName.trim()}`}
                                    </span>
                                </p>
                            </div>
                        </div>

                        <button
                            className={`w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed ${working ? 'animate-pulse' : ''}`}
                            disabled={working}
                            onClick={handleSpawnWorld}
                        >
                            {working ? (
                                <>
                                    <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                                    Spawning...
                                </>
                            ) : (
                                <>
                                    <IconRocket size={16} />
                                    New World
                                </>
                            )}
                        </button>
                        <ProgressShimmer active={working} message={spawnProgressMessage} />
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

/* ── New World Dialog ── */

interface SpawnAgentListItem {
    name: string;
    path: string;
    layers: Record<string, string[]>;
}

interface WorkspaceDraft {
    id: string;
    name: string;
    path: string;
    readonly: boolean;
}

let workspaceDraftCounter = 0;
function newWorkspaceDraft(
    init: Omit<WorkspaceDraft, 'id'> = { name: 'default', path: '', readonly: false },
): WorkspaceDraft {
    workspaceDraftCounter += 1;
    return { id: `ws-${workspaceDraftCounter}`, ...init };
}

function SpawnWorldDialog({
    onClose,
    onComplete,
}: {
    onClose: () => void;
    onComplete: () => void;
}) {
    const router = useRouter();
    const [worldName, setWorldName] = useState('');
    const [selectedAgents, setSelectedAgents] = useState<Set<string>>(new Set());
    const [workspaces, setWorkspaces] = useState<WorkspaceDraft[]>(() => [newWorkspaceDraft()]);
    const [config, setConfig] = useState('default');
    const [role, setRole] = useState('worker');
    const [spawning, setSpawning] = useState(false);
    const [availableAgents, setAvailableAgents] = useState<SpawnAgentListItem[]>([]);
    const [error, setError] = useState('');
    const [creatingAgent, setCreatingAgent] = useState(false);
    const [newAgentName, setNewAgentName] = useState('');

    const spawnProgressMessage = useProgressMessages(spawning, [
        { after: 0, text: 'Creating world...' },
        { after: 5, text: 'Building Docker image (first run could take a few minutes)...' },
        { after: 30, text: 'Still building... installing dependencies...' },
        { after: 60, text: 'Almost there...' },
    ]);

    // Generate a sensible default workspace path. Uses the first selected
    // Agent's name when one exists, else a generic suffix.
    const defaultWorkspacePath = useMemo(() => {
        const first = [...selectedAgents][0];
        const rand = Math.random().toString(36).substring(2, 6);
        return first ? `/tmp/spwn-${first}-${rand}` : `/tmp/spwn-workspace-${rand}`;
    }, [selectedAgents]);

    // Fetch available agents for the checkable list
    useEffect(() => {
        apiGet<SpawnAgentListItem[]>('/api/agents')
            .then((agents) => setAvailableAgents(agents ?? []))
            .catch(() => {});
    }, []);

    const toggleAgent = (name: string) => {
        setSelectedAgents((prev) => {
            const next = new Set(prev);
            if (next.has(name)) {
                next.delete(name);
            } else {
                next.add(name);
            }
            return next;
        });
    };

    const handleCreateInlineAgent = async () => {
        if (!newAgentName.trim()) {
            return;
        }
        setCreatingAgent(true);
        setError('');
        try {
            const result = await apiAction('/api/agents', { name: newAgentName.trim() });
            if (!result.ok) {
                setError(result.error || 'Failed to create agent');
                return;
            }
            const name = newAgentName.trim();
            const created = { name, path: '', layers: {} };
            setAvailableAgents((prev) => [...prev, created]);
            setSelectedAgents((prev) => new Set(prev).add(name));
            setNewAgentName('');
        } catch {
            setError('Failed to connect to API');
        } finally {
            setCreatingAgent(false);
        }
    };

    const handleSpawn = async () => {
        setSpawning(true);
        setError('');
        // Filter out blank rows and fill in defaults. A fully empty list = ephemeral world.
        const cleanWorkspaces = workspaces
            .map((w, i) => ({
                name: w.name.trim() || (workspaces.length === 1 ? 'default' : `w${i}`),
                path: w.path.trim() || (i === 0 ? defaultWorkspacePath : ''),
                readonly: w.readonly,
            }))
            .filter((w) => w.path !== '');
        try {
            const res = await fetch(goApiUrl('/api/worlds'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: worldName.trim(),
                    agents: [...selectedAgents].map((n) => ({ name: n, role })),
                    workspaces: cleanWorkspaces,
                    config,
                    role,
                }),
                signal: AbortSignal.timeout(600_000), // 10 min - first run may build Docker images
            });
            const data = await res.json().catch(() => ({}));
            if (!res.ok) {
                setError(data.error || 'Failed to spawn world');
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
            setError('Failed to connect to API');
            setSpawning(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            {/* Backdrop */}
            <div
                className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                onClick={() => !spawning && onClose()}
            />

            {/* Dialog */}
            <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl overflow-hidden">
                {/* Top shimmer bar */}
                {spawning && (
                    <div className="w-full h-0.5 overflow-hidden bg-white/[0.04]">
                        <div
                            className="h-full w-1/3 rounded-full bg-emerald-500/30"
                            style={{ animation: 'progressSlide 1.5s ease-in-out infinite' }}
                        />
                    </div>
                )}
                {/* Header */}
                <div className="px-6 pt-6 pb-4 flex items-center justify-between">
                    <div>
                        <h2 className="text-lg font-heading text-foreground/90">New World</h2>
                        <p className="text-[11px] text-muted-foreground/40 mt-0.5">
                            Create a new isolated world for your agent
                        </p>
                    </div>
                    <button
                        className="text-muted-foreground/40 hover:text-foreground/60 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                        disabled={spawning}
                        onClick={onClose}
                    >
                        <IconX size={18} />
                    </button>
                </div>

                {/* Form */}
                <div className="px-6 pb-6 space-y-4">
                    {/* World name (optional) */}
                    <div>
                        <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                            Name{' '}
                            <span className="text-muted-foreground/25 normal-case tracking-normal">
                                (optional)
                            </span>
                        </label>
                        <input
                            className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                            onChange={(e) => setWorldName(e.target.value)}
                            placeholder="My Project"
                            value={worldName}
                        />
                    </div>

                    {/* Agents - checkable list, optional (0 = empty world) */}
                    <div>
                        <div className="flex items-baseline justify-between mb-1.5">
                            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40">
                                Agents{' '}
                                <span className="text-muted-foreground/25 normal-case tracking-normal">
                                    (
                                    {selectedAgents.size === 0
                                        ? 'none - empty world'
                                        : `${selectedAgents.size} selected`}
                                    )
                                </span>
                            </label>
                            {selectedAgents.size > 0 && (
                                <button
                                    className="text-[10px] text-muted-foreground/40 hover:text-foreground/70 transition-colors"
                                    onClick={() => setSelectedAgents(new Set())}
                                    type="button"
                                >
                                    Clear
                                </button>
                            )}
                        </div>
                        <div className="rounded-lg bg-white/[0.02] border border-white/[0.08] max-h-44 overflow-y-auto">
                            {availableAgents.length === 0 ? (
                                <p className="text-[11px] text-muted-foreground/40 px-3 py-2.5">
                                    No agents yet - create one below or skip to spawn an empty
                                    world.
                                </p>
                            ) : (
                                <ul className="divide-y divide-white/[0.04]">
                                    {availableAgents.map((a) => {
                                        const checked = selectedAgents.has(a.name);
                                        return (
                                            <li key={a.name}>
                                                <label className="flex items-center gap-2.5 px-3 py-2 cursor-pointer hover:bg-white/[0.03] transition-colors">
                                                    <input
                                                        checked={checked}
                                                        className="w-3.5 h-3.5 rounded border-white/[0.15] bg-white/[0.04] accent-foreground cursor-pointer"
                                                        onChange={() => toggleAgent(a.name)}
                                                        type="checkbox"
                                                    />
                                                    <span
                                                        className={`text-sm font-mono ${checked ? 'text-foreground/90' : 'text-foreground/60'}`}
                                                    >
                                                        {a.name}
                                                    </span>
                                                </label>
                                            </li>
                                        );
                                    })}
                                </ul>
                            )}
                            {/* Inline "add new" row */}
                            <div className="flex gap-2 px-3 py-2 border-t border-white/[0.06]">
                                <input
                                    className="flex-1 bg-transparent text-xs text-foreground/80 placeholder:text-muted-foreground/30 focus:outline-none"
                                    onChange={(e) => setNewAgentName(e.target.value)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter') {
                                            handleCreateInlineAgent();
                                        }
                                    }}
                                    placeholder="New agent name…"
                                    value={newAgentName}
                                />
                                <button
                                    className="shrink-0 px-2.5 py-1 rounded text-[11px] bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed"
                                    disabled={!newAgentName.trim() || creatingAgent}
                                    onClick={handleCreateInlineAgent}
                                    type="button"
                                >
                                    {creatingAgent ? '…' : '+ Add'}
                                </button>
                            </div>
                        </div>
                    </div>

                    {/* Workspaces */}
                    <div>
                        <div className="flex items-baseline justify-between mb-1.5">
                            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40">
                                Workspaces{' '}
                                {workspaces.length === 0 && (
                                    <span className="text-muted-foreground/25 normal-case tracking-normal">
                                        (ephemeral)
                                    </span>
                                )}
                            </label>
                            <button
                                className="text-[10px] text-muted-foreground/50 hover:text-foreground/80 transition-colors"
                                onClick={() =>
                                    setWorkspaces((prev) => [
                                        ...prev,
                                        newWorkspaceDraft({
                                            name: prev.length === 0 ? 'default' : `w${prev.length}`,
                                            path: '',
                                            readonly: false,
                                        }),
                                    ])
                                }
                                type="button"
                            >
                                + Add
                            </button>
                        </div>
                        {workspaces.length === 0 ? (
                            <button
                                className="w-full text-left px-3 py-2.5 rounded-lg bg-white/[0.02] border border-dashed border-white/[0.08] text-[11px] text-muted-foreground/40 hover:text-foreground/60 hover:border-white/[0.15] transition-colors"
                                onClick={() => setWorkspaces([newWorkspaceDraft()])}
                                type="button"
                            >
                                Ephemeral world - click to add a host mount
                            </button>
                        ) : (
                            <div className="space-y-2">
                                {workspaces.map((ws, idx) => (
                                    <div className="flex gap-1.5 items-center" key={ws.id}>
                                        <input
                                            className="w-24 shrink-0 bg-white/[0.03] border border-white/[0.08] rounded-lg px-2.5 py-2 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                                            onChange={(e) =>
                                                setWorkspaces((prev) =>
                                                    prev.map((w, i) =>
                                                        i === idx
                                                            ? { ...w, name: e.target.value }
                                                            : w,
                                                    ),
                                                )
                                            }
                                            placeholder="name"
                                            value={ws.name}
                                        />
                                        <input
                                            className="flex-1 min-w-0 bg-white/[0.03] border border-white/[0.08] rounded-lg px-2.5 py-2 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                                            onChange={(e) =>
                                                setWorkspaces((prev) =>
                                                    prev.map((w, i) =>
                                                        i === idx
                                                            ? { ...w, path: e.target.value }
                                                            : w,
                                                    ),
                                                )
                                            }
                                            placeholder={
                                                idx === 0 ? defaultWorkspacePath : '/path/to/dir'
                                            }
                                            value={ws.path}
                                        />
                                        <button
                                            className={`shrink-0 w-8 h-8 flex items-center justify-center rounded-lg text-[10px] font-mono transition-colors ${ws.readonly ? 'bg-amber-500/15 border border-amber-500/25 text-amber-300' : 'bg-white/[0.03] border border-white/[0.08] text-muted-foreground/40 hover:text-foreground/70'}`}
                                            onClick={() =>
                                                setWorkspaces((prev) =>
                                                    prev.map((w, i) =>
                                                        i === idx
                                                            ? { ...w, readonly: !w.readonly }
                                                            : w,
                                                    ),
                                                )
                                            }
                                            title={ws.readonly ? 'Read-only' : 'Read-write'}
                                            type="button"
                                        >
                                            ro
                                        </button>
                                        <button
                                            className="shrink-0 w-8 h-8 flex items-center justify-center rounded-lg text-muted-foreground/30 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                                            onClick={() =>
                                                setWorkspaces((prev) =>
                                                    prev.filter((_, i) => i !== idx),
                                                )
                                            }
                                            type="button"
                                        >
                                            <IconX size={14} />
                                        </button>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Config + Role row */}
                    <div className="grid grid-cols-2 gap-3">
                        <div>
                            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                                Config
                            </label>
                            <select
                                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
                                onChange={(e) => setConfig(e.target.value)}
                                value={config}
                            >
                                {AVAILABLE_CONFIGS.map((c) => (
                                    <option key={c} value={c}>
                                        {c}
                                    </option>
                                ))}
                            </select>
                        </div>
                        <div>
                            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                                Agent Role
                            </label>
                            <select
                                className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
                                onChange={(e) => setRole(e.target.value)}
                                value={role}
                            >
                                <option value="chief">Chief</option>
                                <option value="manager">Manager</option>
                                <option value="worker">Worker</option>
                            </select>
                        </div>
                    </div>

                    {/* Preview of what will happen */}
                    {(() => {
                        const wsFlags = workspaces
                            .filter((w) => w.path.trim())
                            .map((w) => `-w ${w.name}=${w.path.trim()}${w.readonly ? ':ro' : ''}`)
                            .join(' ');
                        const agentList = [...selectedAgents];
                        const agentFlags =
                            agentList.length > 0
                                ? agentList.map((n) => `-a ${n}`).join(' ')
                                : '--no-agent';
                        return (
                            <div className="rounded-lg bg-white/[0.02] border border-white/[0.05] px-3 py-3 space-y-2">
                                <p className="text-[10px] uppercase tracking-widest text-muted-foreground/30 mb-1">
                                    Preview
                                </p>
                                <div className="font-mono text-[11px] text-muted-foreground/35 break-all">
                                    spwn up {agentFlags} --role {role} --config {config}
                                    {wsFlags ? ` ${wsFlags}` : ' (ephemeral)'}
                                </div>
                                <div className="text-[10px] text-muted-foreground/25 space-y-0.5">
                                    <p>→ Creates isolated Docker container</p>
                                    {agentList.length === 0 ? (
                                        <p>→ Empty world (no agents deployed)</p>
                                    ) : (
                                        <p>
                                            → Deploys {agentList.length} agent
                                            {agentList.length === 1 ? '' : 's'}:{' '}
                                            <span className="font-mono">
                                                {agentList.join(', ')}
                                            </span>
                                        </p>
                                    )}
                                    {workspaces.filter((w) => w.path.trim()).length === 0 ? (
                                        <p>→ No host workspace (uses image&apos;s /workspace)</p>
                                    ) : (
                                        workspaces
                                            .filter((w) => w.path.trim())
                                            .map((w) => (
                                                <p key={w.id}>
                                                    → {w.name}:{' '}
                                                    <span className="font-mono">
                                                        {w.path.trim()}
                                                    </span>
                                                    {w.readonly ? ' (read-only)' : ''}
                                                </p>
                                            ))
                                    )}
                                </div>
                            </div>
                        );
                    })()}

                    {/* Error display */}
                    {error && (
                        <div className="rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2 text-xs text-red-400 font-mono">
                            {error}
                        </div>
                    )}

                    {/* Spawn button */}
                    <button
                        className={`w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] hover:text-foreground/90 border border-white/[0.08] transition-all disabled:opacity-30 disabled:cursor-not-allowed ${spawning ? 'animate-pulse' : ''}`}
                        disabled={spawning}
                        onClick={handleSpawn}
                    >
                        {spawning ? (
                            <>
                                <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                                Spawning...
                            </>
                        ) : (
                            <>
                                <IconRocket size={16} />
                                New World
                            </>
                        )}
                    </button>
                    <ProgressShimmer active={spawning} message={spawnProgressMessage} />
                </div>
            </div>
        </div>
    );
}
