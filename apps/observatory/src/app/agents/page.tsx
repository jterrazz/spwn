"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  IconPlus,
  IconUser,
  IconArrowRight,
  IconGhostFilled,
  IconBoltFilled,
  IconCircleFilled,
  IconMessageFilled,
  IconMoonFilled,
  IconX,
  IconCheck,
} from "@tabler/icons-react";
import { Page } from "@/components/page";
import { PageHeader } from "@/components/page-header";
import { ActionButton } from "@/components/action-button";
import { ExpandingSearch } from "@/components/expanding-search";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiAction } from "@/lib/api-client";
import { useRefetch } from "@/components/app-shell";
import { usePageTitle } from "@/hooks/use-page-title";
import { getWorldName, type World } from "@/lib/types";
import { TIER_BADGE } from "@/lib/status";

interface AgentListItem {
  name: string;
  path: string;
  layers: Record<string, string[] | null>;
}

// An agent enriched with its current deployment (if any).
interface EnrichedAgent {
  name: string;
  tier: string;                       // from world membership, else "citizen"
  status: string;                     // running/waiting/idle/sleeping/stopped/limbo
  worldID?: string;                   // when deployed
  worldName?: string;
  layersCount: number;                // non-empty mind layers
  journalEntries: number;
  sessionsCount: number;
}

type StatusFilter = "all" | "deployed" | "limbo";

const STATUS_ICON: Record<string, { icon: typeof IconBoltFilled; color: string }> = {
  running:  { icon: IconBoltFilled,    color: "text-green-400" },
  waiting:  { icon: IconMessageFilled, color: "text-amber-400" },
  idle:     { icon: IconCircleFilled,  color: "text-amber-400/50" },
  sleeping: { icon: IconMoonFilled,    color: "text-purple-400" },
  stopped:  { icon: IconCircleFilled,  color: "text-zinc-500/40" },
  limbo:    { icon: IconGhostFilled,   color: "text-muted-foreground/40" },
};

function countLayerFiles(layers: Record<string, string[] | null>): { nonEmpty: number; journal: number; sessions: number } {
  let nonEmpty = 0;
  let journal = 0;
  let sessions = 0;
  for (const [key, val] of Object.entries(layers ?? {})) {
    if (!Array.isArray(val) || val.length === 0) continue;
    nonEmpty++;
    if (key === "memory/journal") journal = val.length;
    if (key === "sessions") sessions = val.length;
  }
  return { nonEmpty, journal, sessions };
}

