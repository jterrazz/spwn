"use client";

import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useState, useEffect, useCallback, useRef, Suspense } from "react";
import type { AgentProfile } from "@/lib/types";
import { apiGet, apiPut, apiAction, apiDelete, goApiUrl, encPath } from "@/lib/api-client";
import { useRefetch } from "@/components/app-shell";
import { streamChat } from "@/lib/stream-chat";
import { Chat, ChatSuggestions, type ChatBubble } from "@/components/chat";
import { InlineEdit, InlineTagsEdit } from "@/components/inline-edit";
import { ROLE_BADGE } from "@/lib/status";
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
import { useProgressMessages } from "@/hooks/use-progress-messages";
import { ProgressShimmer } from "@/components/progress-shimmer";
import { ActionButton } from "@/components/action-button";
import { PageHeader } from "@/components/page-header";
import { SectionHeader, SectionLabel, SubLabel, Separator, MetricGrid, ItemList, StatusDot, KeyValue } from "@/components/ds";
import { getWorldName, AVAILABLE_ROLES, type Organization, type Team, type World } from "@/lib/types";

export default function AgentProfilePageWrapper() {
  return (
    <Suspense>
      <AgentProfilePage />
    </Suspense>
  );
}

function AgentProfilePage() {
  const params = useParams();
  const router = useRouter();
  const searchParams = useSearchParams();
  const agentName = decodeURIComponent(params.name as string);
  const worldId = searchParams.get("world") ?? undefined;

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
  const [deployRole, setDeployRole] = useState("worker");
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [worldData, setWorldData] = useState<World | null>(null);
  const refetchSidebar = useRefetch();

  const deployProgressMessage = useProgressMessages(deploying, [
    { after: 0, text: "Deploying agent..." },
    { after: 5, text: "Building Docker image (first run takes a few minutes)..." },
    { after: 30, text: "Still building... installing dependencies..." },
    { after: 60, text: "Almost there..." },
  ]);

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
    apiGet<World[]>("/api/worlds").then((w) => setAvailableWorlds(w ?? [])).catch(() => {});
    apiGet<Organization[]>("/api/organizations").then((h) => setOrganizations(h ?? [])).catch(() => {});
    if (worldId) {
      apiGet<World>(`/api/worlds/${worldId}`).then((w) => setWorldData(w ?? null)).catch(() => {});
    }
  }, [fetchProfile, worldId]);

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
        body: JSON.stringify({ name: agentName, role: deployRole }),
        signal: AbortSignal.timeout(600000), // 10 min — first run may build Docker images
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setDeployError(data.error || `Deploy failed (HTTP ${res.status})`);
        setDeploying(false);
        return;
      }
      refetchSidebar();
      setShowDeployDialog(false);
      router.push(`/agents/${encPath(agentName)}?world=${deployTargetWorld}`);
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

  const roleStyle = ROLE_BADGE[profile.role] ?? ROLE_BADGE.worker;
  const deployedAgent = worldData?.agents.find((a) => a.name === agentName);
  const worldName = worldData ? getWorldName(worldData) : undefined;

  const mainContent = (
    <div className="flex-1 min-w-0 p-4 md:p-8 space-y-6 md:space-y-8">
      <PageHeader
        title={agentName}
        description={`${profile.engine} · ${profile.provider} · ${profile.role}`}
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
          <div className="relative z-10 w-full max-w-md mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl overflow-hidden">
            {/* Top shimmer bar */}
            {deploying && (
              <div className="w-full h-0.5 overflow-hidden bg-white/[0.04]">
                <div
                  className="h-full w-1/3 rounded-full bg-emerald-500/30"
                  style={{ animation: "progressSlide 1.5s ease-in-out infinite" }}
                />
              </div>
            )}
            <div className="p-6">
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
            {/* Role selector */}
            <label className="block text-[10px] uppercase tracking-[0.15em] text-muted-foreground/40 mt-4 mb-2">Role</label>
            <select
              value={deployRole}
              onChange={(e) => setDeployRole(e.target.value)}
              disabled={deploying}
              className="w-full bg-white/[0.04] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 focus:outline-none focus:border-white/[0.16] transition-colors disabled:opacity-50"
            >
              {(() => {
                // Try to get roles from the selected world's organization, fall back to AVAILABLE_ROLES
                const selectedWorld = availableWorlds.find((w) => w.id === deployTargetWorld);
                const worldOrganizationSlug = (selectedWorld as Record<string, unknown> | undefined)?.organization as string | undefined;
                const org = worldOrganizationSlug
                  ? organizations.find((h) => h.slug === worldOrganizationSlug)
                  : null;
                const roleNames = org
                  ? org.roles.map((r) => r.name)
                  : [...AVAILABLE_ROLES];
                return roleNames.map((r) => (
                  <option key={r} value={r}>{r}</option>
                ));
              })()}
            </select>
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
            <ProgressShimmer active={deploying} message={deployProgressMessage} />
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
                {deleting ? (
                  <span className="flex items-center gap-2">
                    <span className="w-3 h-3 border-2 border-red-400/40 border-t-red-400 rounded-full animate-spin" />
                    Deleting...
                  </span>
                ) : "Delete Agent"}
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

      {/* Profile tab — diagnostics panel style */}
      {activeTab === "profile" && (<>

      <MetricGrid columns={2} items={[
        { label: "Files", value: totalFiles },
        { label: "Journal", value: profile.journal?.length ?? 0 },
        { label: "Skills", value: profile.skills?.length ?? 0 },
        { label: "Traits", value: profile.traits?.length ?? 0 },
      ]} className="gap-x-8 gap-y-4" />

      <Separator />

      {/* Team */}
      <div className="flex items-center justify-between">
        <SubLabel>Team</SubLabel>
        <select
          value={profile.team ?? ""}
          onChange={async (e) => {
            const ok = await saveIdentityField("team", e.target.value);
            if (ok) fetchProfile();
          }}
          className="bg-transparent text-sm font-mono text-foreground/80 focus:outline-none cursor-pointer text-right"
        >
          <option value="">—</option>
          {availableTeams.map((t) => (
            <option key={t.slug} value={t.slug}>{t.name}</option>
          ))}
        </select>
      </div>

      {/* Deployment History */}
      {(() => {
        const journalFiles = mindTree["journal"] ?? [];
        const worldIds = journalFiles
          .filter((f) => f.endsWith(".json"))
          .map((f) => f.replace(".json", ""));
        if (worldIds.length === 0) return null;
        return (
          <>
            <Separator />
            <div>
              <SectionHeader>Deployment History</SectionHeader>
              <ItemList items={worldIds.map((wid) => {
                const parts = wid.split("-");
                const wName = parts.length >= 2 ? parts[1].charAt(0).toUpperCase() + parts[1].slice(1) : wid;
                return { name: wName, detail: wid, href: `/agents/${encPath(agentName)}?world=${wid}` };
              })} />
            </div>
          </>
        );
      })()}

      <Separator />

      {/* Identity */}
      <div>
        <SectionHeader>Identity</SectionHeader>

        <div className="mb-4">
          <SubLabel className="mb-1.5">Purpose</SubLabel>
          <InlineEdit
            value={profile.purpose || ""}
            placeholder="Define this agent's purpose..."
            onSave={(v) => saveIdentityField("purpose", v)}
            multiline
            className="text-sm text-foreground/75 leading-relaxed"
          />
        </div>

        <div className="mb-4">
          <SubLabel className="mb-1.5">Persona</SubLabel>
          <InlineEdit
            value={profile.persona || ""}
            placeholder="Describe the agent's persona..."
            onSave={(v) => saveIdentityField("persona", v)}
            multiline
            className="text-sm text-foreground/60 leading-relaxed italic"
          />
        </div>

        <div>
          <SubLabel className="mb-1.5">Traits</SubLabel>
          <InlineTagsEdit
            tags={profile.traits ?? []}
            onSave={async (tags) => {
              const ok = await saveIdentityField("traits", tags.map((t) => `- ${t}`).join("\n"));
              return ok;
            }}
            color="bg-white/[0.06] text-foreground/60 border-white/[0.08]"
          />
        </div>
      </div>

      {/* Skills */}
      {(profile.skills?.length ?? 0) > 0 && (
        <>
          <Separator />
          <div>
            <SectionHeader>Skills</SectionHeader>
            <div className="flex flex-wrap gap-1.5">
              {(profile.skills ?? []).map((skill) => (
                <span key={skill} className="px-2.5 py-1 text-[11px] font-mono text-foreground/60 bg-white/[0.04] border border-white/[0.06]">
                  {skill}
                </span>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Journal */}
      {(profile.journal?.length ?? 0) > 0 && (
        <>
          <Separator />
          <div>
            <SectionHeader>Journal</SectionHeader>
            <div className="space-y-3">
              {(profile.journal ?? []).map((entry) => (
                <div key={entry.date}>
                  <SubLabel className="mb-1">{entry.date}</SubLabel>
                  <p className="text-xs text-foreground/60 leading-relaxed">{entry.summary}</p>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {/* Reference */}
      <Separator />
      <div>
        <SectionHeader>Reference</SectionHeader>
        <div className="font-mono text-[10px] text-muted-foreground/30 space-y-1">
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
        <AgentChat agentName={agentName} worldId={worldId} />
      )}

      {/* Files tab */}
      {activeTab === "files" && (
        <MindFileViewer agentName={agentName} mindTree={mindTree} />
      )}
    </div>
  );

  return (
    <div className="flex h-[calc(100vh-1px)] overflow-hidden">
      <div className="flex-1 min-w-0 overflow-y-auto">
        {mainContent}
      </div>
      {/* ── Right: Quick info panel ── */}
      <div className="w-72 border-l border-border/30 overflow-y-auto shrink-0 hidden lg:block">
        <div className="p-5 space-y-6">
          <SectionHeader>Diagnostics</SectionHeader>

          {/* Identity */}
          <div>
            <SectionLabel>Identity</SectionLabel>
            <div className="space-y-2">
              <KeyValue label="Name" value={agentName} />
              <div className="flex items-center justify-between">
                <SubLabel>Role</SubLabel>
                {(() => {
                  const role = deployedAgent?.role ?? profile.role;
                  const badge = ROLE_BADGE[role] ?? ROLE_BADGE.default;
                  return (
                    <span className={`px-1.5 py-0.5 rounded text-[9px] font-mono uppercase tracking-wider border ${badge}`}>
                      {role}
                    </span>
                  );
                })()}
              </div>
              {deployedAgent && (
                <div className="flex items-center justify-between">
                  <SubLabel>Status</SubLabel>
                  <div className="flex items-center gap-1.5">
                    <StatusDot status={deployedAgent.status} />
                    <span className="text-xs font-mono font-medium text-foreground/80 capitalize">{deployedAgent.status}</span>
                  </div>
                </div>
              )}
              <KeyValue label="Engine" value={profile.engine} />
              <KeyValue label="Provider" value={profile.provider} />
              {profile.team && <KeyValue label="Team" value={profile.team} />}
            </div>
          </div>

          <Separator />

          {/* Stats */}
          <div>
            <SectionLabel>Metrics</SectionLabel>
            <MetricGrid columns={2} items={[
              { label: "Files", value: totalFiles },
              { label: "Journal", value: profile.journal?.length ?? 0 },
              { label: "Skills", value: profile.skills?.length ?? 0 },
              { label: "Traits", value: profile.traits?.length ?? 0 },
            ]} />
          </div>

          {/* World info — only when deployed */}
          {worldData && (
            <>
              <Separator />
              <div>
                <SectionLabel>World</SectionLabel>
                <div className="space-y-2">
                  <KeyValue label="Name" value={worldName ?? worldId!} />
                  <div className="flex items-center justify-between">
                    <SubLabel>Status</SubLabel>
                    <div className="flex items-center gap-1.5">
                      <StatusDot status={worldData.status} />
                      <span className="text-xs font-mono font-medium text-foreground/80 capitalize">{worldData.status}</span>
                    </div>
                  </div>
                  <KeyValue label="Agents" value={worldData.agents.length} />
                  {worldData.workspaces && worldData.workspaces.length > 0 && (
                    <KeyValue label="Workspaces" value={worldData.workspaces.length} />
                  )}
                </div>
              </div>
            </>
          )}

          <Separator />

          {/* Commands */}
          <div>
            <SectionLabel>Reference</SectionLabel>
            <div className="font-mono text-[10px] text-muted-foreground/30 space-y-1">
              <p>spwn agent talk {agentName} &quot;msg&quot;</p>
              <p>spwn agent dream {agentName}</p>
              <p>spwn agent sleep {agentName}</p>
              <p>spwn profile {agentName}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ── Agent Chat ── */

function AgentChat({ agentName, worldId }: { agentName: string; worldId?: string }) {
  const [messages, setMessages] = useState<ChatBubble[]>([]);
  const [sending, setSending] = useState(false);

  // Load conversation history when in world context
  useEffect(() => {
    if (!worldId) return;
    apiGet<{ sessions: { messages: { role: string; content: string; timestamp: string; type: string; toolName?: string; cost?: number; durationMs?: number }[] }[] }>(
      `/api/worlds/${worldId}/history?agent=${encPath(agentName)}`
    ).then((data) => {
      if (!data?.sessions?.length) return;
      const historyMsgs: ChatBubble[] = [];
      for (const session of data.sessions) {
        for (const msg of session.messages) {
          if (msg.type === "text" && msg.content) {
            historyMsgs.push({
              role: msg.role === "user" ? "user" : "assistant",
              content: msg.content,
              blocks: [{ type: "text", content: msg.content }],
              timestamp: new Date(msg.timestamp),
            });
          }
        }
      }
      if (historyMsgs.length > 0) setMessages(historyMsgs);
    }).catch(() => {});
  }, [worldId, agentName]);

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

    // When in world context, talk through the world endpoint (which routes to the deployed container).
    // Otherwise, use the standalone agent talk endpoint (limbo / direct).
    const chatUrl = worldId
      ? goApiUrl(`/api/worlds/${worldId}/talk`)
      : goApiUrl(`/api/agents/${encPath(agentName)}/talk`);
    const chatBody = worldId
      ? { message: msg, agent: agentName }
      : { message: msg };

    await streamChat({
      url: chatUrl,
      body: chatBody,
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
  core: IconUser,
  skills: IconBrain,
  knowledge: IconBook,
  playbooks: IconBook,
  journal: IconNotebook,
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
    const order = ["core", "skills", "knowledge", "playbooks", "journal"];
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
