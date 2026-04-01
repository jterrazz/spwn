"use client";

import { useParams } from "next/navigation";
import { useState, useRef, useEffect } from "react";
import { MOCK_WORLDS, MOCK_PROFILES, MOCK_MESSAGES } from "@/lib/mock-data";
import type { AgentProfile, AgentMessage, World } from "@/lib/mock-data";
import {
  IconBrain,
  IconMessageFilled,
  IconUserFilled,
  IconRefresh,
  IconMoonFilled,
  IconGitFork,
  IconDownload,
  IconSend2,
  IconBook,
  IconNotebook,
  IconUsers,
  IconSparkles,
  IconFileText,
} from "@tabler/icons-react";

interface Message {
  role: "user" | "agent";
  content: string;
  timestamp: Date;
}

const STATUS_DOT: Record<string, string> = {
  running: "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]",
  idle: "bg-yellow-500 shadow-[0_0_6px_rgba(234,179,8,0.5)]",
  stopped: "bg-white/20",
};

const TIER_LABEL: Record<string, string> = {
  governor: "Governor",
  citizen: "Citizen",
  npc: "NPC",
};

// Mock conversation
const INITIAL_MESSAGES: Message[] = [
  {
    role: "agent",
    content: "I'm online and ready. I've read the AGENT.md and understand my role as a citizen in this world. My workspace is at /workspace and I have access to the full unix toolchain.\n\nWhat would you like me to work on?",
    timestamp: new Date(Date.now() - 1000 * 60 * 5),
  },
];

function timeStr(d: Date) {
  const h = d.getHours().toString().padStart(2, "0");
  const m = d.getMinutes().toString().padStart(2, "0");
  return `${h}:${m}`;
}

type Tab = "chat" | "profile" | "mind" | "messages";

