"use client";

import { useState, useEffect } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiAction } from "@/lib/api-client";

interface ArchitectStatus {
  status: "running" | "stopped";
  containerId: string | null;
  uptime: string | null;
  channels: string[];
  error?: string;
}

interface StatusData {
  worlds: number;
  agents: number;
  running: number;
  limbo: number;
}

const CHANNELS = [
  { type: "cli", status: "connected", label: "CLI", icon: "⬡" },
  { type: "slack", status: "connected", label: "Slack", icon: "◧" },
  { type: "telegram", status: "disconnected", label: "Telegram", icon: "◈" },
  { type: "discord", status: "disconnected", label: "Discord", icon: "◉" },
];

const LOGS = [
  { time: "17:01:02", level: "info", msg: "Architect daemon started", source: "core" },
  { time: "17:01:03", level: "info", msg: "Docker backend connected (v28.5.2)", source: "backend" },
  { time: "17:01:03", level: "info", msg: "Loaded universe manifest — meson", source: "manifest" },
  { time: "17:01:04", level: "info", msg: "Channel connected: cli", source: "channel" },
  { time: "17:01:04", level: "info", msg: "Channel connected: slack → #agents", source: "channel" },
  { time: "17:01:05", level: "info", msg: "Restored 3 worlds from state.json", source: "state" },
  { time: "17:01:12", level: "info", msg: "Spawned agent neo in w-titan-84721", source: "spawn" },
  { time: "17:01:45", level: "info", msg: "Spawned colony (morpheus+trinity) in w-europa-39205", source: "spawn" },
  { time: "17:02:10", level: "warn", msg: "Agent atlas idle for >60m in w-ganymede-51003", source: "health" },
  { time: "17:05:33", level: "info", msg: "neo completed task — exit 0, 2m34s", source: "journal" },
  { time: "17:12:01", level: "info", msg: "morpheus delegated subtask to trinity", source: "msg" },
  { time: "17:14:22", level: "info", msg: "Dream cycle triggered for neo — 2 patterns promoted", source: "evolution" },
];

const LEVEL_COLOR: Record<string, string> = {
  info: "text-muted-foreground/50",
  warn: "text-yellow-500/80",
  error: "text-red-400/80",
};

const LEVEL_DOT: Record<string, string> = {
  info: "bg-muted-foreground/30",
  warn: "bg-yellow-500 shadow-[0_0_4px_rgba(234,179,8,0.5)]",
  error: "bg-red-400 shadow-[0_0_4px_rgba(248,113,113,0.5)]",
};

function StatCard({ label, value, sub, accent, loading: isLoading }: { label: string; value: string; sub?: string; accent?: string; loading?: boolean }) {
  if (isLoading) {
    return (
      <div className="glass-subtle px-5 py-4 flex-1 min-w-[140px]">
        <Skeleton className="h-3 w-16 mb-2" />
        <Skeleton className="h-7 w-12 mb-1" />
        <Skeleton className="h-3 w-20" />
      </div>
    );
  }
  return (
    <div className="glass-subtle px-5 py-4 flex-1 min-w-[140px]">
      <p className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-1">{label}</p>
      <p className={`text-2xl font-heading ${accent ?? "text-foreground/90"}`}>{value}</p>
      {sub && <p className="text-[11px] font-mono text-muted-foreground/40 mt-0.5">{sub}</p>}
    </div>
  );
}

