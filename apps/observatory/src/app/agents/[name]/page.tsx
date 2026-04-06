"use client";

import { useParams, useRouter } from "next/navigation";
import { useState, useEffect, useCallback, useRef } from "react";
import type { AgentProfile } from "@/lib/types";
import { apiGet, apiPut, apiAction, apiDelete, goApiUrl, encPath } from "@/lib/api-client";
import { useRefetch } from "@/components/app-shell";
import { streamChat } from "@/lib/stream-chat";
import { Chat, ChatSuggestions, type ChatBubble } from "@/components/chat";
import { InlineEdit, InlineTagsEdit } from "@/components/inline-edit";
import { TIER_BADGE } from "@/lib/status";
import {
  IconBrain,
  IconSparkles,
  IconBook,
  IconFileText,
  IconNotebook,
  IconUsers,
  IconRefresh,
  IconGitFork,
  IconDownload,
  IconUser,
  IconRocket,
  IconPlanet,
  IconTrash,
  IconFolder,
  IconFolderOpen,
  IconChevronRight,
  IconChevronDown,
  IconFile,
  IconMessageCircle,
  IconTerminal,
  IconX,
} from "@tabler/icons-react";
import { Skeleton } from "@/components/ui/skeleton";
import { usePageTitle } from "@/hooks/use-page-title";
import { ActionButton } from "@/components/action-button";
import { PageHeader } from "@/components/page-header";
import { getWorldName, type Team, type World } from "@/lib/types";

