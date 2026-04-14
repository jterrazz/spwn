"use client";

import { useState, useEffect } from "react";
import {
  IconWorldFilled,
  IconTrashFilled,
  IconUserFilled,
  IconUserOff,
  IconBrain,
  IconMoonFilled,
  IconGitFork,
  IconMessageFilled,
  IconHexagonFilled,
  IconHexagonOff,
  IconCamera,
  IconSparkles,
} from "@tabler/icons-react";
import { apiGet } from "@/lib/api-client";

type ActivityType =
  | "world.spawned" | "world.destroyed" | "world.snapshot" | "world.state_changed"
  | "agent.created" | "agent.deleted" | "agent.joined" | "agent.left"
  | "agent.dreamed" | "agent.slept" | "agent.forked" | "agent.talked"
  | "architect.started" | "architect.stopped" | "architect.talked"
  | "world.session_ended";

interface ActivityEvent {
  id: string;
  timestamp: string;
  type: ActivityType;
  actor: string;
  verb: string;
  target?: string;
  phrase: string;
  world_id?: string;
  agent_id?: string;
  duration_ms?: number;
  cost_usd?: number;
}

function timeAgo(iso: string): string {
  const d = Date.now() - new Date(iso).getTime();
  if (d < 0) return "just now";
  const s = Math.floor(d / 1000);
  if (s < 60) return "just now";
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

const TYPE_CONFIG: Record<ActivityType, { icon: typeof IconWorldFilled; color: string; bg: string }> = {
  "world.spawned":        { icon: IconWorldFilled,      color: "text-green-400/70",  bg: "bg-green-500/[0.08]" },
  "world.destroyed":      { icon: IconTrashFilled,      color: "text-red-400/70",    bg: "bg-red-500/[0.08]" },
  "world.snapshot":       { icon: IconCamera,           color: "text-blue-400/70",   bg: "bg-blue-500/[0.08]" },
  "world.state_changed":  { icon: IconSparkles,         color: "text-amber-400/70",  bg: "bg-amber-500/[0.08]" },
  "world.session_ended":  { icon: IconSparkles,         color: "text-foreground/50", bg: "bg-white/[0.04]" },
  "agent.created":        { icon: IconUserFilled,       color: "text-blue-400/70",   bg: "bg-blue-500/[0.08]" },
  "agent.deleted":        { icon: IconUserOff,          color: "text-red-400/70",    bg: "bg-red-500/[0.08]" },
  "agent.joined":         { icon: IconUserFilled,       color: "text-blue-400/70",   bg: "bg-blue-500/[0.08]" },
  "agent.left":           { icon: IconUserOff,          color: "text-zinc-400/60",   bg: "bg-zinc-500/[0.08]" },
  "agent.dreamed":        { icon: IconBrain,            color: "text-purple-400/70", bg: "bg-purple-500/[0.08]" },
  "agent.slept":          { icon: IconMoonFilled,       color: "text-purple-400/70", bg: "bg-purple-500/[0.08]" },
  "agent.forked":         { icon: IconGitFork,          color: "text-cyan-400/70",   bg: "bg-cyan-500/[0.08]" },
  "agent.talked":         { icon: IconMessageFilled,    color: "text-foreground/50", bg: "bg-white/[0.04]" },
  "architect.started":    { icon: IconHexagonFilled,    color: "text-green-400/70",  bg: "bg-green-500/[0.08]" },
  "architect.stopped":    { icon: IconHexagonOff,       color: "text-zinc-400/60",   bg: "bg-zinc-500/[0.08]" },
  "architect.talked":     { icon: IconHexagonFilled,    color: "text-foreground/50", bg: "bg-white/[0.04]" },
};

export function RecentActivity() {
  const [events, setEvents] = useState<ActivityEvent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchEvents = async () => {
      try {
        const data = await apiGet<{ events: ActivityEvent[] }>("/api/activity?limit=12");
        setEvents(data.events ?? []);
      } catch {
        // keep old events
      } finally {
        setLoading(false);
      }
    };
    fetchEvents();
    const interval = setInterval(fetchEvents, 10000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="space-y-2">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-14 rounded-xl bg-white/[0.02] animate-pulse" />
        ))}
      </div>
    );
  }

  if (events.length === 0) {
    return (
      <p className="text-xs text-muted-foreground/30 px-4 py-6 text-center">
        No activity yet - spawn a world to get started
      </p>
    );
  }

  return (
    <div className="space-y-2">
      {events.map((event) => {
        const cfg = TYPE_CONFIG[event.type] ?? TYPE_CONFIG["world.session_ended"];
        const Icon = cfg.icon;
        const meta = [];
        if (event.cost_usd) meta.push(`$${event.cost_usd.toFixed(3)}`);
        if (event.duration_ms) {
          const s = Math.floor(event.duration_ms / 1000);
          meta.push(s < 60 ? `${s}s` : `${Math.floor(s / 60)}m`);
        }

        return (
          <div
            key={event.id}
            className="group flex items-center gap-4 px-4 py-3 rounded-xl hover:bg-white/[0.03] transition-all"
          >
            <div className={`w-8 h-8 rounded-lg ${cfg.bg} flex items-center justify-center shrink-0`}>
              <Icon size={14} className={cfg.color} />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-xs text-foreground/70 truncate">{event.phrase}</p>
              {meta.length > 0 && (
                <p className="text-[10px] font-mono text-muted-foreground/25 mt-0.5">{meta.join(" · ")}</p>
              )}
            </div>
            <span className="text-[10px] font-mono text-muted-foreground/20 shrink-0">
              {timeAgo(event.timestamp)}
            </span>
          </div>
        );
      })}
    </div>
  );
}