export default function ArchitectPage() {
  const [logFilter, setLogFilter] = useState<string>("all");
  const [architectStatus, setArchitectStatus] = useState<ArchitectStatus | null>(null);
  const [statusData, setStatusData] = useState<StatusData | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      apiGet<ArchitectStatus>("/api/architect/status", "/api/architect/status").catch(() => ({ status: "stopped" as const, containerId: null, uptime: null, channels: [] })),
      apiGet<StatusData>("/api/status", "/api/status").catch(() => null),
    ]).then(([archStatus, sData]) => {
      setArchitectStatus(archStatus);
      setStatusData(sData);
      setLoading(false);
    });
  }, []);

  const showFeedback = (msg: string) => {
    setFeedback(msg);
    setTimeout(() => setFeedback(null), 3000);
  };

  const handleStart = async () => {
    setActionLoading("start");
    try {
      const result = await apiAction("/api/architect/start", undefined, "/api/architect/start");
      if (result.ok) {
        showFeedback("Architect started successfully");
        setArchitectStatus((s) => s ? { ...s, status: "running" } : s);
      } else {
        showFeedback(`Error: ${result.error}`);
      }
    } catch {
      showFeedback("Error: Failed to connect to API");
    } finally {
      setActionLoading(null);
    }
  };

  const handleStop = async () => {
    setActionLoading("stop");
    try {
      const result = await apiAction("/api/architect/stop", undefined, "/api/architect/stop");
      if (result.ok) {
        showFeedback("Architect stopped");
        setArchitectStatus((s) => s ? { ...s, status: "stopped" } : s);
      } else {
        showFeedback(`Error: ${result.error}`);
      }
    } catch {
      showFeedback("Error: Failed to connect to API");
    } finally {
      setActionLoading(null);
    }
  };

  const isRunning = architectStatus?.status === "running";
  const connectedChannels = CHANNELS.filter((c) => c.status === "connected").length;

  const filteredLogs = logFilter === "all"
    ? LOGS
    : LOGS.filter((l) => l.source === logFilter);

  const sources = [...new Set(LOGS.map((l) => l.source))];

  return (
    <div className="p-8 space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          {loading ? (
            <Skeleton className="w-3 h-3 rounded-full" />
          ) : (
            <div className={`w-3 h-3 rounded-full ${isRunning ? "bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)] animate-pulse" : "bg-white/20"}`} />
          )}
          <div>
            <h1 className="text-2xl font-heading tracking-wide text-foreground/90">Architect</h1>
            {loading ? (
              <Skeleton className="h-3 w-48 mt-1" />
            ) : (
              <p className="text-xs font-mono text-muted-foreground/40 mt-0.5">
                Orchestration daemon · {isRunning ? "running" : "stopped"}
                {architectStatus?.containerId && ` · ${architectStatus.containerId.slice(0, 12)}`}
                {architectStatus?.uptime && ` · uptime ${architectStatus.uptime}`}
              </p>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          {isRunning ? (
            <>
              <button
                onClick={handleStart}
                disabled={actionLoading !== null}
                className="glass-subtle px-4 py-2 text-[11px] font-mono uppercase tracking-wider text-muted-foreground/50 hover:text-foreground transition-colors disabled:opacity-30"
              >
                {actionLoading === "start" ? "Restarting..." : "Restart"}
              </button>
              <button
                onClick={handleStop}
                disabled={actionLoading !== null}
                className="glass-subtle px-4 py-2 text-[11px] font-mono uppercase tracking-wider text-red-400/50 hover:text-red-400 transition-colors disabled:opacity-30"
              >
                {actionLoading === "stop" ? "Stopping..." : "Stop"}
              </button>
            </>
          ) : (
            <button
              onClick={handleStart}
              disabled={actionLoading !== null}
              className="glass-subtle px-4 py-2 text-[11px] font-mono uppercase tracking-wider text-green-400/50 hover:text-green-400 transition-colors disabled:opacity-30"
            >
              {actionLoading === "start" ? "Starting..." : "Start"}
            </button>
          )}
        </div>
      </div>

      {/* Feedback toast */}
      {feedback && (
        <div className={`px-4 py-2 rounded-lg text-xs font-mono animate-in fade-in slide-in-from-top-2 duration-200 ${
          feedback.startsWith("Error")
            ? "bg-red-500/10 border border-red-500/20 text-red-400"
            : "bg-green-500/10 border border-green-500/20 text-green-400"
        }`}>
          {feedback}
        </div>
      )}

      {/* Stats */}
      <div className="flex gap-4 flex-wrap">
        <StatCard
          label="Status"
          value={loading ? "" : isRunning ? "Online" : "Offline"}
          accent={isRunning ? "text-green-400" : "text-red-400/60"}
          loading={loading}
        />
        <StatCard
          label="Worlds"
          value={statusData ? String(statusData.worlds) : "—"}
          sub={statusData ? `${statusData.running} running` : undefined}
          loading={loading}
        />
        <StatCard
          label="Agents"
          value={statusData ? String(statusData.agents) : "—"}
          sub="across all worlds"
          loading={loading}
        />
        <StatCard
          label="Channels"
          value={`${connectedChannels}/${CHANNELS.length}`}
          sub="connected"
          loading={loading}
        />
      </div>

      {/* Two-column layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Channels — left column */}
        <div>
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">Channels</h2>
          <div className="space-y-2">
            {CHANNELS.map((ch) => (
              <div key={ch.type} className="glass-subtle px-4 py-3 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="text-sm">{ch.icon}</span>
                  <div>
                    <p className="text-sm text-foreground/80">{ch.label}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <div className={`w-1.5 h-1.5 rounded-full ${ch.status === "connected" ? "bg-green-500 shadow-[0_0_4px_rgba(34,197,94,0.5)]" : "bg-white/15"}`} />
                  <span className="text-[10px] font-mono text-muted-foreground/30 uppercase">{ch.status}</span>
                </div>
              </div>
            ))}
            <button className="w-full glass-subtle px-4 py-2.5 text-[11px] font-mono text-muted-foreground/30 hover:text-muted-foreground/60 transition-colors text-center">
              + Connect channel
            </button>
          </div>

          {/* Config */}
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4 mt-8">Config</h2>
          <div className="glass-subtle p-4 space-y-2">
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Universe</span>
              <span className="text-xs font-mono text-foreground/60">meson</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Runtime</span>
              <span className="text-xs font-mono text-foreground/60">claude-code</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Backend</span>
              <span className="text-xs font-mono text-foreground/60">docker</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Auth</span>
              <span className="text-xs font-mono text-foreground/60">subscription</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Max worlds</span>
              <span className="text-xs font-mono text-foreground/60">10</span>
            </div>
          </div>
        </div>

        {/* Logs — right 2 columns */}
        <div className="lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40">Event Stream</h2>
              <span className="text-[9px] font-mono text-muted-foreground/20 px-2 py-0.5 rounded bg-white/[0.03] border border-white/[0.05]">
                mock data · real log streaming coming soon
              </span>
            </div>
            <div className="flex gap-1">
              <button
                onClick={() => setLogFilter("all")}
                className={`px-2.5 py-1 text-[10px] font-mono uppercase tracking-wider rounded transition-colors ${logFilter === "all" ? "text-foreground/70 glass-subtle" : "text-muted-foreground/30 hover:text-muted-foreground/50"}`}
              >
                All
              </button>
              {sources.map((s) => (
                <button
                  key={s}
                  onClick={() => setLogFilter(s)}
                  className={`px-2.5 py-1 text-[10px] font-mono uppercase tracking-wider rounded transition-colors ${logFilter === s ? "text-foreground/70 glass-subtle" : "text-muted-foreground/30 hover:text-muted-foreground/50"}`}
                >
                  {s}
                </button>
              ))}
            </div>
          </div>

          <div className="glass-subtle rounded-xl overflow-hidden">
            <div className="divide-y divide-border/20 max-h-[500px] overflow-y-auto">
              {filteredLogs.map((log, i) => (
                <div key={i} className="px-4 py-2.5 flex items-start gap-3 hover:bg-white/[0.02] transition-colors">
                  <span className="text-[10px] font-mono text-muted-foreground/25 w-16 shrink-0 pt-0.5">
                    {log.time}
                  </span>
                  <div className={`w-1.5 h-1.5 rounded-full shrink-0 mt-1.5 ${LEVEL_DOT[log.level]}`} />
                  <span className={`text-xs flex-1 ${LEVEL_COLOR[log.level]}`}>
                    {log.msg}
                  </span>
                  <span className="text-[9px] font-mono text-muted-foreground/20 shrink-0">
                    {log.source}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Commands */}
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4 mt-8">Commands</h2>
          <div className="glass-subtle p-4 font-mono text-[10px] text-muted-foreground/35 space-y-1.5">
            <p>spwn architect start</p>
            <p>spwn architect stop</p>
            <p>spwn architect status</p>
            <p>spwn architect connect slack</p>
            <p>spwn architect connect telegram</p>
          </div>
        </div>
      </div>
    </div>
  );
}