export default function AgentProfilePage() {
  const params = useParams();
  const router = useRouter();
  const agentName = decodeURIComponent(params.name as string);

  const [profile, setProfile] = useState<AgentProfile | null>(null);
  const [mindTree, setMindTree] = useState<Record<string, string[]>>({});
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [activeTab, setActiveTab] = useState<"profile" | "chat" | "files">("profile");
  const [showWizard, setShowWizard] = useState(false);
  const [showDeployDialog, setShowDeployDialog] = useState(false);
  const [availableTeams, setAvailableTeams] = useState<Team[]>([]);
  const [availableWorlds, setAvailableWorlds] = useState<World[]>([]);
  const [deployTargetWorld, setDeployTargetWorld] = useState("");
  const [deploying, setDeploying] = useState(false);
  const [deployError, setDeployError] = useState("");
  const refetchSidebar = useRefetch();

  usePageTitle(agentName, "Agent");

  const fetchProfile = useCallback(() => {
    Promise.all([
      apiGet<AgentProfile>(`/api/agents/${encPath(agentName)}`).catch(() => null),
      apiGet<Record<string, string[]>>(`/api/agents/${agentName}/mind`).catch(() => null),
    ]).then(([agentProfile, tree]) => {
      setProfile(agentProfile ?? null);
      setMindTree(tree ?? {});
      setLoading(false);
      // Show wizard for new agents without a purpose
      if (agentProfile && !agentProfile.purpose) {
        setShowWizard(true);
      }
    });
  }, [agentName]);

  useEffect(() => {
    fetchProfile();
    apiGet<Team[]>("/api/teams").then((t) => setAvailableTeams(t ?? [])).catch(() => {});
    apiGet<World[]>("/api/universes").then((w) => setAvailableWorlds(w ?? [])).catch(() => {});
  }, [fetchProfile]);

  const showFeedback = (msg: string) => {
    setFeedback(msg);
    setTimeout(() => setFeedback(null), 2500);
  };

  const saveIdentityField = async (field: string, content: string): Promise<boolean> => {
    try {
      await apiPut(`/api/agents/${agentName}/identity`, { field, content });
      showFeedback(`${field} updated`);
      fetchProfile(); // Refresh data
      return true;
    } catch {
      showFeedback(`Error: failed to update ${field}`);
      return false;
    }
  };

  const callAction = async (action: string, body?: object): Promise<boolean> => {
    setActionLoading(action);
    try {
      const result = await apiAction(
        `/api/agents/${agentName}/${action}`,
        body,
      );
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

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await apiDelete(`/api/agents/${encPath(agentName)}`);
      router.push("/");
    } catch {
      showFeedback("Error: Failed to delete agent");
      setDeleting(false);
      setShowDeleteConfirm(false);
    }
  };

  const handleDeploy = async () => {
    if (!deployTargetWorld) return;
    setDeploying(true);
    setDeployError("");
    try {
      const res = await fetch(goApiUrl(`/api/worlds/${deployTargetWorld}/agents`), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: agentName, tier: "citizen" }),
        signal: AbortSignal.timeout(30000),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setDeployError(data.error || `Deploy failed (HTTP ${res.status})`);
        setDeploying(false);
        return;
      }
      refetchSidebar();
      setShowDeployDialog(false);
      router.push(`/world/${deployTargetWorld}/${agentName}`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Unknown error";
      setDeployError(`Failed to connect: ${msg}`);
      setDeploying(false);
    }
  };

  if (loading) {
    return (
      <div className="p-8 space-y-6 max-w-3xl">
        <div className="flex items-center gap-4">
          <Skeleton className="w-12 h-12 rounded-xl" />
          <div className="space-y-2">
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-3 w-48" />
          </div>
        </div>
        <Skeleton className="h-24 w-full rounded-xl" />
        <Skeleton className="h-16 w-full rounded-xl" />
        <div className="flex gap-2">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-8 w-24 rounded-full" />
          ))}
        </div>
        <Skeleton className="h-32 w-full rounded-xl" />
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="p-8 flex flex-col items-center justify-center min-h-[60vh]">
        <div className="w-16 h-16 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-4">
          <IconUser size={28} className="text-muted-foreground/20" />
        </div>
        <p className="text-muted-foreground/50 text-lg font-heading">Agent &quot;{agentName}&quot; not found</p>
        <p className="text-xs text-muted-foreground/30 mt-2 font-mono">
          Create this agent with: spwn agent create {agentName}
        </p>
        <button
          onClick={() => {
            apiAction("/api/agents", { name: agentName }).then((result) => {
              if (result.ok) {
                fetchProfile();
                showFeedback("Agent created!");
              } else {
                showFeedback(`Error: ${result.error}`);
              }
            });
          }}
          className="mt-6 flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm bg-white/[0.04] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.06] transition-all"
        >
          <IconRocket size={16} />
          Create &quot;{agentName}&quot;
        </button>
      </div>
    );
  }

  const totalFiles = Object.values(mindTree).reduce((n, f) => n + (f?.length ?? 0), 0);
  const activeLayers = Object.keys(mindTree).filter((k) => (mindTree[k]?.length ?? 0) > 0).length;

  const tierStyle = TIER_BADGE[profile.tier] ?? TIER_BADGE.citizen;

  return (
    <div className="p-4 md:p-8 space-y-6 md:space-y-8 max-w-3xl">
      <PageHeader
        title={agentName}
        description={`${profile.engine} · ${profile.provider} · ${profile.tier}`}
        actions={
          <>
            <ActionButton
              compact
              onClick={async () => {
                if ((profile?.journal?.length ?? 0) === 0) {
                  showFeedback("Nothing to dream about yet — spawn the agent in a world first");
                  return;
                }
                const ok = await callAction("dream");
                if (ok) {
                  showFeedback("Dream cycle complete — check playbooks for promoted patterns");
                  fetchProfile();
                }
              }}
              disabled={actionLoading !== null}
              label="Dream"
              icon={<IconRefresh size={16} stroke={2.2} />}
            />
            <ActionButton
              compact
              onClick={async () => {
                const target = prompt("Fork target name:");
                if (!target) return;
                const ok = await callAction("fork", { target });
                if (ok) showFeedback(`Forked to "${target}"`);
              }}
              disabled={actionLoading !== null}
              label="Fork"
              icon={<IconGitFork size={16} stroke={2.2} />}
            />
            <ActionButton
              compact
              onClick={async () => {
                const ok = await callAction("export");
                if (ok) showFeedback("Export complete!");
              }}
              disabled={actionLoading !== null}
              label="Export"
              icon={<IconDownload size={16} stroke={2.2} />}
            />
            <ActionButton
              compact
              onClick={() => { setDeployError(""); setShowDeployDialog(true); }}
              disabled={actionLoading !== null || deploying}
              label="Deploy"
              icon={<IconPlanet size={16} stroke={2.2} />}
            />
            <ActionButton
              compact
              danger
              onClick={() => setShowDeleteConfirm(true)}
              disabled={actionLoading !== null || deleting}
              label="Delete"
              icon={<IconTrash size={16} stroke={2.2} />}
            />
          </>
        }
      />

      {/* Deploy dialog — select a running world to deploy this agent into */}
      {showDeployDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => !deploying && setShowDeployDialog(false)} />
          <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
            <h3 className="text-lg font-heading text-foreground/90 mb-1">Deploy to World</h3>
            <p className="text-sm text-muted-foreground/50 mb-5">
              Add <span className="font-mono text-foreground/70">{agentName}</span> to a running world.
            </p>
            <label className="block text-[10px] uppercase tracking-[0.15em] text-muted-foreground/40 mb-2">Select World</label>
            {availableWorlds.length === 0 ? (
              <p className="text-[11px] text-muted-foreground/40 px-3 py-2.5 rounded-lg bg-white/[0.02] border border-white/[0.06]">
                No running worlds. Spawn one first from the Worlds page.
              </p>
            ) : (
              <div className="rounded-lg bg-white/[0.02] border border-white/[0.08] max-h-48 overflow-y-auto">
                {availableWorlds.map((w) => {
                  const wName = getWorldName(w);
                  const isSelected = deployTargetWorld === w.id;
                  const alreadyDeployed = w.agents.some((a) => a.name === agentName);
                  return (
                    <button
                      key={w.id}
                      onClick={() => !alreadyDeployed && setDeployTargetWorld(w.id)}
                      disabled={alreadyDeployed}
                      className={`w-full flex items-center gap-3 px-3 py-2.5 text-left transition-colors ${
                        alreadyDeployed
                          ? "opacity-40 cursor-not-allowed"
                          : isSelected
                            ? "bg-white/[0.06]"
                            : "hover:bg-white/[0.03]"
                      }`}
                    >
                      <span className={`w-2 h-2 rounded-full shrink-0 ${
                        isSelected ? "bg-emerald-400" : "bg-white/[0.15]"
                      }`} />
                      <span className="flex-1 min-w-0">
                        <span className="text-sm text-foreground/80 truncate block">{wName}</span>
                        <span className="text-[10px] text-muted-foreground/35 font-mono">
                          {w.agents.length} agent{w.agents.length === 1 ? "" : "s"}
                          {alreadyDeployed && " · already deployed"}
                        </span>
                      </span>
                    </button>
                  );
                })}
              </div>
            )}
            {deployError && (
              <p className="text-xs text-red-400/80 mt-3">{deployError}</p>
            )}
            <div className="flex gap-3 justify-end mt-6">
              <button
                onClick={() => setShowDeployDialog(false)}
                disabled={deploying}
                className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleDeploy}
                disabled={deploying || !deployTargetWorld}
                className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 border border-emerald-500/20 transition-colors disabled:opacity-50"
              >
                {deploying ? (
                  <>
                    <div className="w-3 h-3 border-2 border-emerald-300/40 border-t-emerald-300 rounded-full animate-spin" />
                    Deploying…
                  </>
                ) : (
                  <>
                    <IconPlanet size={14} />
                    Deploy
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation dialog */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => setShowDeleteConfirm(false)} />
          <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
            <h3 className="text-lg font-heading text-foreground/90 mb-2">Delete Agent</h3>
            <p className="text-sm text-muted-foreground/50 mb-6">
              Are you sure you want to delete <span className="font-mono text-foreground/70">{agentName}</span>? This will permanently remove all mind files, memories, and identity data.
            </p>
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setShowDeleteConfirm(false)}
                className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="px-4 py-2 rounded-lg text-sm bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/20 transition-colors disabled:opacity-50"
              >
                {deleting ? "Deleting..." : "Delete Agent"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Get Started Wizard Banner */}
      {showWizard && profile && !profile.purpose && (
        <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-5 animate-in fade-in slide-in-from-top-2 duration-300">
          <div className="flex items-start justify-between mb-3">
            <div>
              <h3 className="text-sm font-heading text-emerald-300">
                Welcome to {agentName}! Let&apos;s set up their identity.
              </h3>
              <p className="text-[11px] text-emerald-300/50 mt-1">
                Fill in at least a purpose to get started.
              </p>
            </div>
            <button
              onClick={() => setShowWizard(false)}
              className="text-emerald-300/30 hover:text-emerald-300/60 transition-colors"
            >
              <IconX size={16} />
            </button>
          </div>
          <div className="space-y-3">
            <div>
              <label className="text-[10px] uppercase tracking-widest text-emerald-300/40 block mb-1.5">
                Purpose
              </label>
              <input
                placeholder="What is this agent's purpose?"
                className="w-full bg-white/[0.03] border border-emerald-500/15 rounded-lg px-3 py-2.5 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-emerald-500/30 transition-colors"
                onKeyDown={async (e) => {
                  if (e.key === "Enter") {
                    const val = (e.target as HTMLInputElement).value.trim();
                    if (val) {
                      await saveIdentityField("purpose", val);
                      setShowWizard(false);
                    }
                  }
                }}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-[10px] uppercase tracking-widest text-emerald-300/40 block mb-1.5">
                  Persona
                </label>
                <input
                  placeholder="Describe their persona..."
                  className="w-full bg-white/[0.03] border border-emerald-500/15 rounded-lg px-3 py-2 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-emerald-500/30 transition-colors"
                  onKeyDown={async (e) => {
                    if (e.key === "Enter") {
                      const val = (e.target as HTMLInputElement).value.trim();
                      if (val) await saveIdentityField("persona", val);
                    }
                  }}
                />
              </div>
              <div>
                <label className="text-[10px] uppercase tracking-widest text-emerald-300/40 block mb-1.5">
                  Traits
                </label>
                <input
                  placeholder="e.g. curious, creative, diligent"
                  className="w-full bg-white/[0.03] border border-emerald-500/15 rounded-lg px-3 py-2 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-emerald-500/30 transition-colors"
                  onKeyDown={async (e) => {
                    if (e.key === "Enter") {
                      const val = (e.target as HTMLInputElement).value.trim();
                      if (val) {
                        const traits = val.split(",").map((t) => `- ${t.trim()}`).join("\n");
                        await saveIdentityField("traits", traits);
                      }
                    }
                  }}
                />
              </div>
            </div>
            <p className="text-[10px] text-emerald-300/30 font-mono">
              Press Enter in any field to save. Fill purpose to dismiss this banner.
            </p>
          </div>
        </div>
      )}

      {/* Tab switcher */}
      <div className="flex gap-1 border-b border-white/[0.06] pb-px">
        <button
          onClick={() => setActiveTab("profile")}
          className={`px-4 py-2 text-xs font-medium transition-colors border-b-2 -mb-px ${
            activeTab === "profile"
              ? "border-foreground/50 text-foreground/80"
              : "border-transparent text-muted-foreground/40 hover:text-muted-foreground/60"
          }`}
        >
          Profile
        </button>
        <button
          onClick={() => setActiveTab("chat")}
          className={`px-4 py-2 text-xs font-medium transition-colors border-b-2 -mb-px flex items-center gap-1.5 ${
            activeTab === "chat"
              ? "border-foreground/50 text-foreground/80"
              : "border-transparent text-muted-foreground/40 hover:text-muted-foreground/60"
          }`}
        >
          <IconMessageCircle size={13} />
          Chat
        </button>
        <button
          onClick={() => setActiveTab("files")}
          className={`px-4 py-2 text-xs font-medium transition-colors border-b-2 -mb-px ${
            activeTab === "files"
              ? "border-foreground/50 text-foreground/80"
              : "border-transparent text-muted-foreground/40 hover:text-muted-foreground/60"
          }`}
        >
          Files ({totalFiles})
        </button>
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

      {/* Profile tab */}
      {activeTab === "profile" && (<>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="glass-subtle p-3 text-center">
          <p className="text-lg font-heading text-foreground/80">{totalFiles}</p>
          <p className="text-[9px] text-muted-foreground/35 uppercase">Files</p>
        </div>
        <div className="glass-subtle p-3 text-center">
          <p className="text-lg font-heading text-foreground/80">{activeLayers}</p>
          <p className="text-[9px] text-muted-foreground/35 uppercase">Layers</p>
        </div>
        <div className="glass-subtle p-3 text-center">
          <p className="text-lg font-heading text-foreground/80">{profile.journal?.length ?? 0}</p>
          <p className="text-[9px] text-muted-foreground/35 uppercase">Journal</p>
        </div>
        <div className="glass-subtle p-3 text-center">
          <p className="text-lg font-heading text-foreground/80">{profile.bonds?.length ?? 0}</p>
          <p className="text-[9px] text-muted-foreground/35 uppercase">Bonds</p>
        </div>
      </div>

      {/* Team selector */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-2">Team</h2>
        <select
          value={profile.team ?? ""}
          onChange={async (e) => {
            const ok = await saveIdentityField("team", e.target.value);
            if (ok) fetchProfile();
          }}
          className="bg-white/[0.04] border border-white/[0.08] rounded-lg px-3 py-2 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.16] transition-colors"
        >
          <option value="">No team</option>
          {availableTeams.map((t) => (
            <option key={t.slug} value={t.slug}>{t.icon ? `${t.icon} ` : ""}{t.name}</option>
          ))}
        </select>
      </div>

      {/* Deployment history — derived from sessions */}
      {(() => {
        const sessionFiles = mindTree["sessions"] ?? [];
        const worldIds = sessionFiles
          .filter((f) => f.endsWith(".json"))
          .map((f) => f.replace(".json", ""));
        if (worldIds.length === 0) return null;
        return (
          <div>
            <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-2 flex items-center gap-1.5">
              Worlds
            </h2>
            <div className="flex flex-wrap gap-1.5">
              {worldIds.map((wid) => {
                const parts = wid.split("-");
                const worldName = parts.length >= 2 ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1) : wid;
                return (
                  <a
                    key={wid}
                    href={`/world/${wid}/${agentName}`}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-[11px] font-mono bg-white/[0.04] text-muted-foreground/50 border border-white/[0.06] hover:bg-white/[0.08] hover:text-foreground/80 transition-all"
                  >
                    <span className="w-1.5 h-1.5 rounded-full bg-foreground/20" />
                    {worldName}
                  </a>
                );
              })}
            </div>
          </div>
        );
      })()}

      {/* Purpose — inline editable */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
          <IconSparkles size={12} />
          Purpose
        </h2>
        <div className="glass-subtle p-4">
          <InlineEdit
            value={profile.purpose || ""}
            placeholder="Define this agent's purpose..."
            onSave={(v) => saveIdentityField("purpose", v)}
            multiline
            className="text-sm text-foreground/70 leading-relaxed"
          />
        </div>
      </div>

      {/* Persona — inline editable */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Persona</h2>
        <div className="glass-subtle p-4">
          <InlineEdit
            value={profile.persona || ""}
            placeholder="Describe the agent's persona..."
            onSave={(v) => saveIdentityField("persona", v)}
            multiline
            className="text-sm text-foreground/60 leading-relaxed italic"
          />
        </div>
      </div>

      {/* Traits — inline tag editing */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Traits</h2>
        <InlineTagsEdit
          tags={profile.traits ?? []}
          onSave={async (tags) => {
            const ok = await saveIdentityField("traits", tags.map((t) => `- ${t}`).join("\n"));
            return ok;
          }}
          color="bg-purple-500/10 text-purple-300/80 border-purple-500/20"
        />
      </div>

      {/* Skills */}
      {(profile.skills?.length ?? 0) > 0 && (
        <div>
          <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
            <IconBrain size={12} />
            Skills
          </h2>
          <div className="flex flex-wrap gap-2">
            {(profile.skills ?? []).map((skill) => (
              <span
                key={skill}
                className="px-2.5 py-1 rounded-full text-[11px] font-mono bg-blue-500/10 text-blue-300/80 border border-blue-500/20"
              >
                {skill}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Playbooks */}
      {(profile.playbooks?.length ?? 0) > 0 && (
        <div>
          <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
            <IconBook size={12} />
            Playbooks
          </h2>
          <div className="glass-subtle divide-y divide-border/20">
            {(profile.playbooks ?? []).map((pb) => (
              <div key={pb} className="px-4 py-2.5 flex items-center gap-2">
                <IconFileText size={13} className="text-muted-foreground/30 shrink-0" />
                <span className="text-xs font-mono text-foreground/60">{pb}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Knowledge */}
      {(profile.knowledge?.length ?? 0) > 0 && (
        <div>
          <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Knowledge Files</h2>
          <div className="glass-subtle divide-y divide-border/20">
            {(profile.knowledge ?? []).map((k) => (
              <div key={k} className="px-4 py-2.5 flex items-center gap-2">
                <IconFileText size={13} className="text-muted-foreground/30 shrink-0" />
                <span className="text-xs font-mono text-foreground/60">{k}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Journal */}
      {(profile.journal?.length ?? 0) > 0 && (
        <div>
          <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
            <IconNotebook size={12} />
            Journal
          </h2>
          <div className="space-y-2">
            {(profile.journal ?? []).map((entry) => (
              <div key={entry.date} className="glass-subtle p-4">
                <p className="text-[10px] font-mono text-muted-foreground/40 mb-1.5">{entry.date}</p>
                <p className="text-xs text-foreground/60 leading-relaxed">{entry.summary}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Bonds */}
      {(profile.bonds?.length ?? 0) > 0 && (
        <div>
          <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
            <IconUsers size={12} />
            Bonds
          </h2>
          <div className="glass-subtle divide-y divide-border/20">
            {(profile.bonds ?? []).map((bond) => (
              <div key={bond.agent} className="px-4 py-3 flex items-center justify-between">
                <span className="text-xs font-mono text-foreground/70">{bond.agent}</span>
                <span className="text-[10px] text-muted-foreground/40 italic">{bond.relationship}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Commands */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Commands</h2>
        <div className="glass-subtle p-3 font-mono text-[10px] text-muted-foreground/35 space-y-1">
          <p>spwn agent talk {agentName} &quot;message&quot;</p>
          <p>spwn agent dream {agentName}</p>
          <p>spwn agent sleep {agentName}</p>
          <p>spwn profile {agentName}</p>
          <p>spwn agent fork {agentName} &lt;new&gt;</p>
          <p>spwn agent export {agentName}</p>
        </div>
      </div>

      </>)}

      {/* Chat tab */}
      {activeTab === "chat" && (
        <AgentChat agentName={agentName} />
      )}

      {/* Files tab */}
      {activeTab === "files" && (
        <MindFileViewer agentName={agentName} mindTree={mindTree} />
      )}
    </div>
  );
}

/* ── Agent Chat ── */

function AgentChat({ agentName }: { agentName: string }) {
  const [messages, setMessages] = useState<ChatBubble[]>([]);
  const [sending, setSending] = useState(false);

  const handleSend = async (msg: string) => {
    setMessages((prev) => [
      ...prev,
      { role: "user", content: msg, blocks: [{ type: "text", content: msg }], timestamp: new Date() },
    ]);
    setSending(true);

    const assistantIndex = messages.length + 1;
    setMessages((prev) => [
      ...prev,
      { role: "assistant", content: "", blocks: [], timestamp: new Date() },
    ]);

    await streamChat({
      url: goApiUrl(`/api/agents/${agentName}/talk`),
      body: { message: msg },
      onBlocks: (newBlocks) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[assistantIndex];
          if (last && last.role === "assistant") {
            const allBlocks = [...last.blocks, ...newBlocks];
            const textContent = allBlocks
              .filter((b): b is { type: "text"; content: string } => b.type === "text")
              .map((b) => b.content)
              .join("");
            updated[assistantIndex] = { ...last, blocks: allBlocks, content: textContent };
          }
          return updated;
        });
      },
      onDone: (meta) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[assistantIndex];
          if (last && last.role === "assistant") {
            updated[assistantIndex] = { ...last, cost: meta.cost, duration: meta.duration };
          }
          return updated;
        });
      },
      onError: (error) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[assistantIndex];
          if (last && last.role === "assistant") {
            updated[assistantIndex] = {
              ...last,
              content: error,
              blocks: [{ type: "error", content: error }],
              error: true,
            };
          }
          return updated;
        });
      },
    });

    setSending(false);
  };

  return (
    <Chat
      messages={messages}
      onSend={handleSend}
      disabled={sending}
      placeholder={`Message ${agentName}...`}
      assistantLabel={agentName}
      typingText={`${agentName} is thinking…`}
      autoFocus
      className="h-[520px]"
      emptyState={
        <div className="flex flex-col items-center justify-center text-center">
          <IconTerminal size={28} className="text-muted-foreground/15 mb-3" />
          <p className="text-sm text-muted-foreground/30">Chat with {agentName}</p>
          <p className="text-[11px] text-muted-foreground/20 mt-1 mb-4">
            Send messages directly to this agent in real-time
          </p>
          <ChatSuggestions
            suggestions={["What are you working on?", "Show me the project structure", "Run the tests"]}
            onPick={(s) => handleSend(s)}
          />
        </div>
      }
    />
  );
}

/* ── Mind File Viewer ── */

const LAYER_ICONS: Record<string, typeof IconFolder> = {
  identity: IconUser,
  skills: IconBrain,
  "memory/knowledge": IconBook,
  "memory/playbooks": IconBook,
  "memory/journal": IconNotebook,
  sessions: IconFileText,
};

function MindFileViewer({ agentName, mindTree }: { agentName: string; mindTree: Record<string, string[]> }) {
  const [expandedLayers, setExpandedLayers] = useState<Set<string>>(new Set());
  const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());
  const [fileContents, setFileContents] = useState<Record<string, string>>({});
  const [loadingFiles, setLoadingFiles] = useState<Set<string>>(new Set());

  const toggleLayer = (layer: string) => {
    setExpandedLayers((prev) => {
      const next = new Set(prev);
      if (next.has(layer)) next.delete(layer);
      else next.add(layer);
      return next;
    });
  };

  const toggleFile = async (layer: string, file: string) => {
    const key = `${layer}/${file}`;
    const fullPath = file.endsWith(".md") ? `${layer}/${file}` : `${layer}/${file}.md`;

    if (expandedFiles.has(key)) {
      setExpandedFiles((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
      return;
    }

    setExpandedFiles((prev) => new Set(prev).add(key));

    // Fetch content if not cached
    if (!fileContents[key]) {
      setLoadingFiles((prev) => new Set(prev).add(key));
      try {
        const res = await fetch(goApiUrl(`/api/agents/${agentName}/files/${fullPath}`));
        if (res.ok) {
          const data = await res.json();
          setFileContents((prev) => ({ ...prev, [key]: data.content }));
        } else {
          setFileContents((prev) => ({ ...prev, [key]: "⚠ Failed to load file content" }));
        }
      } catch {
        setFileContents((prev) => ({ ...prev, [key]: "⚠ Failed to connect to API" }));
      } finally {
        setLoadingFiles((prev) => {
          const next = new Set(prev);
          next.delete(key);
          return next;
        });
      }
    }
  };

  const sortedLayers = Object.keys(mindTree).sort((a, b) => {
    const order = ["identity", "skills", "memory/knowledge", "memory/playbooks", "memory/journal", "sessions"];
    return (order.indexOf(a) === -1 ? 99 : order.indexOf(a)) - (order.indexOf(b) === -1 ? 99 : order.indexOf(b));
  });

  if (sortedLayers.length === 0) {
    return (
      <div className="text-center py-12">
        <IconFolder size={32} className="mx-auto text-muted-foreground/15 mb-3" />
        <p className="text-muted-foreground/40 text-sm">No mind files found</p>
        <p className="text-muted-foreground/25 text-xs mt-1 font-mono">Create files with: spwn agent dream {agentName}</p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {sortedLayers.map((layer) => {
        const files = mindTree[layer] ?? [];
        const isExpanded = expandedLayers.has(layer);
        const LayerIcon = LAYER_ICONS[layer] ?? IconFolder;

        return (
          <div key={layer} className="glass-subtle overflow-hidden">
            {/* Layer header */}
            <button
              onClick={() => toggleLayer(layer)}
              className="w-full flex items-center gap-2.5 px-4 py-3 text-left hover:bg-white/[0.02] transition-colors"
            >
              {isExpanded ? (
                <IconChevronDown size={14} className="text-muted-foreground/40 shrink-0" />
              ) : (
                <IconChevronRight size={14} className="text-muted-foreground/40 shrink-0" />
              )}
              {isExpanded ? (
                <IconFolderOpen size={16} className="text-foreground/50 shrink-0" />
              ) : (
                <LayerIcon size={16} className="text-foreground/40 shrink-0" />
              )}
              <span className="text-xs font-mono text-foreground/70 flex-1">{layer}/</span>
              <span className="text-[10px] font-mono text-muted-foreground/30">{files.length} files</span>
            </button>

            {/* Files list */}
            {isExpanded && files.length > 0 && (
              <div className="border-t border-white/[0.04]">
                {files.map((file) => {
                  const key = `${layer}/${file}`;
                  const isFileExpanded = expandedFiles.has(key);
                  const isLoading = loadingFiles.has(key);
                  const content = fileContents[key];

                  return (
                    <div key={file}>
                      <button
                        onClick={() => toggleFile(layer, file)}
                        className="w-full flex items-center gap-2.5 px-4 py-2 pl-10 text-left hover:bg-white/[0.02] transition-colors"
                      >
                        {isFileExpanded ? (
                          <IconChevronDown size={12} className="text-muted-foreground/30 shrink-0" />
                        ) : (
                          <IconChevronRight size={12} className="text-muted-foreground/30 shrink-0" />
                        )}
                        <IconFile size={13} className="text-muted-foreground/30 shrink-0" />
                        <span className="text-[11px] font-mono text-foreground/60">{file}</span>
                      </button>

                      {/* File content */}
                      {isFileExpanded && (
                        <div className="px-4 py-3 pl-16 border-t border-white/[0.03] bg-white/[0.01]">
                          {isLoading ? (
                            <div className="flex items-center gap-2 text-muted-foreground/30 text-xs">
                              <div className="w-3 h-3 border-2 border-foreground/20 border-t-foreground/50 rounded-full animate-spin" />
                              Loading...
                            </div>
                          ) : (
                            <pre className="text-[11px] font-mono text-foreground/50 whitespace-pre-wrap leading-relaxed overflow-x-auto max-h-96 overflow-y-auto">
                              {content ?? "No content"}
                            </pre>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
