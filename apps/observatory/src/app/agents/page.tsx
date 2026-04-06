"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  IconPlus,
  IconUser,
  IconX,
  IconCheck,
  IconUsers,
} from "@tabler/icons-react";
import { Page } from "@/components/page";
import { PageHeader } from "@/components/page-header";
import { ActionButton } from "@/components/action-button";
import { ExpandingSearch } from "@/components/expanding-search";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiAction, goApiUrl } from "@/lib/api-client";
import { useRefetch } from "@/components/app-shell";
import { usePageTitle } from "@/hooks/use-page-title";
import { getWorldName, type World, type Team } from "@/lib/types";
import { DataTable, StatusDot, SectionLabel } from "@/components/ds";
import { ROLE_BADGE } from "@/lib/status";

interface AgentListItem {
  name: string;
  path: string;
  team?: string;
  layers: Record<string, string[] | null>;
}

// An agent enriched with its current deployment (if any).
interface EnrichedAgent {
  name: string;
  role: string;
  team?: string;                      // team slug
  status: string;                     // running/waiting/idle/sleeping/stopped/limbo
  worldID?: string;
  worldName?: string;
  journalEntries: number;
  sessionsCount: number;
}

type StatusFilter = "all" | "deployed" | "limbo";

function countLayerFiles(layers: Record<string, string[] | null>): { journal: number; sessions: number } {
  let journal = 0;
  let sessions = 0;
  for (const [key, val] of Object.entries(layers ?? {})) {
    if (!Array.isArray(val) || val.length === 0) continue;
    if (key === "memory/journal") journal = val.length;
    if (key === "sessions") sessions = val.length;
  }
  return { journal, sessions };
}

