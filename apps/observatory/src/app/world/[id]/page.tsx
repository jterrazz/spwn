"use client";

import { useParams, useRouter } from "next/navigation";
import { useState, useEffect, useCallback } from "react";
import type { World } from "@/lib/types";
import { apiGet, apiAction } from "@/lib/api-client";
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
} from "@tabler/icons-react";
import { Skeleton } from "@/components/ui/skeleton";

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

const STATUS_DOT: Record<string, string> = {
  running: "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]",
  idle: "bg-yellow-500 shadow-[0_0_6px_rgba(234,179,8,0.5)]",
  stopped: "bg-white/20",
};

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

  const fetchWorld = useCallback(() => {
    apiGet<World[]>("/api/universes", "/api/worlds")
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

  const callAction = async (goPath: string, nextFallback: string, body?: unknown) => {
    setActionLoading(goPath);
    try {
      const result = await apiAction(goPath, body, nextFallback);
      if (!result.ok) {
        showFeedback(`Error: ${result.error || "Unknown error"}`);
        return false;
      }
      return true;
    } catch {
      showFeedback("Error: Failed to connect to API");
      return false;
    } finally {
      setActionLoading(null);
    }
  };

  // Logs and snapshots are not yet available from the API
  const snapshots: { id: string; worldId: string; name: string; created_at: string; size: string; agents: number }[] = [];
  const logs: { timestamp: string; level: string; source: string; message: string }[] = [];

  const showFeedback = (msg: string) => {
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
      <div className="flex-1 overflow-y-auto p-8 space-y-8">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <div className={`w-2.5 h-2.5 rounded-full ${STATUS_DOT[world.status]}`} />
            <div>
              <h1 className="text-2xl font-heading tracking-wide text-foreground/90">{name}</h1>
              <p className="text-xs font-mono text-muted-foreground/40 mt-0.5">
                {world.id} · {world.config} · {timeAgo(world.created_at)}
              </p>
            </div>
          </div>

          {/* World Actions */}
          <div className="flex items-center gap-1">
            <button
              onClick={async () => {
                const ok = await callAction(`/api/worlds/${worldId}/snapshot`, `/api/worlds/${worldId}/snapshot`);
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
        </div>

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
                      const ok = await callAction(`/api/worlds/${worldId}`, `/api/worlds/${worldId}/destroy`);
                      if (ok) {
                        showFeedback("World destroyed");
                        setShowDestroyConfirm(false);
                        router.push("/");
                      } else {
                        setShowDestroyConfirm(false);
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
        <div className="flex gap-4 flex-wrap">
          <StatCard label="Status" value={world.status} />
          <StatCard label="Agents" value={String(world.agents.length)} />
          <StatCard label="Config" value={world.config} />
          <StatCard label="Uptime" value={timeAgo(world.created_at)} />
        </div>

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
                    const ok = await callAction("/api/agents", "/api/agents/create", { name: newAgentName.trim() });
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
            {world.agents.map((agent) => (
              <a
                key={agent.name}
                href={`/world/${worldId}/${agent.name}`}
                className="glass-subtle p-4 flex items-center justify-between hover:bg-white/[0.04] transition-colors group"
              >
                <div className="flex items-center gap-3">
                  <div className={`w-2 h-2 rounded-full ${STATUS_DOT[agent.status] ?? "bg-white/20"}`} />
                  <div>
                    <p className="text-sm text-foreground/80 group-hover:text-foreground/90">{agent.name}</p>
                    <p className="text-[10px] font-mono text-muted-foreground/40 capitalize">{agent.tier}</p>
                  </div>
                </div>
                <span className="text-[10px] font-mono text-muted-foreground/30 uppercase">{agent.status}</span>
              </a>
            ))}
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
            <p>spwn agent talk {world.agent}</p>
            <p>spwn logs {world.id}</p>
            <p>spwn down {world.id}</p>
            <p>spwn snap {world.id}</p>
          </div>
        </div>
      </div>

      {/* ── Side panel for Logs/Snapshots ── */}
      {activePanel && (
        <div className="w-96 border-l border-border/30 flex flex-col shrink-0 overflow-hidden">
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
                {logs.length === 0 ? (
                  <p className="text-sm text-muted-foreground/30 text-center py-8">No logs available. Use the CLI: spwn logs {worldId}</p>
                ) : (
                  logs.map((log, i) => (
                    <div key={i} className="flex gap-2 py-1.5 border-b border-border/10 last:border-0">
                      <span className="text-muted-foreground/25 shrink-0 w-14">
                        {new Date(log.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                      </span>
                      <span className={`shrink-0 w-10 uppercase ${LOG_LEVEL_COLORS[log.level]}`}>
                        {log.level}
                      </span>
                      <span className="text-muted-foreground/40 shrink-0 w-16">{log.source}</span>
                      <span className="text-foreground/60">{log.message}</span>
                    </div>
                  ))
                )}
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
