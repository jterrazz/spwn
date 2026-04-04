"use client";

import { useParams, useRouter } from "next/navigation";
import { useState, useEffect, useCallback, useRef } from "react";
import type { AgentProfile } from "@/lib/types";
import { apiGet, apiPut, apiAction, apiDelete, goApiUrl } from "@/lib/api-client";
import { streamChat } from "@/lib/stream-chat";
import type { ActivityBlock } from "@/lib/activity-types";
import { ActivityMessageView } from "@/components/activity-blocks";
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
  IconTrash,
  IconFolder,
  IconFolderOpen,
  IconChevronRight,
  IconChevronDown,
  IconFile,
  IconMessageCircle,
  IconSend,
  IconTerminal,
  IconX,
} from "@tabler/icons-react";
import { Skeleton } from "@/components/ui/skeleton";
import { usePageTitle } from "@/hooks/use-page-title";

export default function AgentProfilePage() {
  const params = useParams();
  const router = useRouter();
  const agentName = params.name as string;

  const [profile, setProfile] = useState<AgentProfile | null>(null);
  const [mindTree, setMindTree] = useState<Record<string, string[]>>({});
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [activeTab, setActiveTab] = useState<"profile" | "chat" | "files">("profile");
  const [showWizard, setShowWizard] = useState(false);

  usePageTitle(agentName, "Agent");

  const fetchProfile = useCallback(() => {
    Promise.all([
      apiGet<AgentProfile>(`/api/agents/${agentName}`).catch(() => null),
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
      await apiDelete(`/api/agents/${agentName}`);
      router.push("/");
    } catch {
      showFeedback("Error: Failed to delete agent");
      setDeleting(false);
      setShowDeleteConfirm(false);
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
      {/* Header */}
      <div className="flex items-start justify-between flex-wrap gap-3">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-white/[0.04] border border-white/[0.08] flex items-center justify-center text-lg font-heading text-foreground/60">
            {agentName.charAt(0).toUpperCase()}
          </div>
          <div>
            <div className="flex items-center gap-2.5">
              <h1 className="text-2xl font-heading tracking-wide text-foreground/90">{agentName}</h1>
              <span className={`px-2 py-0.5 rounded-full text-[10px] font-mono border ${tierStyle}`}>
                {profile.tier}
              </span>
            </div>
            <p className="text-xs font-mono text-muted-foreground/40 mt-0.5">
              {profile.engine} · {profile.provider}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <button
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
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
          >
            {actionLoading === "dream" ? (
              <>
                <div className="w-3 h-3 border-2 border-purple-400/50 border-t-purple-400 rounded-full animate-spin" />
                <span className="text-purple-400/70">Dreaming...</span>
              </>
            ) : (
              <>
                <IconRefresh size={14} />
                Dream
              </>
            )}
          </button>
          <button
            onClick={async () => {
              const target = prompt("Fork target name:");
              if (!target) return;
              const ok = await callAction("fork", { target });
              if (ok) showFeedback(`Forked to "${target}"`);
            }}
            disabled={actionLoading !== null}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
          >
            {actionLoading === "fork" ? (
              <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
            ) : (
              <IconGitFork size={14} />
            )}
            Fork
          </button>
          <button
            onClick={async () => {
              const ok = await callAction("export");
              if (ok) showFeedback("Export complete!");
            }}
            disabled={actionLoading !== null}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
          >
            {actionLoading === "export" ? (
              <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
            ) : (
              <IconDownload size={14} />
            )}
            Export
          </button>
          <div className="w-px h-4 bg-white/[0.06]" />
          <button
            onClick={() => setShowDeleteConfirm(true)}
            disabled={actionLoading !== null || deleting}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-red-400/50 hover:text-red-400 hover:bg-red-500/10 transition-colors disabled:opacity-30"
          >
            <IconTrash size={14} />
            Delete
          </button>
        </div>
      </div>

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

interface ChatMessage {
  role: "user" | "agent";
  content: string;
  blocks: ActivityBlock[];
  timestamp: Date;
  error?: boolean;
  cost?: number;
  duration?: number;
}

function AgentChat({ agentName }: { agentName: string }) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const chatEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const handleSend = async () => {
    const msg = input.trim();
    if (!msg || sending) return;

    setMessages((prev) => [...prev, { role: "user", content: msg, blocks: [{ type: "text", content: msg }], timestamp: new Date() }]);
    setInput("");
    setSending(true);

    const msgIndex = messages.length + 1;
    setMessages((prev) => [...prev, { role: "agent", content: "", blocks: [], timestamp: new Date() }]);

    await streamChat({
      url: goApiUrl(`/api/agents/${agentName}/talk`),
      body: { message: msg },
      onBlocks: (newBlocks) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[msgIndex];
          if (last && last.role === "agent") {
            const allBlocks = [...last.blocks, ...newBlocks];
            const textContent = allBlocks
              .filter((b): b is { type: "text"; content: string } => b.type === "text")
              .map((b) => b.content)
              .join("");
            updated[msgIndex] = { ...last, blocks: allBlocks, content: textContent };
          }
          return updated;
        });
      },
      onDone: (meta) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[msgIndex];
          if (last && last.role === "agent") {
            updated[msgIndex] = { ...last, cost: meta.cost, duration: meta.duration };
          }
          return updated;
        });
      },
      onError: (error) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[msgIndex];
          if (last && last.role === "agent") {
            updated[msgIndex] = { ...last, content: error, blocks: [{ type: "error", content: error }], error: true };
          }
          return updated;
        });
      },
    });

    setSending(false);
    inputRef.current?.focus();
  };

  return (
    <div className="flex flex-col" style={{ height: "520px" }}>
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto glass-subtle rounded-xl mb-3">
        <div className="p-4 space-y-3 h-full">
          {messages.length === 0 && (
            <div className="flex flex-col items-center justify-center h-full text-center">
              <IconTerminal size={28} className="text-muted-foreground/15 mb-3" />
              <p className="text-sm text-muted-foreground/30">Chat with {agentName}</p>
              <p className="text-[11px] text-muted-foreground/20 mt-1">
                Send messages directly to this agent in real-time
              </p>
              <div className="flex gap-2 mt-4">
                {["What are you working on?", "Show me the project structure", "Run the tests"].map((suggestion) => (
                  <button
                    key={suggestion}
                    onClick={() => {
                      setInput(suggestion);
                      inputRef.current?.focus();
                    }}
                    className="px-3 py-1.5 rounded-lg text-[10px] font-mono text-muted-foreground/30 bg-white/[0.03] border border-white/[0.06] hover:text-muted-foreground/50 hover:bg-white/[0.05] transition-colors"
                  >
                    {suggestion}
                  </button>
                ))}
              </div>
            </div>
          )}
          {messages.map((msg, i) => (
            <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
              <div className={`max-w-[85%] rounded-xl px-3.5 py-2.5 ${
                msg.role === "user"
                  ? "bg-white/[0.08] text-foreground/80"
                  : msg.error
                    ? "bg-red-500/10 border border-red-500/15 text-red-400/80"
                    : "bg-white/[0.03] border border-white/[0.06] text-foreground/70"
              }`}>
                {msg.role === "agent" ? (
                  <ActivityMessageView message={{ role: "agent", blocks: msg.blocks, timestamp: msg.timestamp, cost: msg.cost, duration: msg.duration }} />
                ) : (
                  <p className="text-xs">{msg.content}</p>
                )}
                <p className="text-[9px] text-muted-foreground/20 mt-1">
                  {msg.role === "agent" ? agentName : "you"} · {msg.timestamp.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                </p>
              </div>
            </div>
          ))}
          {sending && (
            <div className="flex justify-start">
              <div className="bg-white/[0.03] border border-white/[0.06] rounded-xl px-3.5 py-2.5">
                <div className="flex items-center gap-2">
                  <div className="flex gap-1">
                    <div className="w-1.5 h-1.5 rounded-full bg-foreground/30 animate-bounce" style={{ animationDelay: "0ms" }} />
                    <div className="w-1.5 h-1.5 rounded-full bg-foreground/30 animate-bounce" style={{ animationDelay: "150ms" }} />
                    <div className="w-1.5 h-1.5 rounded-full bg-foreground/30 animate-bounce" style={{ animationDelay: "300ms" }} />
                  </div>
                  <span className="text-xs text-muted-foreground/40">{agentName} is thinking...</span>
                </div>
              </div>
            </div>
          )}
          <div ref={chatEndRef} />
        </div>
      </div>

      {/* Input area */}
      <div className="glass-subtle rounded-xl p-3 flex gap-2">
        <input
          ref={inputRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleSend();
            }
          }}
          placeholder={`Message ${agentName}...`}
          className="flex-1 bg-transparent text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none"
          disabled={sending}
        />
        <button
          onClick={handleSend}
          disabled={!input.trim() || sending}
          className="p-2 rounded-lg text-muted-foreground/40 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-20 disabled:cursor-not-allowed"
        >
          <IconSend size={16} />
        </button>
      </div>
    </div>
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
