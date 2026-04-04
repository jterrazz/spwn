"use client";

import { useParams, useRouter } from "next/navigation";
import { useState, useEffect, useCallback, useRef } from "react";
import type { World } from "@/lib/types";
import { apiGet, apiAction, apiDelete, goApiUrl } from "@/lib/api-client";
import { streamChat } from "@/lib/stream-chat";
import type { ActivityBlock } from "@/lib/activity-types";
import { ActivityBlocksRenderer } from "@/components/activity-blocks";
import {
  IconTrash,
  IconCamera,
  IconFileText,
  IconPlayerPlay,
  IconDownload,
  IconX,
  IconAlertTriangle,
  IconRestore,
  IconPlus,
  IconSend,
  IconMessageCircle,
} from "@tabler/icons-react";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/page-header";
import { STATUS_DOT, STATUS_BADGE, TIER_BADGE } from "@/lib/status";
import { usePageTitle } from "@/hooks/use-page-title";
import { useToast } from "@/components/toast-provider";
import { useRefetch } from "@/components/app-shell";

function extractName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2 ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1) : id;
}

function timeAgo(iso: string): string {
  const d = Date.now() - new Date(iso).getTime();
  const m = Math.floor(d / 60000);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

const LOG_LEVEL_COLORS: Record<string, string> = {
  info: "text-blue-400/70",
  warn: "text-yellow-400/70",
  error: "text-red-400/70",
  debug: "text-muted-foreground/40",
};

function StatCard({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="glass-subtle px-5 py-4 flex-1 min-w-[140px]">
      <p className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-1">{label}</p>
      <p className="text-2xl font-heading text-foreground/90">{value}</p>
      {sub && <p className="text-[11px] font-mono text-muted-foreground/40 mt-0.5">{sub}</p>}
    </div>
  );
}

type Panel = null | "logs" | "snapshots";

export default function WorldDashboard() {
  const params = useParams();
  const router = useRouter();
  const worldId = params.id as string;
  const [world, setWorld] = useState<World | null>(null);
  const [loading, setLoading] = useState(true);
  const [activePanel, setActivePanel] = useState<Panel>(null);
  const [showDestroyConfirm, setShowDestroyConfirm] = useState(false);
  const [actionFeedback, setActionFeedback] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [newAgentName, setNewAgentName] = useState("");
  const [newAgentTier, setNewAgentTier] = useState("citizen");
  const [showNewAgent, setShowNewAgent] = useState(false);

  const { toast } = useToast();
  const refetchSidebar = useRefetch();
  const worldName = world ? extractName(world.id) : null;
  usePageTitle(worldName);

  const fetchWorld = useCallback(() => {
    apiGet<World[]>("/api/universes")
      .then((worlds) => {
        const found = worlds.find((w) => w.id === worldId);
        setWorld(found ?? null);
        setLoading(false);
      })
      .catch(() => {
        setWorld(null);
        setLoading(false);
      });
  }, [worldId]);

  useEffect(() => {
    fetchWorld();
    const interval = setInterval(fetchWorld, 5000);
    return () => clearInterval(interval);
  }, [fetchWorld]);

  const callAction = async (goPath: string, body?: unknown) => {
    setActionLoading(goPath);
    try {
      const result = await apiAction(goPath, body);
      if (!result.ok) {
        showFeedback(`Error: ${result.error || "Unknown error"}`);
        return false;
      }
      // Immediately refetch data after successful mutation
      fetchWorld();
      refetchSidebar();
      return true;
    } catch {
      showFeedback("Error: Failed to connect to API");
      return false;
    } finally {
      setActionLoading(null);
    }
  };

  // Snapshots are not yet available from the API
  const snapshots: { id: string; worldId: string; name: string; created_at: string; size: string; agents: number }[] = [];

  // Log streaming state
  const [logs, setLogs] = useState<{ timestamp: string; level: string; source: string; message: string }[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const logsEndRef = useRef<HTMLDivElement>(null);

  // Fetch logs when panel is opened
  useEffect(() => {
    if (activePanel !== "logs") return;
    setLogsLoading(true);
    const controller = new AbortController();

    fetch(goApiUrl(`/api/worlds/${worldId}/logs`), { signal: controller.signal })
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch logs");
        const reader = res.body?.getReader();
        if (!reader) return;
        const decoder = new TextDecoder();
        let buffer = "";

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() ?? "";
          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed || trimmed.startsWith(":")) continue;
            if (trimmed.startsWith("data: ")) {
              try {
                const data = JSON.parse(trimmed.slice(6));
                setLogs((prev) => [...prev.slice(-500), {
                  timestamp: data.timestamp || new Date().toISOString(),
                  level: data.level || "info",
                  source: data.source || "world",
                  message: data.message || data.line || trimmed.slice(6),
                }]);
              } catch {
                setLogs((prev) => [...prev.slice(-500), {
                  timestamp: new Date().toISOString(),
                  level: "info",
                  source: "world",
                  message: trimmed.slice(6),
                }]);
              }
            }
          }
        }
      })
      .catch(() => {
        // Logs endpoint not available — that's OK
      })
      .finally(() => setLogsLoading(false));

    return () => controller.abort();
  }, [activePanel, worldId]);

  const showFeedback = (msg: string) => {
    const isError = msg.toLowerCase().startsWith("error");
    toast(msg, isError ? "error" : "success");
    setActionFeedback(msg);
    setTimeout(() => setActionFeedback(null), 2500);
  };

  if (loading) {
    return (
      <div className="p-8 space-y-8">
        <div className="flex items-center gap-4">
          <Skeleton className="w-2.5 h-2.5 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-7 w-32" />
            <Skeleton className="h-3 w-48" />
          </div>
        </div>
        <div className="flex gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="glass-subtle px-5 py-4 flex-1 min-w-[140px]">
              <Skeleton className="h-3 w-16 mb-2" />
              <Skeleton className="h-7 w-12" />
            </div>
          ))}
        </div>
        <div className="space-y-3">
          <Skeleton className="h-4 w-24" />
          <div className="grid grid-cols-2 gap-3">
            {[1, 2].map((i) => (
              <Skeleton key={i} className="h-16 rounded-xl" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (!world) {
    return (
      <div className="p-8">
        <p className="text-muted-foreground/50">World not found</p>
      </div>
    );
  }

  const name = extractName(world.id);

  return (
    <div className="flex h-[calc(100vh-1px)] overflow-hidden">
      {/* Main content */}
      <div className="flex-1 overflow-y-auto px-6 md:px-8 pt-6 md:pt-8 pb-12 space-y-6 md:space-y-8">
        <PageHeader
          title={name}
          description={`${world.config} · ${timeAgo(world.created_at)}${world.workspace ? " · " + world.workspace : ""}`}
          actions={
            <div className="flex items-center gap-1">
            <button
              onClick={async () => {
                const ok = await callAction(`/api/worlds/${worldId}/snapshot`);
                if (ok) showFeedback("Snapshot saved!");
              }}
              disabled={actionLoading !== null}
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
              title="Snapshot"
            >
              {actionLoading?.includes("snapshot") ? (
                <div className="w-3.5 h-3.5 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
              ) : (
                <IconCamera size={15} />
              )}
              <span className="hidden sm:inline">Snapshot</span>
            </button>
            <button
              onClick={() => setActivePanel(activePanel === "logs" ? null : "logs")}
              className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-[11px] transition-colors ${
                activePanel === "logs"
                  ? "bg-white/[0.08] text-foreground/70"
                  : "text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04]"
              }`}
              title="View Logs"
            >
              <IconFileText size={15} />
              <span className="hidden sm:inline">Logs</span>
            </button>
            <button
              onClick={() => setActivePanel(activePanel === "snapshots" ? null : "snapshots")}
              className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-[11px] transition-colors ${
                activePanel === "snapshots"
                  ? "bg-white/[0.08] text-foreground/70"
                  : "text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04]"
              }`}
              title="Snapshots"
            >
              <IconRestore size={15} />
              <span className="hidden sm:inline">Snapshots</span>
            </button>
            <div className="w-px h-5 bg-border/20 mx-1" />
            <button
              onClick={() => setShowDestroyConfirm(true)}
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-[11px] text-red-400/60 hover:text-red-400 hover:bg-red-500/10 transition-colors"
              title="Destroy World"
            >
              <IconTrash size={15} />
              <span className="hidden sm:inline">Destroy</span>
            </button>
          </div>
          }
        />

        {/* Destroy confirmation */}
        {showDestroyConfirm && (
          <div className="rounded-xl border border-red-500/30 bg-red-500/5 p-5">
            <div className="flex items-start gap-3">
              <IconAlertTriangle size={20} className="text-red-400 shrink-0 mt-0.5" />
              <div className="flex-1">
                <h3 className="text-sm font-heading text-red-300">Destroy World?</h3>
                <p className="text-xs text-red-300/60 mt-1">
                  This will permanently destroy <span className="font-mono">{world.id}</span> and all its agents.
                  This action cannot be undone.
                </p>
                <div className="flex gap-2 mt-4">
                  <button
                    onClick={async () => {
                      setActionLoading("destroy");
                      try {
                        await apiDelete(`/api/worlds/${worldId}`);
                        showFeedback("World destroyed");
                        setShowDestroyConfirm(false);
                        refetchSidebar();
                        router.push("/");
                      } catch {
                        showFeedback("Error: Failed to destroy world");
                        setShowDestroyConfirm(false);
                      } finally {
                        setActionLoading(null);
                      }
                    }}
                    disabled={actionLoading !== null}
                    className="px-4 py-2 rounded-lg text-xs bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/30 transition-colors disabled:opacity-30"
                  >
                    {actionLoading?.includes("destroy") ? "Destroying..." : "Yes, destroy it"}
                  </button>
                  <button
                    onClick={() => setShowDestroyConfirm(false)}
                    className="px-4 py-2 rounded-lg text-xs text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Action feedback toast */}
        {actionFeedback && (
          <div className="px-4 py-2 rounded-lg bg-green-500/10 border border-green-500/20 text-green-400 text-xs font-mono animate-in fade-in slide-in-from-top-2 duration-200">
            {actionFeedback}
          </div>
        )}

        {/* Stats */}
        <div className="grid grid-cols-2 md:flex md:flex-row gap-3 md:gap-4">
          <StatCard label="Status" value={world.status} sub={world.id.substring(0, 12)} />
          <StatCard label="Agents" value={String(world.agents.length)} />
          <StatCard label="Config" value={world.config} />
          <StatCard label="Uptime" value={timeAgo(world.created_at)} />
        </div>

        {/* Physics / Manifest Info */}
        {world.manifest && (
          <div>
            <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">Physics</h2>
            <div className="flex gap-3 flex-wrap">
              {world.manifest.cpu && (
                <div className="glass-subtle px-4 py-3">
                  <p className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-0.5">CPU</p>
                  <p className="text-sm font-mono text-foreground/70">{world.manifest.cpu}</p>
                </div>
              )}
              {world.manifest.memory && (
                <div className="glass-subtle px-4 py-3">
                  <p className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-0.5">Memory</p>
                  <p className="text-sm font-mono text-foreground/70">{world.manifest.memory}</p>
                </div>
              )}
              {world.manifest.timeout && (
                <div className="glass-subtle px-4 py-3">
                  <p className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-0.5">Timeout</p>
                  <p className="text-sm font-mono text-foreground/70">{world.manifest.timeout}</p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Elements */}
        {world.manifest?.elements && world.manifest.elements.length > 0 && (
          <div>
            <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">Elements</h2>
            <div className="flex flex-wrap gap-2">
              {world.manifest.elements.map((el) => (
                <span
                  key={el}
                  className="px-3 py-1.5 rounded-lg text-[11px] font-mono bg-blue-500/10 text-blue-300/80 border border-blue-500/20"
                >
                  {el}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Agents */}
        <div>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40">Agents</h2>
            <button
              onClick={() => setShowNewAgent(!showNewAgent)}
              className="flex items-center gap-1 text-[11px] text-muted-foreground/40 hover:text-foreground/60 hover:bg-white/[0.04] px-2.5 py-1.5 rounded-lg transition-colors"
            >
              <IconPlus size={13} />
              New Agent
            </button>
          </div>

          {/* New Agent form */}
          {showNewAgent && (
            <div className="glass-subtle p-4 mb-3 space-y-3">
              <div className="flex gap-3">
                <input
                  value={newAgentName}
                  onChange={(e) => setNewAgentName(e.target.value)}
                  placeholder="Agent name..."
                  className="flex-1 bg-transparent text-sm text-foreground/70 placeholder:text-muted-foreground/30 border-b border-border/20 pb-2 focus:outline-none"
                />
                <select
                  value={newAgentTier}
                  onChange={(e) => setNewAgentTier(e.target.value)}
                  className="bg-transparent text-sm text-foreground/70 border-b border-border/20 pb-2 focus:outline-none"
                >
                  <option value="governor">Governor</option>
                  <option value="citizen">Citizen</option>
                  <option value="npc">NPC</option>
                </select>
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setShowNewAgent(false)}
                  className="px-3 py-1.5 rounded-lg text-[11px] text-muted-foreground/40 hover:text-foreground/60 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={async () => {
                    if (!newAgentName.trim()) return;
                    const ok = await callAction("/api/agents", { name: newAgentName.trim() });
                    if (ok) {
                      showFeedback(`Agent "${newAgentName}" created`);
                      setShowNewAgent(false);
                      setNewAgentName("");
                      fetchWorld();
                    }
                  }}
                  disabled={!newAgentName.trim() || actionLoading !== null}
                  className="px-3 py-1.5 rounded-lg text-[11px] bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] transition-colors disabled:opacity-30"
                >
                  {actionLoading ? "Creating..." : "Create Agent"}
                </button>
              </div>
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {world.agents.map((agent) => {
              const tierStyle = TIER_BADGE[agent.tier] ?? TIER_BADGE.citizen;
              const statusStyle = STATUS_BADGE[agent.status] ?? STATUS_BADGE.stopped;
              return (
                <a
                  key={agent.name}
                  href={`/world/${worldId}/${agent.name}`}
                  className="glass-subtle p-4 flex items-center justify-between hover:bg-white/[0.04] transition-colors group"
                >
                  <div className="flex items-center gap-3">
                    <div className={`w-2 h-2 rounded-full ${STATUS_DOT[agent.status] ?? "bg-white/20"}`} />
                    <div>
                      <div className="flex items-center gap-2">
                        <p className="text-sm text-foreground/80 group-hover:text-foreground/90">{agent.name}</p>
                        <span className={`px-1.5 py-0.5 rounded-full text-[9px] font-mono border ${tierStyle}`}>
                          {agent.tier}
                        </span>
                      </div>
                    </div>
                  </div>
                  <span className={`px-2 py-0.5 rounded-full text-[10px] font-mono border ${statusStyle}`}>
                    {agent.status}
                  </span>
                </a>
              );
            })}
          </div>
        </div>

        {/* Activity */}
        <div>
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">Recent Activity</h2>
          <div className="glass-subtle px-5 py-8 text-center">
            <p className="text-sm text-muted-foreground/30">No activity recorded yet</p>
          </div>
        </div>

        {/* Quick commands */}
        <div>
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">Commands</h2>
          <div className="glass-subtle p-4 font-mono text-xs text-muted-foreground/40 space-y-1.5">
            {world.agents.length > 0 && (
              <p>spwn agent talk {world.agents[0].name} &quot;message&quot;</p>
            )}
            <p>spwn agent dream {world.agents[0]?.name ?? "‹agent›"}</p>
            <p>spwn agent sleep {world.agents[0]?.name ?? "‹agent›"}</p>
            <p>spwn profile {world.agents[0]?.name ?? "‹agent›"}</p>
            <p>spwn agent fork {world.agents[0]?.name ?? "‹agent›"} &lt;new&gt;</p>
            <p>spwn agent export {world.agents[0]?.name ?? "‹agent›"}</p>
          </div>
        </div>
      </div>

      {/* ── Side panel for Logs/Snapshots ── */}
      {activePanel && (
        <div className="hidden md:flex w-96 border-l border-border/30 flex-col shrink-0 overflow-hidden">
          <div className="px-5 py-4 border-b border-border/30 flex items-center justify-between shrink-0">
            <h2 className="text-sm font-heading text-foreground/80 capitalize">{activePanel}</h2>
            <button
              onClick={() => setActivePanel(null)}
              className="text-muted-foreground/40 hover:text-foreground/70 transition-colors"
            >
              <IconX size={16} />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto">
            {activePanel === "logs" && (
              <div className="p-4 space-y-0.5 font-mono text-[11px]">
                {logsLoading && logs.length === 0 && (
                  <div className="flex items-center gap-2 text-muted-foreground/30 py-8 justify-center">
                    <div className="w-3 h-3 border-2 border-foreground/20 border-t-foreground/50 rounded-full animate-spin" />
                    <span className="text-sm">Connecting to log stream...</span>
                  </div>
                )}
                {!logsLoading && logs.length === 0 && (
                  <div className="text-center py-8">
                    <p className="text-sm text-muted-foreground/30">No logs available</p>
                    <p className="text-[10px] text-muted-foreground/20 font-mono mt-1">Use the CLI: spwn logs {worldId}</p>
                  </div>
                )}
                {logs.map((log, i) => (
                  <div key={i} className="flex gap-2 py-1.5 border-b border-border/10 last:border-0">
                    <span className="text-muted-foreground/25 shrink-0 w-14">
                      {new Date(log.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                    </span>
                    <span className={`shrink-0 w-10 uppercase ${LOG_LEVEL_COLORS[log.level]}`}>
                      {log.level}
                    </span>
                    <span className="text-muted-foreground/40 shrink-0 w-16">{log.source}</span>
                    <span className="text-foreground/60 break-all">{log.message}</span>
                  </div>
                ))}
                <div ref={logsEndRef} />
              </div>
            )}

            {activePanel === "snapshots" && (
              <div className="p-4 space-y-3">
                {snapshots.length === 0 ? (
                  <p className="text-sm text-muted-foreground/30 text-center py-8">No snapshots</p>
                ) : (
                  snapshots.map((snap) => (
                    <div key={snap.id} className="glass-subtle p-4">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs font-mono text-foreground/70">{snap.name}</span>
                        <span className="text-[10px] font-mono text-muted-foreground/30">{snap.size}</span>
                      </div>
                      <p className="text-[10px] font-mono text-muted-foreground/40 mb-3">
                        {timeAgo(snap.created_at)} · {snap.agents} agent{snap.agents !== 1 ? "s" : ""}
                      </p>
                      <div className="flex gap-2">
                        <button
                          onClick={() => showFeedback(`Restoring "${snap.name}"...`)}
                          className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-[10px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors"
                        >
                          <IconPlayerPlay size={12} />
                          Restore
                        </button>
                        <button
                          onClick={() => showFeedback("Downloading...")}
                          className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-[10px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors"
                        >
                          <IconDownload size={12} />
                          Export
                        </button>
                        <button
                          onClick={() => showFeedback("Snapshot deleted")}
                          className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-[10px] text-red-400/50 hover:text-red-400 hover:bg-red-500/10 transition-colors ml-auto"
                        >
                          <IconTrash size={12} />
                          Delete
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}