export default function AgentPage() {
  const params = useParams();
  const worldId = params.id as string;
  const agentName = params.agent as string;

  const [world, setWorld] = useState<World | null>(null);
  const [profile, setProfile] = useState<AgentProfile | null>(null);
  const [mindTree, setMindTree] = useState<Record<string, string[]>>({});
  const [loading, setLoading] = useState(true);

  // Fetch world + agent data from API
  useEffect(() => {
    Promise.all([
      fetch("/api/worlds").then((r) => r.json()).catch(() => MOCK_WORLDS),
      fetch(`/api/agents/${agentName}`).then((r) => r.ok ? r.json() : null).catch(() => null),
      fetch(`/api/agents/${agentName}/mind`).then((r) => r.ok ? r.json() : null).catch(() => null),
    ]).then(([worlds, agentProfile, tree]) => {
      const found = (worlds as World[]).find((w) => w.id === worldId);
      setWorld(found ?? MOCK_WORLDS.find((w) => w.id === worldId) ?? null);
      setProfile(agentProfile ?? MOCK_PROFILES[agentName] ?? null);
      setMindTree(tree ?? {});
      setLoading(false);
    });
  }, [worldId, agentName]);

  const agent = world?.agents.find((a) => a.name === agentName);
  const agentMessages = MOCK_MESSAGES.filter(
    (m) => m.from === agentName || m.to === agentName
  );

  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [mounted, setMounted] = useState(false);
  const [activeTab, setActiveTab] = useState<Tab>("chat");
  const [actionFeedback, setActionFeedback] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setMessages(INITIAL_MESSAGES);
    setMounted(true);
  }, []);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
  }, [messages]);

  const showFeedback = (msg: string) => {
    setActionFeedback(msg);
    setTimeout(() => setActionFeedback(null), 2000);
  };

  const callAgentAction = async (action: string, body?: object): Promise<boolean> => {
    setActionLoading(action);
    try {
      const res = await fetch(`/api/agents/${agentName}/${action}`, {
        method: "POST",
        headers: body ? { "Content-Type": "application/json" } : undefined,
        body: body ? JSON.stringify(body) : undefined,
      });
      const data = await res.json();
      if (!res.ok) {
        showFeedback(`Error: ${data.error || "Unknown error"}`);
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

  const send = () => {
    if (!input.trim()) return;
    const userMsg: Message = { role: "user", content: input.trim(), timestamp: new Date() };
    setMessages((m) => [...m, userMsg]);
    setInput("");
    setIsTyping(true);

    setTimeout(() => {
      setMessages((m) => [
        ...m,
        {
          role: "agent",
          content: `I'll look into that. Let me check the codebase...\n\n\`\`\`bash\nls /workspace/src/\n\`\`\`\n\nI found the relevant files. Working on it now.`,
          timestamp: new Date(),
        },
      ]);
      setIsTyping(false);
    }, 1500);
  };

  if (loading) {
    return <div className="p-8 text-muted-foreground/30 animate-pulse">Loading agent...</div>;
  }

  if (!world || !agent) {
    return <div className="p-8 text-muted-foreground/50">Agent not found</div>;
  }

  const tabs: { id: Tab; label: string; icon: typeof IconMessageFilled }[] = [
    { id: "chat", label: "Chat", icon: IconMessageFilled },
    { id: "profile", label: "Profile", icon: IconUserFilled },
    { id: "mind", label: "Files", icon: IconBrain },
    { id: "messages", label: "Inbox", icon: IconMessageFilled },
  ];

  return (
    <div className="flex h-[calc(100vh-1px)] overflow-hidden">
      {/* ── Left panel ── */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Header with tabs + actions */}
        <div className="px-6 py-4 border-b border-border/30 shrink-0">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-3">
              <div className={`w-2 h-2 rounded-full ${STATUS_DOT[agent.status]}`} />
              <div>
                <h1 className="text-base font-heading text-foreground/90">{agentName}</h1>
                <p className="text-[10px] font-mono text-muted-foreground/40">
                  {TIER_LABEL[agent.tier]} · {worldId}
                </p>
              </div>
            </div>
            {/* Agent actions */}
            <div className="flex items-center gap-1">
              <button
                onClick={async () => {
                  const ok = await callAgentAction("reflect");
                  if (ok) showFeedback("Reflection complete!");
                }}
                disabled={actionLoading !== null}
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
                title="Reflect"
              >
                {actionLoading === "reflect" ? (
                  <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                ) : (
                  <IconRefresh size={14} />
                )}
                <span className="hidden sm:inline">Reflect</span>
              </button>
              <button
                onClick={async () => {
                  const ok = await callAgentAction("sleep");
                  if (ok) showFeedback("Agent is sleeping");
                }}
                disabled={actionLoading !== null}
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
                title="Sleep"
              >
                {actionLoading === "sleep" ? (
                  <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                ) : (
                  <IconMoonFilled size={14} />
                )}
                <span className="hidden sm:inline">Sleep</span>
              </button>
              <button
                onClick={async () => {
                  const target = prompt("Fork target name:");
                  if (!target) return;
                  const ok = await callAgentAction("fork", { target });
                  if (ok) showFeedback(`Forked to "${target}"`);
                }}
                disabled={actionLoading !== null}
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
                title="Fork"
              >
                {actionLoading === "fork" ? (
                  <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                ) : (
                  <IconGitFork size={14} />
                )}
                <span className="hidden sm:inline">Fork</span>
              </button>
              <button
                onClick={async () => {
                  const ok = await callAgentAction("export");
                  if (ok) showFeedback("Export complete!");
                }}
                disabled={actionLoading !== null}
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-30"
                title="Export"
              >
                {actionLoading === "export" ? (
                  <div className="w-3 h-3 border-2 border-foreground/30 border-t-foreground/70 rounded-full animate-spin" />
                ) : (
                  <IconDownload size={14} />
                )}
                <span className="hidden sm:inline">Export</span>
              </button>
            </div>
          </div>
          {/* Tab bar */}
          <div className="flex gap-1">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[11px] transition-colors ${
                    activeTab === tab.id
                      ? "bg-white/[0.08] text-foreground/80"
                      : "text-muted-foreground/40 hover:text-foreground/60 hover:bg-white/[0.03]"
                  }`}
                >
                  <Icon size={13} />
                  {tab.label}
                  {tab.id === "messages" && agentMessages.filter((m) => !m.read && m.to === agentName).length > 0 && (
                    <span className="w-1.5 h-1.5 rounded-full bg-blue-400 shadow-[0_0_4px_rgba(96,165,250,0.5)]" />
                  )}
                </button>
              );
            })}
          </div>
        </div>

        {/* Action feedback toast */}
        {actionFeedback && (
          <div className="mx-6 mt-3 px-4 py-2 rounded-lg bg-green-500/10 border border-green-500/20 text-green-400 text-xs font-mono animate-in fade-in slide-in-from-top-2 duration-200">
            {actionFeedback}
          </div>
        )}

        {/* Tab content */}
        {activeTab === "chat" && (
          <>
            {/* Messages */}
            <div ref={scrollRef} className="flex-1 overflow-y-auto px-6 py-6 space-y-5">
              {messages.map((msg, i) => (
                <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
                  <div className={`max-w-[75%] ${msg.role === "user" ? "text-right" : ""}`}>
                    <div
                      className={`inline-block px-4 py-2.5 rounded-xl text-sm leading-relaxed whitespace-pre-wrap ${
                        msg.role === "user"
                          ? "glass-subtle text-foreground/80"
                          : "text-foreground/70"
                      }`}
                    >
                      {msg.content}
                    </div>
                    <p className="text-[9px] font-mono text-muted-foreground/25 mt-1 px-1">
                      {mounted ? timeStr(msg.timestamp) : ""}
                    </p>
                  </div>
                </div>
              ))}
              {isTyping && (
                <div className="flex items-center gap-2 text-muted-foreground/40 text-xs">
                  <div className="flex gap-1">
                    <div className="w-1.5 h-1.5 rounded-full bg-muted-foreground/30 animate-bounce" style={{ animationDelay: "0ms" }} />
                    <div className="w-1.5 h-1.5 rounded-full bg-muted-foreground/30 animate-bounce" style={{ animationDelay: "150ms" }} />
                    <div className="w-1.5 h-1.5 rounded-full bg-muted-foreground/30 animate-bounce" style={{ animationDelay: "300ms" }} />
                  </div>
                  <span>{agentName} is thinking...</span>
                </div>
              )}
            </div>

            {/* Input */}
            <div className="px-6 py-4 border-t border-border/30 shrink-0">
              <div className="glass-subtle flex items-center rounded-lg">
                <input
                  ref={inputRef}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && send()}
                  placeholder={`Talk to ${agentName}...`}
                  className="flex-1 bg-transparent px-4 py-3 text-sm text-foreground/80 placeholder:text-muted-foreground/30 focus:outline-none"
                />
                <button
                  onClick={send}
                  disabled={!input.trim()}
                  className="px-4 py-3 text-muted-foreground/40 hover:text-foreground/70 transition-colors disabled:opacity-30"
                >
                  <IconSend2 size={16} />
                </button>
              </div>
              <p className="text-[9px] font-mono text-muted-foreground/20 mt-2 text-center">
                ↵ Enter to send · Connected via spwn agent talk
              </p>
            </div>
          </>
        )}

        {activeTab === "profile" && profile && (
          <div className="flex-1 overflow-y-auto px-6 py-6 space-y-6">
            <ProfileView profile={profile} />
          </div>
        )}

        {activeTab === "profile" && !profile && (
          <div className="flex-1 flex items-center justify-center text-muted-foreground/30 text-sm">
            No profile data available
          </div>
        )}

        {activeTab === "mind" && (
          <div className="flex-1 overflow-y-auto px-6 py-6">
            <MindView mindTree={mindTree} />
          </div>
        )}

        {activeTab === "messages" && (
          <div className="flex-1 overflow-y-auto px-6 py-6">
            <MessagesView agentName={agentName} messages={agentMessages} />
          </div>
        )}
      </div>

      {/* ── Right: Quick info panel ── */}
      <div className="w-72 border-l border-border/30 overflow-y-auto shrink-0 hidden lg:block">
        <div className="p-5 space-y-6">
          {/* Identity */}
          <div>
            <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Identity</h2>
            <div className="glass-subtle p-4 space-y-2">
              <div className="flex justify-between">
                <span className="text-[10px] text-muted-foreground/40">Name</span>
                <span className="text-xs font-mono text-foreground/70">{agentName}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-[10px] text-muted-foreground/40">Tier</span>
                <span className="text-xs font-mono text-foreground/70 capitalize">{agent.tier}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-[10px] text-muted-foreground/40">Status</span>
                <div className="flex items-center gap-1.5">
                  <div className={`w-1.5 h-1.5 rounded-full ${STATUS_DOT[agent.status]}`} />
                  <span className="text-xs font-mono text-foreground/70 capitalize">{agent.status}</span>
                </div>
              </div>
              {profile && (
                <>
                  <div className="flex justify-between">
                    <span className="text-[10px] text-muted-foreground/40">Engine</span>
                    <span className="text-xs font-mono text-foreground/70">{profile.engine}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-[10px] text-muted-foreground/40">Provider</span>
                    <span className="text-xs font-mono text-foreground/70">{profile.provider}</span>
                  </div>
                </>
              )}
            </div>
          </div>

          {/* Stats */}
          <div>
            <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Stats</h2>
            <div className="grid grid-cols-2 gap-2">
              <div className="glass-subtle p-3 text-center">
                <p className="text-lg font-heading text-foreground/80">{Object.values(mindTree).reduce((n, f) => n + f.length, 0) || 14}</p>
                <p className="text-[9px] text-muted-foreground/35 uppercase">Files</p>
              </div>
              <div className="glass-subtle p-3 text-center">
                <p className="text-lg font-heading text-foreground/80">{Object.keys(mindTree).filter(k => (mindTree[k]?.length ?? 0) > 0).length || 6}</p>
                <p className="text-[9px] text-muted-foreground/35 uppercase">Layers</p>
              </div>
              <div className="glass-subtle p-3 text-center">
                <p className="text-lg font-heading text-foreground/80">{profile?.journal.length ?? 0}</p>
                <p className="text-[9px] text-muted-foreground/35 uppercase">Journal</p>
              </div>
              <div className="glass-subtle p-3 text-center">
                <p className="text-lg font-heading text-foreground/80">{profile?.bonds.length ?? 0}</p>
                <p className="text-[9px] text-muted-foreground/35 uppercase">Bonds</p>
              </div>
            </div>
          </div>

          {/* Quick commands */}
          <div>
            <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Commands</h2>
            <div className="glass-subtle p-3 font-mono text-[10px] text-muted-foreground/35 space-y-1">
              <p>spwn agent talk {agentName}</p>
              <p>spwn agent inspect {agentName}</p>
              <p>spwn agent reflect {agentName}</p>
              <p>spwn agent sleep {agentName}</p>
              <p>spwn agent fork {agentName}</p>
              <p>spwn agent export {agentName}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ── Profile View Component ── */
function ProfileView({ profile }: { profile: AgentProfile }) {
  return (
    <div className="space-y-6 max-w-2xl">
      {/* Purpose & Persona */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
          <IconSparkles size={12} />
          Purpose
        </h2>
        <div className="glass-subtle p-4">
          <p className="text-sm text-foreground/70 leading-relaxed">{profile.purpose}</p>
        </div>
      </div>

      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Persona</h2>
        <div className="glass-subtle p-4">
          <p className="text-sm text-foreground/60 leading-relaxed italic">{profile.persona}</p>
        </div>
      </div>

      {/* Traits */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Traits</h2>
        <div className="flex flex-wrap gap-2">
          {profile.traits.map((trait) => (
            <span
              key={trait}
              className="px-2.5 py-1 rounded-full text-[11px] font-mono bg-purple-500/10 text-purple-300/80 border border-purple-500/20"
            >
              {trait}
            </span>
          ))}
        </div>
      </div>

      {/* Skills */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
          <IconBrain size={12} />
          Skills
        </h2>
        <div className="flex flex-wrap gap-2">
          {profile.skills.map((skill) => (
            <span
              key={skill}
              className="px-2.5 py-1 rounded-full text-[11px] font-mono bg-blue-500/10 text-blue-300/80 border border-blue-500/20"
            >
              {skill}
            </span>
          ))}
        </div>
      </div>

      {/* Playbooks */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
          <IconBook size={12} />
          Playbooks
        </h2>
        <div className="glass-subtle divide-y divide-border/20">
          {profile.playbooks.map((pb) => (
            <div key={pb} className="px-4 py-2.5 flex items-center gap-2">
              <IconFileText size={13} className="text-muted-foreground/30 shrink-0" />
              <span className="text-xs font-mono text-foreground/60">{pb}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Knowledge */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Knowledge Files</h2>
        <div className="glass-subtle divide-y divide-border/20">
          {profile.knowledge.map((k) => (
            <div key={k} className="px-4 py-2.5 flex items-center gap-2">
              <IconFileText size={13} className="text-muted-foreground/30 shrink-0" />
              <span className="text-xs font-mono text-foreground/60">{k}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Journal */}
      <div>
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
          <IconNotebook size={12} />
          Journal
        </h2>
        <div className="space-y-2">
          {profile.journal.map((entry) => (
            <div key={entry.date} className="glass-subtle p-4">
              <p className="text-[10px] font-mono text-muted-foreground/40 mb-1.5">{entry.date}</p>
              <p className="text-xs text-foreground/60 leading-relaxed">{entry.summary}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Bonds */}
      {profile.bonds.length > 0 && (
        <div>
          <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3 flex items-center gap-1.5">
            <IconUsers size={12} />
            Bonds
          </h2>
          <div className="glass-subtle divide-y divide-border/20">
            {profile.bonds.map((bond) => (
              <div key={bond.agent} className="px-4 py-3 flex items-center justify-between">
                <span className="text-xs font-mono text-foreground/70">{bond.agent}</span>
                <span className="text-[10px] text-muted-foreground/40 italic">{bond.relationship}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

/* ── Files View Component ── */
function MindView({ mindTree }: { mindTree: Record<string, string[]> }) {
  // Fallback to mock mind data if no real data
  const data = Object.keys(mindTree).length > 0
    ? mindTree
    : {
        identity: ["persona.md"],
        skills: ["code-review.md", "testing.md"],
        "memory/knowledge": ["project-structure.md", "api-patterns.md"],
        "memory/playbooks": ["delegate-subtask.md"],
        "memory/journal": ["2026-03-31_w-titan.md", "2026-04-01_w-titan.md"],
        sessions: ["w-titan-84721.json"],
      };

  return (
    <div className="space-y-1 max-w-lg">
      <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Profile Layers</h2>
      {Object.entries(data).map(([layer, files]) => (
        <div key={layer} className="glass-subtle px-3 py-2">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-mono text-foreground/60">{layer}/</span>
            <span className="text-[9px] font-mono text-muted-foreground/30">
              {files.length} file{files.length !== 1 ? "s" : ""}
            </span>
          </div>
          <div className="mt-1 space-y-0.5">
            {files.map((f) => (
              <p key={f} className="text-[10px] font-mono text-muted-foreground/40 pl-3">
                {f}
              </p>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

/* ── Messages View Component ── */
function MessagesView({ agentName, messages }: { agentName: string; messages: AgentMessage[] }) {
  const [composeTo, setComposeTo] = useState("");
  const [composeMsg, setComposeMsg] = useState("");
  const [showCompose, setShowCompose] = useState(false);

  return (
    <div className="space-y-4 max-w-2xl">
      <div className="flex items-center justify-between">
        <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40">
          Messages ({messages.length})
        </h2>
        <button
          onClick={() => setShowCompose(!showCompose)}
          className="text-[11px] px-2.5 py-1 rounded-lg text-muted-foreground/50 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors"
        >
          + New Message
        </button>
      </div>

      {/* Compose */}
      {showCompose && (
        <div className="glass-subtle p-4 space-y-3">
          <input
            value={composeTo}
            onChange={(e) => setComposeTo(e.target.value)}
            placeholder="To agent..."
            className="w-full bg-transparent text-sm text-foreground/70 placeholder:text-muted-foreground/30 border-b border-border/20 pb-2 focus:outline-none"
          />
          <textarea
            value={composeMsg}
            onChange={(e) => setComposeMsg(e.target.value)}
            placeholder="Write a message..."
            rows={3}
            className="w-full bg-transparent text-sm text-foreground/70 placeholder:text-muted-foreground/30 focus:outline-none resize-none"
          />
          <div className="flex justify-end gap-2">
            <button
              onClick={() => setShowCompose(false)}
              className="px-3 py-1.5 rounded-lg text-[11px] text-muted-foreground/40 hover:text-foreground/60 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => {
                setShowCompose(false);
                setComposeTo("");
                setComposeMsg("");
              }}
              className="px-3 py-1.5 rounded-lg text-[11px] bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] transition-colors"
            >
              Send
            </button>
          </div>
        </div>
      )}

      {/* Message list */}
      <div className="space-y-2">
        {messages.length === 0 ? (
          <p className="text-sm text-muted-foreground/30 text-center py-8">No messages</p>
        ) : (
          messages.map((msg) => {
            const isIncoming = msg.to === agentName;
            return (
              <div key={msg.id} className="glass-subtle p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] text-muted-foreground/40">
                      {isIncoming ? "from" : "to"}
                    </span>
                    <span className="text-xs font-mono text-foreground/70">
                      {isIncoming ? msg.from : msg.to}
                    </span>
                    {!msg.read && isIncoming && (
                      <span className="w-1.5 h-1.5 rounded-full bg-blue-400 shadow-[0_0_4px_rgba(96,165,250,0.5)]" />
                    )}
                  </div>
                  <span className="text-[9px] font-mono text-muted-foreground/30">
                    {new Date(msg.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                  </span>
                </div>
                <p className="text-xs text-foreground/60 leading-relaxed">{msg.content}</p>
                <p className="text-[9px] font-mono text-muted-foreground/25 mt-2">#{msg.channel}</p>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