export default function AgentsPage() {
  const router = useRouter();
  const refetchSidebar = useRefetch();
  usePageTitle("Agents");

  const [agents, setAgents] = useState<EnrichedAgent[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<StatusFilter>("all");
  const [query, setQuery] = useState("");
  const [showNew, setShowNew] = useState(false);
  const [newName, setNewName] = useState("");
  const [newTeam, setNewTeam] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");
  const [showTeamDialog, setShowTeamDialog] = useState(false);
  const [editingTeam, setEditingTeam] = useState<Team | null>(null);
  const [teamName, setTeamName] = useState("");
  const [teamIcon, setTeamIcon] = useState("");
  const [teamColor, setTeamColor] = useState("");
  const [teamDesc, setTeamDesc] = useState("");
  const [savingTeam, setSavingTeam] = useState(false);

  const fetchAll = useCallback(async () => {
    try {
      const [worlds, rawAgents, rawTeams] = await Promise.all([
        apiGet<World[]>("/api/universes").catch(() => [] as World[]),
        apiGet<AgentListItem[]>("/api/agents").catch(() => [] as AgentListItem[]),
        apiGet<Team[]>("/api/teams").catch(() => [] as Team[]),
      ]);

      setTeams(rawTeams ?? []);

      // Build name → { worldID, worldName, role, status } map from world records.
      const placement = new Map<string, { worldID: string; worldName: string; role: string; status: string }>();
      for (const w of worlds) {
        for (const a of w.agents ?? []) {
          placement.set(a.name, {
            worldID: w.id,
            worldName: getWorldName(w),
            role: a.role ?? "worker",
            status: a.status ?? "idle",
          });
        }
      }

      const enriched: EnrichedAgent[] = rawAgents.map((a) => {
        const counts = countLayerFiles(a.layers ?? {});
        const p = placement.get(a.name);
        return {
          name: a.name,
          role: p?.role ?? "worker",
          team: a.team,
          status: p?.status ?? "limbo",
          worldID: p?.worldID,
          worldName: p?.worldName,
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

  // Group filtered agents by team for the list view.
  const grouped = useMemo(() => {
    const teamMap = new Map<string, Team>();
    for (const t of teams) teamMap.set(t.slug, t);

    const groups: { team: Team | null; agents: EnrichedAgent[] }[] = [];
    const bySlug = new Map<string, EnrichedAgent[]>();
    const solo: EnrichedAgent[] = [];

    for (const a of filtered) {
      if (a.team) {
        const list = bySlug.get(a.team) ?? [];
        list.push(a);
        bySlug.set(a.team, list);
      } else {
        solo.push(a);
      }
    }

    // Teams first (sorted by name), then solo
    for (const [slug, members] of bySlug) {
      groups.push({ team: teamMap.get(slug) ?? { slug, name: slug }, agents: members });
    }
    groups.sort((a, b) => (a.team?.name ?? "").localeCompare(b.team?.name ?? ""));
    if (solo.length > 0) {
      groups.push({ team: null, agents: solo });
    }
    return groups;
  }, [filtered, teams]);

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
      // Assign team if selected
      if (newTeam) {
        await fetch(goApiUrl(`/api/agents/${encodeURIComponent(name)}/identity`), {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ field: "team", content: newTeam }),
        }).catch(() => {});
      }
      setNewName("");
      setNewTeam("");
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

  const openTeamDialog = (t?: Team) => {
    if (t) {
      setEditingTeam(t);
      setTeamName(t.name);
      setTeamIcon(t.icon ?? "");
      setTeamColor(t.color ?? "");
      setTeamDesc(t.description ?? "");
    } else {
      setEditingTeam(null);
      setTeamName("");
      setTeamIcon("");
      setTeamColor("");
      setTeamDesc("");
    }
    setShowTeamDialog(true);
  };

  const handleSaveTeam = async () => {
    if (!teamName.trim()) return;
    setSavingTeam(true);
    try {
      const body = { name: teamName.trim(), icon: teamIcon.trim(), color: teamColor.trim(), description: teamDesc.trim() };
      if (editingTeam) {
        await fetch(goApiUrl(`/api/teams/${editingTeam.slug}`), {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
      } else {
        await fetch(goApiUrl("/api/teams"), {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
      }
      setShowTeamDialog(false);
      await fetchAll();
    } catch {
      // ignore
    } finally {
      setSavingTeam(false);
    }
  };

  const handleDeleteTeam = async (slug: string) => {
    await fetch(goApiUrl(`/api/teams/${slug}`), { method: "DELETE" }).catch(() => {});
    await fetchAll();
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
              onClick={() => openTeamDialog()}
              label="New Team"
              icon={<IconUsers size={16} stroke={2.2} />}
            />
            <ActionButton
              compact
              onClick={() => { setCreateError(""); setNewTeam(""); setShowNew(true); }}
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
        <div className="space-y-8">
          {grouped.map(({ team: t, agents: groupAgents }) => (
            <div key={t?.slug ?? "solo"}>
              {/* Team header */}
              <div className="flex items-center gap-2 mb-3">
                {t ? (
                  <>
                    {t.icon && <span className="text-base">{t.icon}</span>}
                    <button
                      onClick={() => openTeamDialog(t)}
                      className="hover:underline underline-offset-2 transition-colors"
                      style={t.color ? { color: t.color } : undefined}
                    >
                      <SectionLabel className="mb-0">{t.name}</SectionLabel>
                    </button>
                    <span className="text-[10px] text-muted-foreground/30 font-mono">{groupAgents.length}</span>
                  </>
                ) : (
                  <>
                    <SectionLabel className="mb-0 text-muted-foreground/30">No team</SectionLabel>
                    <span className="text-[10px] text-muted-foreground/20 font-mono">{groupAgents.length}</span>
                  </>
                )}
              </div>
              {/* Agent table */}
              <DataTable<EnrichedAgent>
                rows={groupAgents}
                rowKey={(a) => a.name}
                rowHref={(a) => a.worldID ? `/agents/${encodeURIComponent(a.name)}?world=${a.worldID}` : `/agents/${encodeURIComponent(a.name)}`}
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
                        <StatusDot status={a.status === "limbo" ? "stopped" : a.status} />
                        <span className="text-[11px] font-mono text-muted-foreground/50 capitalize">{a.status}</span>
                      </span>
                    ),
                  },
                  {
                    key: "world",
                    label: "World",
                    width: "120px",
                    render: (a) => a.worldName
                      ? <span className="text-[11px] font-mono text-foreground/60 truncate">{a.worldName}</span>
                      : <span className="text-[11px] font-mono text-muted-foreground/25">—</span>,
                  },
                ]}
              />
            </div>
          ))}
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
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mt-4 mb-1.5">
              Team <span className="text-muted-foreground/25 normal-case tracking-normal">(optional)</span>
            </label>
            <select
              value={newTeam}
              onChange={(e) => setNewTeam(e.target.value)}
              disabled={creating}
              className="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.16] transition-colors disabled:opacity-50"
            >
              <option value="">No team</option>
              {teams.map((t) => (
                <option key={t.slug} value={t.slug}>{t.icon ? `${t.icon} ` : ""}{t.name}</option>
              ))}
            </select>
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
      {/* Team create/edit dialog */}
      {showTeamDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => !savingTeam && setShowTeamDialog(false)} />
          <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
            <h3 className="text-lg font-heading text-foreground/90 mb-1">
              {editingTeam ? "Edit Team" : "New Team"}
            </h3>
            <p className="text-sm text-muted-foreground/50 mb-5">
              {editingTeam ? `Editing ${editingTeam.name}` : "Create a new team to group agents together."}
            </p>
            <div className="space-y-3">
              <div>
                <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">Name</label>
                <input
                  value={teamName}
                  onChange={(e) => setTeamName(e.target.value)}
                  placeholder="e.g. Matrix Ops"
                  autoFocus
                  onKeyDown={(e) => { if (e.key === "Enter") handleSaveTeam(); }}
                  className="w-full px-3 py-2.5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-foreground/80 placeholder:text-muted-foreground/30 focus:outline-none focus:border-white/[0.16] transition-colors"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">Icon</label>
                  <input
                    value={teamIcon}
                    onChange={(e) => setTeamIcon(e.target.value)}
                    placeholder="⬡"
                    className="w-full px-3 py-2.5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-foreground/80 placeholder:text-muted-foreground/30 focus:outline-none focus:border-white/[0.16] transition-colors"
                  />
                </div>
                <div>
                  <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">Color</label>
                  <input
                    value={teamColor}
                    onChange={(e) => setTeamColor(e.target.value)}
                    placeholder="#8B5CF6"
                    className="w-full px-3 py-2.5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm font-mono text-foreground/80 placeholder:text-muted-foreground/30 focus:outline-none focus:border-white/[0.16] transition-colors"
                  />
                </div>
              </div>
              <div>
                <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                  Description <span className="text-muted-foreground/25 normal-case tracking-normal">(optional)</span>
                </label>
                <input
                  value={teamDesc}
                  onChange={(e) => setTeamDesc(e.target.value)}
                  placeholder="What this team does…"
                  className="w-full px-3 py-2.5 rounded-lg bg-white/[0.04] border border-white/[0.08] text-sm text-foreground/80 placeholder:text-muted-foreground/30 focus:outline-none focus:border-white/[0.16] transition-colors"
                />
              </div>
            </div>
            <div className="flex items-center justify-between mt-6">
              <div>
                {editingTeam && (
                  <button
                    onClick={() => { handleDeleteTeam(editingTeam.slug); setShowTeamDialog(false); }}
                    className="text-[11px] text-red-400/60 hover:text-red-400 transition-colors"
                  >
                    Delete team
                  </button>
                )}
              </div>
              <div className="flex gap-3">
                <button
                  onClick={() => setShowTeamDialog(false)}
                  disabled={savingTeam}
                  className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSaveTeam}
                  disabled={savingTeam || !teamName.trim()}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm bg-white/[0.1] text-foreground/90 hover:bg-white/[0.16] border border-white/[0.08] transition-colors disabled:opacity-50"
                >
                  {savingTeam ? "Saving…" : editingTeam ? "Save" : "Create"}
                </button>
              </div>
            </div>
            <button
              aria-label="Close"
              onClick={() => !savingTeam && setShowTeamDialog(false)}
              className="absolute top-4 right-4 text-muted-foreground/30 hover:text-foreground/60 transition-colors"
              disabled={savingTeam}
            >
              <IconX size={16} />
            </button>
          </div>
        </div>
      )}
    </Page>
  );
}
