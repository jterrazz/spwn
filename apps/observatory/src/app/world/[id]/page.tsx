"use client";

import { useParams } from "next/navigation";
import { MOCK_WORLDS, MOCK_ACTIVITY } from "@/lib/mock-data";

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

function StatCard({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="glass-subtle px-5 py-4 flex-1 min-w-[140px]">
      <p className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-1">{label}</p>
      <p className="text-2xl font-heading text-foreground/90">{value}</p>
      {sub && <p className="text-[11px] font-mono text-muted-foreground/40 mt-0.5">{sub}</p>}
    </div>
  );
}

export default function WorldDashboard() {
  const params = useParams();
  const worldId = params.id as string;
  const world = MOCK_WORLDS.find((w) => w.id === worldId);

  if (!world) {
    return (
      <div className="p-8">
        <p className="text-muted-foreground/50">World not found</p>
      </div>
    );
  }

  const name = extractName(world.id);

  return (
    <div className="p-8 space-y-8">
      {/* Header */}
      <div className="flex items-center gap-4">
        <div className={`w-2.5 h-2.5 rounded-full ${STATUS_DOT[world.status]}`} />
        <div>
          <h1 className="text-2xl font-heading tracking-wide text-foreground/90">{name}</h1>
          <p className="text-xs font-mono text-muted-foreground/40 mt-0.5">
            {world.id} · {world.config} · {timeAgo(world.created_at)}
          </p>
        </div>
      </div>

      {/* Stats */}
      <div className="flex gap-4 flex-wrap">
        <StatCard label="Status" value={world.status} />
        <StatCard label="Agents" value={String(world.agents.length)} />
        <StatCard label="Config" value={world.config} />
        <StatCard label="Uptime" value={timeAgo(world.created_at)} />
      </div>

      {/* Agents */}
      <div>
        <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">Agents</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {world.agents.map((agent) => (
            <div key={agent.name} className="glass-subtle p-4 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`w-2 h-2 rounded-full ${STATUS_DOT[agent.status] ?? "bg-white/20"}`} />
                <div>
                  <p className="text-sm text-foreground/80">{agent.name}</p>
                  <p className="text-[10px] font-mono text-muted-foreground/40 capitalize">{agent.tier}</p>
                </div>
              </div>
              <span className="text-[10px] font-mono text-muted-foreground/30 uppercase">{agent.status}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Activity */}
      <div>
        <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">Recent Activity</h2>
        <div className="glass-subtle divide-y divide-border/30">
          {MOCK_ACTIVITY.filter((a) => a.world === name).map((item, i) => (
            <div key={i} className="px-5 py-3 flex items-center gap-4">
              <span className="text-[10px] font-mono text-muted-foreground/30 w-14 shrink-0">{item.time}</span>
              <div
                className="w-1 h-1 rounded-full shrink-0"
                style={{
                  backgroundColor: item.type === "success" ? "#22c55e" : "rgba(255,255,255,0.3)",
                  boxShadow: item.type === "success" ? "0 0 4px rgba(34,197,94,0.5)" : "none",
                }}
              />
              <span className="text-xs text-foreground/70">{item.event}</span>
            </div>
          ))}
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
  );
}
