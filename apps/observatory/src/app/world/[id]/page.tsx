"use client";

import { useParams, useRouter } from "next/navigation";
import { useState, useEffect, useCallback, useRef } from "react";
import { getWorkspaceSummary, getWorldName, type World, type Agent } from "@/lib/types";
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
  IconSend,
  IconPencil,
} from "@tabler/icons-react";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/page-header";
import { ActionButton } from "@/components/action-button";
import { WorldPlanet } from "@/components/world-planet";
import { usePageTitle } from "@/hooks/use-page-title";
import { useToast } from "@/components/toast-provider";
import { MetricGrid, SectionHeader, SectionLabel, SubLabel, Separator, StatusDot as DSStatusDot, KeyValue, DataTable } from "@/components/ds";
import { ROLE_BADGE } from "@/lib/status";
import { useRefetch } from "@/components/app-shell";

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
  const [showRenameDialog, setShowRenameDialog] = useState(false);
  const [renameInput, setRenameInput] = useState("");
  const [renaming, setRenaming] = useState(false);

  const { toast } = useToast();
  const refetchSidebar = useRefetch();
  const worldName = world ? getWorldName(world) : null;
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

  const handleRename = async () => {
    setRenaming(true);
    try {
      const res = await fetch(goApiUrl(`/api/worlds/${worldId}`), {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: renameInput.trim() }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        showFeedback(`Error: ${data.error || "Failed to rename"}`);
        setRenaming(false);
        return;
      }
      fetchWorld();
      refetchSidebar();
      setShowRenameDialog(false);
    } catch {
      showFeedback("Error: Failed to connect to API");
    } finally {
      setRenaming(false);
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
        <div className="flex gap-10">
          {[1, 2, 3, 4].map((i) => (
            <div key={i}>
              <Skeleton className="h-3 w-14 mb-2" />
              <Skeleton className="h-7 w-10" />
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

  const name = getWorldName(world);

  return (
    <div className="flex h-[calc(100vh-1px)] overflow-hidden">
      {/* Main content */}
      <div className="flex-1 overflow-y-auto px-6 md:px-8 pt-6 md:pt-8 pb-12 space-y-6 md:space-y-8">
        <PageHeader
          leading={<WorldPlanet world={world} size="lg" />}
          title={name}
          description={`${world.config} · ${timeAgo(world.created_at)} · ${getWorkspaceSummary(world)}`}
          actions={
            <>
              <ActionButton
                compact
                onClick={async () => {
                  const ok = await callAction(`/api/worlds/${worldId}/snapshot`);
                  if (ok) showFeedback("Snapshot saved!");
                }}
                disabled={actionLoading !== null}
                label="Snapshot"
                icon={<IconCamera size={16} stroke={2.2} />}
              />
              <ActionButton
                compact
                onClick={() => setActivePanel(activePanel === "logs" ? null : "logs")}
                label="Logs"
                icon={<IconFileText size={16} stroke={2.2} />}
              />
              <ActionButton
                compact
                onClick={() => setActivePanel(activePanel === "snapshots" ? null : "snapshots")}
                label="Snapshots"
                icon={<IconRestore size={16} stroke={2.2} />}
              />
              <ActionButton
                compact
                onClick={() => { setRenameInput(world.name ?? ""); setShowRenameDialog(true); }}
                label="Rename"
                icon={<IconPencil size={16} stroke={2.2} />}
              />
              <ActionButton
                compact
                danger
                onClick={() => setShowDestroyConfirm(true)}
                label="Destroy"
                icon={<IconTrash size={16} stroke={2.2} />}
              />
            </>
          }
        />

        {/* Rename dialog */}
        {showRenameDialog && (
          <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => !renaming && setShowRenameDialog(false)} />
            <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
              <h3 className="text-lg font-heading text-foreground/90 mb-1">Rename World</h3>
              <p className="text-sm text-muted-foreground/50 mb-5">
                Leave empty to fall back to the auto-generated name (<span className="font-mono">{world.id.split("-")[1] ?? world.id}</span>).
              </p>
              <input
                type="text"
                value={renameInput}
                onChange={(e) => setRenameInput(e.target.value)}
                placeholder="My Project"
                disabled={renaming}
                className="w-full px-3 py-2.5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-foreground/80 placeholder:text-muted-foreground/30 focus:outline-none focus:border-white/[0.16] transition-colors disabled:opacity-50"
                onKeyDown={(e) => { if (e.key === "Enter") handleRename(); }}
                autoFocus
              />
              <div className="flex gap-3 justify-end mt-6">
                <button
                  onClick={() => setShowRenameDialog(false)}
                  disabled={renaming}
                  className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  onClick={handleRename}
                  disabled={renaming}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm bg-white/[0.1] text-foreground/90 hover:bg-white/[0.16] border border-white/[0.08] transition-colors disabled:opacity-50"
                >
                  {renaming ? (
                    <>
                      <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/80 rounded-full animate-spin" />
                      Saving…
                    </>
                  ) : (
                    "Save"
                  )}
                </button>
              </div>
            </div>
          </div>
        )}

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
        <MetricGrid columns={4} className="w-fit gap-x-10" items={[
          { label: "Status", value: world.status },
          { label: "Agents", value: world.agents.length },
          { label: "Config", value: world.config },
          { label: "Uptime", value: timeAgo(world.created_at) },
        ]} />

        <Separator />

        {/* Elements */}
        {world.manifest?.elements && world.manifest.elements.length > 0 && (
          <>
            <div>
              <SectionHeader>Elements</SectionHeader>
              <div className="flex flex-wrap gap-1.5">
                {world.manifest.elements.map((el) => (
                  <span
                    key={el}
                    className="px-2.5 py-1 text-[11px] font-mono text-foreground/60 bg-white/[0.04] border border-white/[0.06]"
                  >
                    {el}
                  </span>
                ))}
              </div>
            </div>
            <Separator />
          </>
        )}

        {/* Agents */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <SectionHeader className="mb-0">Agents</SectionHeader>
            <button
              onClick={() => router.push("/agents")}
              className="text-[10px] font-mono text-muted-foreground/35 hover:text-foreground/70 transition-colors"
            >
              + Deploy
            </button>
          </div>
          <DataTable<Agent>
            rows={world.agents}
            rowKey={(a) => a.name}
            rowHref={(a) => `/agents/${encodeURIComponent(a.name)}?world=${worldId}`}
            emptyText="No agents deployed — deploy from the Agents page."
            columns={[
              {
                key: "name",
                label: "Name",
                width: "1fr",
                render: (a) => <span className="text-[13px] font-mono text-foreground/85 truncate">{a.name}</span>,
              },
              {
                key: "role",
                label: "Role",
                width: "80px",
                render: (a) => {
                  const badge = ROLE_BADGE[a.role] ?? ROLE_BADGE.default;
                  return (
                    <span className={`px-1.5 py-0.5 rounded text-[9px] font-mono uppercase tracking-wider border ${badge}`}>
                      {a.role}
                    </span>
                  );
                },
              },
              {
                key: "status",
                label: "Status",
                width: "100px",
                render: (a) => (
                  <span className="flex items-center gap-1.5">
                    <DSStatusDot status={a.status} />
                    <span className="text-[11px] font-mono text-muted-foreground/50 capitalize">{a.status}</span>
                  </span>
                ),
              },
            ]}
          />
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


