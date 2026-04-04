"use client";

import { useState, useEffect } from "react";
import { IconActivity } from "@tabler/icons-react";
import type { World, AgentProfile } from "@/lib/types";
import { apiGet } from "@/lib/api-client";

interface ActivityEvent {
  id: string;
  text: string;
  time: string;
  type: "spawn" | "dream" | "create" | "info";
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

function extractName(id: string): string {
  const parts = id.split("-");
  return parts.length >= 2
    ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1)
    : id;
}

export function RecentActivity({ worlds }: { worlds: World[] }) {
  const [events, setEvents] = useState<ActivityEvent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const allEvents: ActivityEvent[] = [];

    // Add world creation events
    for (const world of worlds) {
      allEvents.push({
        id: `world-${world.id}`,
        text: `${extractName(world.id)} was spawned`,
        time: world.created_at,
        type: "spawn",
      });

      // Add agent events from worlds
      for (const agent of world.agents) {
        allEvents.push({
          id: `agent-${world.id}-${agent.name}`,
          text: `${agent.name} joined ${extractName(world.id)}`,
          time: world.created_at,
          type: "create",
        });
      }
    }

    // Fetch journal entries from all agents in worlds
    const agentNames = new Set<string>();
    for (const world of worlds) {
      for (const agent of world.agents) {
        agentNames.add(agent.name);
      }
    }

    const fetchPromises = Array.from(agentNames).map((name) =>
      apiGet<AgentProfile>(`/api/agents/${name}`)
        .then((profile) => {
          if (profile?.journal) {
            for (const entry of profile.journal) {
              allEvents.push({
                id: `journal-${name}-${entry.date}`,
                text: `${name}: ${entry.summary}`,
                time: entry.date,
                type: "dream",
              });
            }
          }
        })
        .catch(() => {})
    );

    Promise.all(fetchPromises).then(() => {
      // Sort by most recent first
      allEvents.sort(
        (a, b) => new Date(b.time).getTime() - new Date(a.time).getTime()
      );
      setEvents(allEvents.slice(0, 10));
      setLoading(false);
    });
  }, [worlds]);

  if (loading && worlds.length > 0) {
    return (
      <div className="mt-8 px-6 pb-8">
        <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4 flex items-center gap-2">
          <IconActivity size={14} className="opacity-40" />
          Recent Activity
        </h2>
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-8 rounded-lg bg-white/[0.02] animate-pulse"
            />
          ))}
        </div>
      </div>
    );
  }

  if (events.length === 0) return null;

  const typeColors: Record<string, string> = {
    spawn: "text-green-400/60",
    dream: "text-purple-400/60",
    create: "text-blue-400/60",
    info: "text-muted-foreground/40",
  };

  const typeDots: Record<string, string> = {
    spawn: "bg-green-500",
    dream: "bg-purple-500",
    create: "bg-blue-500",
    info: "bg-white/20",
  };

  return (
    <div className="mt-8 px-6 pb-8">
      <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4 flex items-center gap-2">
        <IconActivity size={14} className="opacity-40" />
        Recent Activity
      </h2>
      <div className="space-y-1">
        {events.map((event) => (
          <div
            key={event.id}
            className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-white/[0.02] transition-colors"
          >
            <div
              className={`w-1.5 h-1.5 rounded-full shrink-0 ${typeDots[event.type]}`}
            />
            <span className="text-xs text-foreground/60 truncate flex-1">
              {event.text}
            </span>
            <span className="text-[10px] font-mono text-muted-foreground/25 shrink-0">
              {timeAgo(event.time)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