export default function AgentsPage() {
  const router = useRouter();
  const refetchSidebar = useRefetch();
  usePageTitle("Agents");

  const [agents, setAgents] = useState<EnrichedAgent[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<StatusFilter>("all");
  const [query, setQuery] = useState("");
  const [showNew, setShowNew] = useState(false);
  const [newName, setNewName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");

  const fetchAll = useCallback(async () => {
    try {
      const [worlds, rawAgents] = await Promise.all([
        apiGet<World[]>("/api/universes").catch(() => [] as World[]),
        apiGet<AgentListItem[]>("/api/agents").catch(() => [] as AgentListItem[]),
      ]);

      // Build name → { worldID, worldName, tier, status } map from world records.
      const placement = new Map<string, { worldID: string; worldName: string; tier: string; status: string }>();
      for (const w of worlds) {
        for (const a of w.agents ?? []) {
          placement.set(a.name, {
            worldID: w.id,
            worldName: getWorldName(w),
            tier: a.tier ?? "citizen",
            status: a.status ?? "idle",
          });
        }
      }

      const enriched: EnrichedAgent[] = rawAgents.map((a) => {
        const counts = countLayerFiles(a.layers ?? {});
        const p = placement.get(a.name);
        return {
          name: a.name,
          tier: p?.tier ?? "citizen",
          status: p?.status ?? "limbo",
          worldID: p?.worldID,
          worldName: p?.worldName,
          layersCount: counts.nonEmpty,
          journalEntries: counts.journal,
          sessionsCount: counts.sessions,
        };
      });

      enriched.sort((a, b) => a.name.localeCompare(b.name));
      setAgents(enriched);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll();
    const id = setInterval(fetchAll, 5000);
    return () => clearInterval(id);
  }, [fetchAll]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return agents.filter((a) => {
      if (filter === "deployed" && !a.worldID) return false;
      if (filter === "limbo" && a.worldID) return false;
      if (q && !a.name.toLowerCase().includes(q)) return false;
      return true;
    });
  }, [agents, filter, query]);

  const counts = useMemo(() => ({
    all: agents.length,
    deployed: agents.filter((a) => a.worldID).length,
    limbo: agents.filter((a) => !a.worldID).length,
  }), [agents]);

  const handleCreate = async () => {
    const name = newName.trim();
    if (!name) return;
    setCreating(true);
    setCreateError("");
    try {
      const result = await apiAction("/api/agents", { name });
      if (!result.ok) {
        setCreateError(result.error || "Failed to create agent");
        setCreating(false);
        return;
      }
      setNewName("");
      setShowNew(false);
      refetchSidebar();
      await fetchAll();
      router.push(`/agents/${name}`);
    } catch {
      setCreateError("Failed to connect to API");
    } finally {
      setCreating(false);
    }
  };

  return (
    <Page>
      <PageHeader
        title="Agents"
        description="Every agent's identity persists independently of any world — deploy, dream, retire."
        actions={
          <>
            <ExpandingSearch value={query} onChange={setQuery} placeholder="Search agents…" />
            <ActionButton
              compact
              onClick={() => { setCreateError(""); setShowNew(true); }}
              label="New Agent"
              icon={<IconPlus size={18} stroke={2.4} />}
            />
          </>
        }
      />

      {/* Filter */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex items-center gap-1 glass-pill px-1 py-1">
          {(["all", "deployed", "limbo"] as StatusFilter[]).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-1 rounded-full text-xs capitalize transition-colors ${
                filter === f
                  ? "bg-white/[0.1] text-foreground/90"
                  : "text-muted-foreground/50 hover:text-foreground/70"
              }`}
            >
              {f} <span className="text-[10px] font-mono text-muted-foreground/40 ml-1">{counts[f]}</span>
            </button>
          ))}
        </div>
      </div>

      {/* List */}
      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-14 w-full rounded-xl" />)}
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <div className="w-14 h-14 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-4">
            <IconUser size={24} className="text-muted-foreground/30" />
          </div>
          <p className="text-sm text-muted-foreground/50">
            {query ? "No agents match your search" : "No agents yet — create one to get started"}
          </p>
        </div>
      ) : (
        <div className="space-y-1">
          {filtered.map((a) => {
            const s = STATUS_ICON[a.status] ?? STATUS_ICON.limbo;
            const Icon = s.icon;
            const tierStyle = TIER_BADGE[a.tier] ?? TIER_BADGE.citizen;
            return (
              <Link
                key={a.name}
                href={`/agents/${a.name}`}
                className="group flex items-center gap-4 px-4 py-3 rounded-xl border border-transparent hover:border-white/[0.08] hover:bg-white/[0.03] transition-all"
              >
                <Icon size={16} className={`${s.color} shrink-0`} />
                <div className="min-w-0 flex-1 flex items-center gap-3">
                  <span className="font-mono text-sm text-foreground/85 group-hover:text-foreground transition-colors truncate">
                    {a.name}
                  </span>
                  <span className={`shrink-0 px-1.5 py-0.5 rounded text-[9px] font-mono uppercase tracking-wider border ${tierStyle}`}>
                    {a.tier}
                  </span>
                </div>

                <div className="hidden sm:flex items-center gap-4 text-[11px] font-mono text-muted-foreground/40 shrink-0">
                  <span title="Mind layers">{a.layersCount} layers</span>
                  <span title="Sessions">{a.sessionsCount} sessions</span>
                </div>

                <div className="shrink-0 min-w-[110px] text-right text-xs text-muted-foreground/50">
                  {a.worldID ? (
                    <>
                      <span className="text-muted-foreground/30">in </span>
                      <span className="text-foreground/65">{a.worldName}</span>
                    </>
                  ) : (
                    <span className="text-muted-foreground/30 uppercase tracking-wider text-[10px] font-mono">Limbo</span>
                  )}
                </div>

                <IconArrowRight size={14} className="text-muted-foreground/20 group-hover:text-muted-foreground/60 transition-colors shrink-0" />
              </Link>
            );
          })}
        </div>
      )}

      {/* New Agent dialog */}
      {showNew && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => !creating && setShowNew(false)} />
          <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
            <h3 className="text-lg font-heading text-foreground/90 mb-1">New Agent</h3>
            <p className="text-sm text-muted-foreground/50 mb-5">
              Creates a new agent identity in limbo. Deploy it to a world when ready.
            </p>
            <input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="e.g. atlas, morpheus, neo…"
              disabled={creating}
              autoFocus
              onKeyDown={(e) => { if (e.key === "Enter") handleCreate(); if (e.key === "Escape") setShowNew(false); }}
              className="w-full px-3 py-2.5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm font-mono text-foreground/80 placeholder:text-muted-foreground/30 focus:outline-none focus:border-white/[0.16] transition-colors disabled:opacity-50"
            />
            {createError && <p className="text-xs text-red-400/80 mt-3">{createError}</p>}
            <div className="flex gap-3 justify-end mt-6">
              <button
                onClick={() => setShowNew(false)}
                disabled={creating}
                className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={creating || !newName.trim()}
                className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm bg-white/[0.1] text-foreground/90 hover:bg-white/[0.16] border border-white/[0.08] transition-colors disabled:opacity-50"
              >
                {creating ? (
                  <>
                    <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/80 rounded-full animate-spin" />
                    Creating…
                  </>
                ) : (
                  <>
                    <IconCheck size={14} />
                    Create
                  </>
                )}
              </button>
            </div>
            <button
              aria-label="Close"
              onClick={() => !creating && setShowNew(false)}
              className="absolute top-4 right-4 text-muted-foreground/30 hover:text-foreground/60 transition-colors disabled:opacity-30"
              disabled={creating}
            >
              <IconX size={16} />
            </button>
          </div>
        </div>
      )}
    </Page>
  );
}
