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

import { apiAction, apiDelete, apiGet, apiPost, goApiUrl } from '@/api/client';
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
import { AVAILABLE_CONFIGS, getWorldName } from '@/domain/model';
import type { World } from '@/domain/model';
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts';
import { usePageTitle } from '@/hooks/use-page-title';
import { useProgressMessages } from '@/hooks/use-progress-messages';

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
                const world = worlds[selected];
                if (world) {
                    router.push(`/world/${world.id}`);
                }
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
        const tmove = (e: TouchEvent) => {
            const touch = e.touches[0];
            if (touch) {
                onDragMove(touch.clientX);
            }
        };
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
        <Page className="flex h-full flex-col">
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
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
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
                            className="relative -mx-6 min-h-[320px] flex-1 overflow-hidden md:-mx-8"
                            onClick={(e) => {
                                if (e.target === e.currentTarget && selected !== null) {
                                    setSelected(null);
                                }
                            }}
                        >
                            <div className="flex h-full items-center pb-24">
                                {/* Planets - full width scrollable */}
                                <div
                                    className="flex items-center gap-10 will-change-transform select-none"
                                    onMouseDown={(e) => {
                                        e.preventDefault();
                                        onDragStart(e.clientX);
                                    }}
                                    onTouchStart={(e) => {
                                        const touch = e.touches[0];
                                        if (touch) {
                                            onDragStart(touch.clientX);
                                        }
                                    }}
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
                                                className="flex shrink-0 cursor-pointer flex-col items-center"
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
                                        opacity={selected === null ? 0.5 : 0.2}
                                        scale={selected === null ? 1 : 0.85}
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
                                        <div className="pointer-events-none absolute inset-y-0 right-6 z-10 flex w-[340px] items-center pb-24 md:right-8">
                                            <div
                                                className="border-foreground/[0.08] bg-foreground/[0.04] animate-in fade-in slide-in-from-right-12 pointer-events-auto w-full overflow-hidden rounded-2xl border shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] backdrop-blur-md duration-500 ease-out dark:border-white/[0.1] dark:bg-white/[0.05] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)]"
                                                ref={panelRef}
                                            >
                                                <div className="space-y-5 p-5">
                                                    {/* Header */}
                                                    <div className="flex items-center justify-between">
                                                        <div className="flex min-w-0 items-center gap-3">
                                                            <WorldPlanet size="md" world={w} />
                                                            <div className="min-w-0">
                                                                <h3 className="text-foreground/95 truncate font-mono text-sm font-bold">
                                                                    {name}
                                                                </h3>
                                                                <p className="text-muted-foreground/35 truncate font-mono text-[10px]">
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
                                                            className="text-muted-foreground/30 hover:text-foreground/70 flex h-7 w-7 shrink-0 items-center justify-center rounded-full transition-colors"
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
                                                            className="text-foreground/70 hover:text-foreground/95 flex h-9 flex-1 items-center justify-center gap-2 rounded-full border border-white/[0.06] bg-white/[0.06] font-mono text-xs font-medium transition-all hover:border-white/[0.12] hover:bg-white/[0.1]"
                                                            onClick={() =>
                                                                router.push(`/world/${w.id}`)
                                                            }
                                                        >
                                                            Enter World
                                                            <IconArrowRight size={13} />
                                                        </button>
                                                        {isRunning && (
                                                            <button
                                                                className="text-muted-foreground/30 h-9 rounded-full border border-transparent px-3.5 font-mono text-[11px] transition-all hover:border-red-500/15 hover:bg-red-500/[0.06] hover:text-red-400"
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
                    <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-sm rounded-2xl border border-red-500/30 p-6 shadow-2xl backdrop-blur-md">
                        <div className="flex flex-col items-center text-center">
                            <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl border border-red-500/20 bg-red-500/10">
                                <IconAlertTriangle className="text-red-400" size={28} />
                            </div>
                            <h2 className="font-heading mb-2 text-lg text-red-300">
                                Destroy All Worlds?
                            </h2>
                            <p className="mb-1 text-xs text-red-300/60">
                                This will permanently destroy{' '}
                                <span className="font-mono font-bold">{worlds.length}</span> world
                                {worlds.length === 1 ? '' : 's'} and all their agents.
                            </p>
                            <p className="mb-6 text-xs text-red-300/40">
                                This action cannot be undone.
                            </p>
                            <div className="flex w-full gap-3">
                                <button
                                    className="text-muted-foreground/50 hover:text-foreground/70 flex-1 rounded-xl px-4 py-2.5 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-30"
                                    disabled={destroyingAll}
                                    onClick={() => setShowDestroyAll(false)}
                                >
                                    Cancel
                                </button>
                                <button
                                    className="flex-1 rounded-xl border border-red-500/30 bg-red-500/20 px-4 py-2.5 text-sm text-red-300 transition-colors hover:bg-red-500/30 disabled:opacity-50"
                                    disabled={destroyingAll}
                                    onClick={handleDestroyAll}
                                >
                                    {destroyingAll ? (
                                        <span className="flex items-center justify-center gap-2">
                                            <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-red-300/30 border-t-red-300/70" />
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
                            <span className="block translate-y-[0.5px] leading-none">
                                {item.icon}
                            </span>
                        </span>
                        <span className="flex items-baseline gap-1.5 self-center whitespace-nowrap">
                            <span className="font-mono text-[12px] leading-none font-medium">
                                {item.value}
                            </span>
                            <span
                                className={`text-[9px] leading-none font-medium uppercase ${item.labelClass}`}
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
        <div className="flex min-h-[400px] flex-1 items-start justify-center px-4 pt-12 pb-16">
            <div className="w-full max-w-5xl">
                <div className="mb-10 text-center">
                    <div className="text-muted-foreground/70 mb-4 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 text-[10px] tracking-wider uppercase">
                        <IconSparkles size={11} />
                        Start from a template
                    </div>
                    <h2 className="font-heading text-foreground/90 text-2xl tracking-wide">
                        {hasAgents
                            ? 'Give your agents a world to work in'
                            : 'Pick a template and spawn in one click'}
                    </h2>
                    <p className="text-muted-foreground/60 mx-auto mt-2 max-w-lg text-sm">
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
                    <p className="text-muted-foreground/60 text-center text-sm">
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
                                onInstall={async () => await handleInstallAndSpawn(ex)}
                            />
                        ))}
                    </div>
                )}

                {installError && (
                    <p className="mt-4 text-center text-xs text-red-300/80">{installError}</p>
                )}

                <div className="mt-10 flex flex-col items-center gap-2">
                    <button
                        className="text-muted-foreground/50 hover:text-foreground/80 inline-flex items-center gap-2 text-[11px] tracking-wider uppercase transition-colors"
                        onClick={onSpawn}
                    >
                        <IconRocket size={12} />
                        Or build your own world from scratch
                    </button>
                    <div className="text-muted-foreground/30 flex items-center gap-2 font-mono text-[10px]">
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
                    <h3 className="font-heading text-foreground/95 text-sm tracking-wide">
                        {example.name}
                    </h3>
                    <p className="text-muted-foreground/60 text-[11px]">{example.tagline}</p>
                </div>
            </div>

            <p
                className={`text-muted-foreground/70 mt-3 text-[11px] leading-relaxed ${featured ? 'line-clamp-4' : 'line-clamp-3'}`}
            >
                {firstParagraph}
            </p>

            <div className="mt-3 flex flex-wrap gap-1.5">
                {example.agents.map((a) => (
                    <span
                        className="text-muted-foreground/70 inline-flex items-center gap-1 rounded-full border border-white/[0.08] bg-white/[0.04] px-2 py-0.5 font-mono text-[10px]"
                        key={a}
                    >
                        <IconUser className="opacity-50" size={9} />
                        {a}
                    </span>
                ))}
            </div>

            {example.command && (
                <div className="mt-3 rounded-lg border border-white/[0.06] bg-white/[0.03] px-3 py-1.5">
                    <code className="text-muted-foreground/50 font-mono text-[10px] leading-relaxed">
                        $ {example.command.split('\n')[0]}
                    </code>
                </div>
            )}

            <div className="flex-1" />

            <button
                className={`mt-4 inline-flex items-center justify-center gap-1.5 rounded-lg border px-3 py-2 text-xs font-medium transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-60 ${
                    featured
                        ? 'text-foreground/95 border-white/[0.15] bg-white/[0.08] hover:border-white/[0.25] hover:bg-white/[0.14]'
                        : 'text-foreground/90 border-white/[0.10] bg-white/[0.06] hover:border-white/[0.18] hover:bg-white/[0.10]'
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
            `/tmp/spwn-${agentName.trim()}-${Math.random().toString(36).slice(2, 6)}`;
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
        <div className="mx-auto w-full max-w-lg px-4">
            {/* Header */}
            <div className="mb-8 text-center">
                <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.08] bg-gradient-to-br from-blue-500/20 to-purple-500/20">
                    <IconSparkles className="text-blue-400/60" size={28} />
                </div>
                <h2 className="font-heading text-foreground/90 text-xl">Get started</h2>
                <p className="text-muted-foreground/40 mt-1 font-mono text-xs">
                    Create an agent, give it a purpose, and spawn a world.
                </p>
            </div>

            {/* Step indicators */}
            <div className="mb-8 flex items-center justify-center gap-2">
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
                                className={`flex items-center gap-1.5 rounded-full px-3 py-1.5 font-mono text-[10px] transition-all ${stepClass}`}
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
            <div className="glass-subtle space-y-4 rounded-2xl p-6">
                {step === 1 && (
                    <>
                        <div>
                            <label className="text-muted-foreground/40 mb-2 block text-[10px] tracking-widest uppercase">
                                Agent name
                            </label>
                            <input
                                autoFocus
                                className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-4 py-3 text-sm transition-colors focus:border-white/[0.15] focus:outline-none"
                                onChange={(e) => setAgentName(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleCreateAgent();
                                    }
                                }}
                                placeholder="e.g. atlas, neo, morpheus..."
                                value={agentName}
                            />
                            <p className="text-muted-foreground/25 mt-2 text-[10px]">
                                Agents are autonomous AI entities that work inside worlds
                            </p>
                        </div>
                        <button
                            className="text-foreground/70 flex w-full items-center justify-center gap-2 rounded-xl border border-white/[0.08] bg-white/[0.06] py-3 text-sm font-medium transition-all hover:bg-white/[0.1] disabled:cursor-not-allowed disabled:opacity-30"
                            disabled={!agentName.trim() || working}
                            onClick={handleCreateAgent}
                        >
                            {working ? (
                                <div className="border-foreground/30 border-t-foreground/70 h-3.5 w-3.5 animate-spin rounded-full border-2" />
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
                            <label className="text-muted-foreground/40 mb-2 block text-[10px] tracking-widest uppercase">
                                What should {agentName} do?
                            </label>
                            <textarea
                                autoFocus
                                className="text-foreground/80 placeholder:text-muted-foreground/25 w-full resize-none rounded-lg border border-white/[0.08] bg-white/[0.03] px-4 py-3 text-sm transition-colors focus:border-white/[0.15] focus:outline-none"
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
                            <p className="text-muted-foreground/25 mt-2 text-[10px]">
                                Optional - you can always change this later
                            </p>
                        </div>
                        <button
                            className="text-foreground/70 flex w-full items-center justify-center gap-2 rounded-xl border border-white/[0.08] bg-white/[0.06] py-3 text-sm font-medium transition-all hover:bg-white/[0.1]"
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
                            <label className="text-muted-foreground/40 mb-2 block text-[10px] tracking-widest uppercase">
                                Workspace path
                            </label>
                            <input
                                autoFocus
                                className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-4 py-3 font-mono text-sm transition-colors focus:border-white/[0.15] focus:outline-none"
                                onChange={(e) => setWorkspace(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleSpawnWorld();
                                    }
                                }}
                                placeholder={`/tmp/spwn-${agentName.trim() || 'agent'}`}
                                value={workspace}
                            />
                            <p className="text-muted-foreground/25 mt-2 text-[10px]">
                                The directory where {agentName} will work - leave empty for default
                            </p>
                        </div>

                        {/* Preview */}
                        <div className="rounded-lg border border-white/[0.05] bg-white/[0.02] px-3 py-3">
                            <p className="text-muted-foreground/30 mb-1 text-[10px] tracking-widest uppercase">
                                Summary
                            </p>
                            <div className="text-muted-foreground/40 space-y-0.5 text-[11px]">
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
                                    <span className="text-foreground/60 font-mono">
                                        {workspace || `/tmp/spwn-${agentName.trim()}`}
                                    </span>
                                </p>
                            </div>
                        </div>

                        <button
                            className={`text-foreground/70 flex w-full items-center justify-center gap-2 rounded-xl border border-white/[0.08] bg-white/[0.06] py-3 text-sm font-medium transition-all hover:bg-white/[0.1] disabled:cursor-not-allowed disabled:opacity-30 ${working ? 'animate-pulse' : ''}`}
                            disabled={working}
                            onClick={handleSpawnWorld}
                        >
                            {working ? (
                                <>
                                    <div className="border-foreground/30 border-t-foreground/70 h-3.5 w-3.5 animate-spin rounded-full border-2" />
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
                    <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 font-mono text-xs text-red-400">
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
        const rand = Math.random().toString(36).slice(2, 6);
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
            <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-md overflow-hidden rounded-2xl border border-white/[0.08] shadow-2xl backdrop-blur-md">
                {/* Top shimmer bar */}
                {spawning && (
                    <div className="h-0.5 w-full overflow-hidden bg-white/[0.04]">
                        <div
                            className="h-full w-1/3 rounded-full bg-emerald-500/30"
                            style={{ animation: 'progressSlide 1.5s ease-in-out infinite' }}
                        />
                    </div>
                )}
                {/* Header */}
                <div className="flex items-center justify-between px-6 pt-6 pb-4">
                    <div>
                        <h2 className="font-heading text-foreground/90 text-lg">New World</h2>
                        <p className="text-muted-foreground/40 mt-0.5 text-[11px]">
                            Create a new isolated world for your agent
                        </p>
                    </div>
                    <button
                        className="text-muted-foreground/40 hover:text-foreground/60 transition-colors disabled:cursor-not-allowed disabled:opacity-30"
                        disabled={spawning}
                        onClick={onClose}
                    >
                        <IconX size={18} />
                    </button>
                </div>

                {/* Form */}
                <div className="space-y-4 px-6 pb-6">
                    {/* World name (optional) */}
                    <div>
                        <label className="text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase">
                            Name{' '}
                            <span className="text-muted-foreground/25 tracking-normal normal-case">
                                (optional)
                            </span>
                        </label>
                        <input
                            className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.15] focus:outline-none"
                            onChange={(e) => setWorldName(e.target.value)}
                            placeholder="My Project"
                            value={worldName}
                        />
                    </div>

                    {/* Agents - checkable list, optional (0 = empty world) */}
                    <div>
                        <div className="mb-1.5 flex items-baseline justify-between">
                            <label className="text-muted-foreground/40 text-[10px] tracking-widest uppercase">
                                Agents{' '}
                                <span className="text-muted-foreground/25 tracking-normal normal-case">
                                    (
                                    {selectedAgents.size === 0
                                        ? 'none - empty world'
                                        : `${selectedAgents.size} selected`}
                                    )
                                </span>
                            </label>
                            {selectedAgents.size > 0 && (
                                <button
                                    className="text-muted-foreground/40 hover:text-foreground/70 text-[10px] transition-colors"
                                    onClick={() => setSelectedAgents(new Set())}
                                    type="button"
                                >
                                    Clear
                                </button>
                            )}
                        </div>
                        <div className="max-h-44 overflow-y-auto rounded-lg border border-white/[0.08] bg-white/[0.02]">
                            {availableAgents.length === 0 ? (
                                <p className="text-muted-foreground/40 px-3 py-2.5 text-[11px]">
                                    No agents yet - create one below or skip to spawn an empty
                                    world.
                                </p>
                            ) : (
                                <ul className="divide-y divide-white/[0.04]">
                                    {availableAgents.map((a) => {
                                        const checked = selectedAgents.has(a.name);
                                        return (
                                            <li key={a.name}>
                                                <label className="flex cursor-pointer items-center gap-2.5 px-3 py-2 transition-colors hover:bg-white/[0.03]">
                                                    <input
                                                        checked={checked}
                                                        className="accent-foreground h-3.5 w-3.5 cursor-pointer rounded border-white/[0.15] bg-white/[0.04]"
                                                        onChange={() => toggleAgent(a.name)}
                                                        type="checkbox"
                                                    />
                                                    <span
                                                        className={`font-mono text-sm ${checked ? 'text-foreground/90' : 'text-foreground/60'}`}
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
                            <div className="flex gap-2 border-t border-white/[0.06] px-3 py-2">
                                <input
                                    className="text-foreground/80 placeholder:text-muted-foreground/30 flex-1 bg-transparent text-xs focus:outline-none"
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
                                    className="text-foreground/70 shrink-0 rounded border border-white/[0.08] bg-white/[0.06] px-2.5 py-1 text-[11px] transition-all hover:bg-white/[0.1] disabled:cursor-not-allowed disabled:opacity-30"
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
                        <div className="mb-1.5 flex items-baseline justify-between">
                            <label className="text-muted-foreground/40 text-[10px] tracking-widest uppercase">
                                Workspaces{' '}
                                {workspaces.length === 0 && (
                                    <span className="text-muted-foreground/25 tracking-normal normal-case">
                                        (ephemeral)
                                    </span>
                                )}
                            </label>
                            <button
                                className="text-muted-foreground/50 hover:text-foreground/80 text-[10px] transition-colors"
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
                                className="text-muted-foreground/40 hover:text-foreground/60 w-full rounded-lg border border-dashed border-white/[0.08] bg-white/[0.02] px-3 py-2.5 text-left text-[11px] transition-colors hover:border-white/[0.15]"
                                onClick={() => setWorkspaces([newWorkspaceDraft()])}
                                type="button"
                            >
                                Ephemeral world - click to add a host mount
                            </button>
                        ) : (
                            <div className="space-y-2">
                                {workspaces.map((ws, idx) => (
                                    <div className="flex items-center gap-1.5" key={ws.id}>
                                        <input
                                            className="text-foreground/80 placeholder:text-muted-foreground/25 w-24 shrink-0 rounded-lg border border-white/[0.08] bg-white/[0.03] px-2.5 py-2 font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none"
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
                                            className="text-foreground/80 placeholder:text-muted-foreground/25 min-w-0 flex-1 rounded-lg border border-white/[0.08] bg-white/[0.03] px-2.5 py-2 font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none"
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
                                            className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg font-mono text-[10px] transition-colors ${ws.readonly ? 'border border-amber-500/25 bg-amber-500/15 text-amber-300' : 'text-muted-foreground/40 hover:text-foreground/70 border border-white/[0.08] bg-white/[0.03]'}`}
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
                                            className="text-muted-foreground/30 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg transition-colors hover:bg-red-500/10 hover:text-red-400"
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
                            <label className="text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase">
                                Config
                            </label>
                            <select
                                className="text-foreground/80 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.15] focus:outline-none"
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
                            <label className="text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase">
                                Agent Role
                            </label>
                            <select
                                className="text-foreground/80 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.15] focus:outline-none"
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
                            <div className="space-y-2 rounded-lg border border-white/[0.05] bg-white/[0.02] px-3 py-3">
                                <p className="text-muted-foreground/30 mb-1 text-[10px] tracking-widest uppercase">
                                    Preview
                                </p>
                                <div className="text-muted-foreground/35 font-mono text-[11px] break-all">
                                    spwn up {agentFlags} --role {role} --config {config}
                                    {wsFlags ? ` ${wsFlags}` : ' (ephemeral)'}
                                </div>
                                <div className="text-muted-foreground/25 space-y-0.5 text-[10px]">
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
                        <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 font-mono text-xs text-red-400">
                            {error}
                        </div>
                    )}

                    {/* Spawn button */}
                    <button
                        className={`text-foreground/70 hover:text-foreground/90 flex w-full items-center justify-center gap-2 rounded-xl border border-white/[0.08] bg-white/[0.06] py-3 text-sm font-medium transition-all hover:bg-white/[0.1] disabled:cursor-not-allowed disabled:opacity-30 ${spawning ? 'animate-pulse' : ''}`}
                        disabled={spawning}
                        onClick={handleSpawn}
                    >
                        {spawning ? (
                            <>
                                <div className="border-foreground/30 border-t-foreground/70 h-3.5 w-3.5 animate-spin rounded-full border-2" />
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
