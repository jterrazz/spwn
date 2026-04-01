"use client";

import { useParams } from "next/navigation";
import { useState, useRef, useEffect } from "react";
import { MOCK_WORLDS } from "@/lib/mock-data";

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

// Mock mind layers
const MOCK_MIND = {
  identity: ["default.md"],
  skills: ["code-review.md", "testing.md"],
  "memory/knowledge": ["project-structure.md", "api-patterns.md"],
  "memory/playbooks": ["delegate-subtask.md"],
  "memory/journal": ["2026-03-31_w-titan.md", "2026-04-01_w-titan.md"],
  sessions: ["w-titan-84721.json"],
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

export default function AgentPage() {
  const params = useParams();
  const worldId = params.id as string;
  const agentName = params.agent as string;

  const world = MOCK_WORLDS.find((w) => w.id === worldId);
  const agent = world?.agents.find((a) => a.name === agentName);

  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [mounted, setMounted] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setMessages(INITIAL_MESSAGES);
    setMounted(true);
  }, []);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
  }, [messages]);

  const send = () => {
    if (!input.trim()) return;
    const userMsg: Message = { role: "user", content: input.trim(), timestamp: new Date() };
    setMessages((m) => [...m, userMsg]);
    setInput("");
    setIsTyping(true);

    // Mock agent response
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

  if (!world || !agent) {
    return <div className="p-8 text-muted-foreground/50">Agent not found</div>;
  }

  return (
    <div className="flex h-[calc(100vh-1px)] overflow-hidden">
      {/* ── Left: Chat ── */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Chat header */}
        <div className="px-6 py-4 border-b border-border/30 flex items-center gap-3 shrink-0">
          <div className={`w-2 h-2 rounded-full ${STATUS_DOT[agent.status]}`} />
          <div>
            <h1 className="text-base font-heading text-foreground/90">{agentName}</h1>
            <p className="text-[10px] font-mono text-muted-foreground/40">
              {TIER_LABEL[agent.tier]} · {worldId}
            </p>
          </div>
        </div>

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
              className="px-4 py-3 text-xs font-mono uppercase tracking-wider text-muted-foreground/40 hover:text-foreground/70 transition-colors disabled:opacity-30"
            >
              Send
            </button>
          </div>
          <p className="text-[9px] font-mono text-muted-foreground/20 mt-2 text-center">
            ↵ Enter to send · Connected via spwn agent talk
          </p>
        </div>
      </div>

      {/* ── Right: Agent profile panel ── */}
      <div className="w-80 border-l border-border/30 overflow-y-auto shrink-0">
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
              <div className="flex justify-between">
                <span className="text-[10px] text-muted-foreground/40">World</span>
                <span className="text-xs font-mono text-foreground/70">{worldId}</span>
              </div>
            </div>
          </div>

          {/* Stats */}
          <div>
            <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Stats</h2>
            <div className="grid grid-cols-2 gap-2">
              <div className="glass-subtle p-3 text-center">
                <p className="text-lg font-heading text-foreground/80">14</p>
                <p className="text-[9px] text-muted-foreground/35 uppercase">Sessions</p>
              </div>
              <div className="glass-subtle p-3 text-center">
                <p className="text-lg font-heading text-foreground/80">2h</p>
                <p className="text-[9px] text-muted-foreground/35 uppercase">Total time</p>
              </div>
              <div className="glass-subtle p-3 text-center">
                <p className="text-lg font-heading text-foreground/80">91%</p>
                <p className="text-[9px] text-muted-foreground/35 uppercase">Success</p>
              </div>
              <div className="glass-subtle p-3 text-center">
                <p className="text-lg font-heading text-foreground/80">3</p>
                <p className="text-[9px] text-muted-foreground/35 uppercase">Reflections</p>
              </div>
            </div>
          </div>

          {/* Mind layers */}
          <div>
            <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Mind</h2>
            <div className="space-y-1">
              {Object.entries(MOCK_MIND).map(([layer, files]) => (
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
          </div>

          {/* Commands */}
          <div>
            <h2 className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-3">Commands</h2>
            <div className="glass-subtle p-3 font-mono text-[10px] text-muted-foreground/35 space-y-1">
              <p>spwn agent talk {agentName}</p>
              <p>spwn agent inspect {agentName}</p>
              <p>spwn agent journal {agentName}</p>
              <p>spwn agent reflect {agentName}</p>
              <p>spwn agent sleep {agentName}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
