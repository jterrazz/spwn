"use client";

import { useParams } from "next/navigation";
import { useState, useEffect } from "react";
import type { AgentProfile } from "@/lib/types";
import { apiGet, apiAction } from "@/lib/api-client";
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
} from "@tabler/icons-react";
import { Skeleton } from "@/components/ui/skeleton";

export default function AgentProfilePage() {
  const params = useParams();
  const agentName = params.name as string;

  const [profile, setProfile] = useState<AgentProfile | null>(null);
  const [mindTree, setMindTree] = useState<Record<string, string[]>>({});
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      apiGet<AgentProfile>(`/api/agents/${agentName}`, `/api/agents/${agentName}`).catch(() => null),
      apiGet<Record<string, string[]>>(`/api/agents/${agentName}/mind`, `/api/agents/${agentName}/mind`).catch(() => null),
    ]).then(([agentProfile, tree]) => {
      setProfile(agentProfile ?? null);
      setMindTree(tree ?? {});
      setLoading(false);
    });
  }, [agentName]);

  const showFeedback = (msg: string) => {
    setFeedback(msg);
    setTimeout(() => setFeedback(null), 2500);
  };

  const callAction = async (action: string, body?: object): Promise<boolean> => {
    setActionLoading(action);
    try {
      const result = await apiAction(
        `/api/agents/${agentName}/${action}`,
        body,
        `/api/agents/${agentName}/${action}`
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
      <div className="p-8">
        <p className="text-muted-foreground/50">Agent &quot;{agentName}&quot; not found</p>
        <p className="text-xs text-muted-foreground/30 mt-2 font-mono">
          Create this agent with: spwn agent create {agentName}
        </p>
      </div>
    );
  }

  const totalFiles = Object.values(mindTree).reduce((n, f) => n + f.length, 0);
  const activeLayers = Object.keys(mindTree).filter((k) => (mindTree[k]?.length ?? 0) > 0).length;

  return (
    <div className="p-8 space-y-8 max-w-3xl">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-white/[0.04] border border-white/[0.08] flex items-center justify-center text-lg font-heading text-foreground/60">
            {agentName.charAt(0).toUpperCase()}
          </div>
          <div>
            <h1 className="text-2xl font-heading tracking-wide text-foreground/90">{agentName}</h1>
            <p className="text-xs font-mono text-muted-foreground/40 mt-0.5">
              {profile.tier} · {profile.engine} · {profile.provider}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={async () => {
              const ok = await callAction("dream");
              if (ok) showFeedback("Dream cycle complete!");
            }}
            disabled={actionLoading !== null}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
          >
            {actionLoading === "dream" ? (
              <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
            ) : (
              <IconRefresh size={14} />
            )}
            Dream
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
      <div className="grid grid-cols-4 gap-3">
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

      {/* Purpose & Persona */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
          <IconSparkles size={12} />
          Purpose
        </h2>
        <div className="glass-subtle p-4">
          <p className="text-sm text-foreground/70 leading-relaxed">{profile.purpose || "Not configured yet"}</p>
        </div>
      </div>

      {profile.persona && (
        <div>
          <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Persona</h2>
          <div className="glass-subtle p-4">
            <p className="text-sm text-foreground/60 leading-relaxed italic">{profile.persona}</p>
          </div>
        </div>
      )}

      {/* Traits */}
      {(profile.traits?.length ?? 0) > 0 && (
        <div>
          <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Traits</h2>
          <div className="flex flex-wrap gap-2">
            {(profile.traits ?? []).map((trait) => (
              <span
                key={trait}
                className="px-2.5 py-1 rounded-full text-[11px] font-mono bg-purple-500/10 text-purple-300/80 border border-purple-500/20"
              >
                {trait}
              </span>
            ))}
          </div>
        </div>
      )}

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
    </div>
  );
}
