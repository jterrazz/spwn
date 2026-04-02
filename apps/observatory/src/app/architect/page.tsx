"use client";

import { useState, useEffect, useRef } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiAction } from "@/lib/api-client";
import { usePageTitle } from "@/hooks/use-page-title";
import { IconSend, IconTerminal } from "@tabler/icons-react";

interface ArchitectStatus {
  status: "running" | "stopped";
  containerId: string | null;
  uptime: string | null;

  error?: string;
}

interface StatusData {
  worlds: number;
  agents: number;
  running: number;
  limbo: number;
}

interface ChatMessage {
  role: "user" | "architect";
  content: string;
  timestamp: Date;
  error?: boolean;
}

function StatCard({ label, value, sub, accent, loading: isLoading }: { label: string; value: string; sub?: string; accent?: string; loading?: boolean }) {
  if (isLoading) {
    return (
      <div className="glass-subtle px-5 py-4 flex-1 min-w-[140px]">
        <Skeleton className="h-3 w-16 mb-2" />
        <Skeleton className="h-7 w-12 mb-1" />
        <Skeleton className="h-3 w-20" />
      </div>
    );
  }
  return (
    <div className="glass-subtle px-5 py-4 flex-1 min-w-[140px]">
      <p className="text-[10px] uppercase tracking-widest text-muted-foreground/40 mb-1">{label}</p>
      <p className={`text-2xl font-heading ${accent ?? "text-foreground/90"}`}>{value}</p>
      {sub && <p className="text-[11px] font-mono text-muted-foreground/40 mt-0.5">{sub}</p>}
    </div>
  );
}

export default function ArchitectPage() {
  usePageTitle("Architect");
  const [architectStatus, setArchitectStatus] = useState<ArchitectStatus | null>(null);
  const [statusData, setStatusData] = useState<StatusData | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);

  // Chat state
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [sending, setSending] = useState(false);
  const chatEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const fetchData = () => {
      Promise.all([
        apiGet<ArchitectStatus>("/api/architect/status", "/api/architect/status").catch(() => ({ status: "stopped" as const, containerId: null, uptime: null })),
        apiGet<StatusData>("/api/status", "/api/status").catch(() => null),
      ]).then(([archStatus, sData]) => {
        setArchitectStatus(archStatus);
        setStatusData(sData);
        setLoading(false);
      });
    };
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const showFeedback = (msg: string) => {
    setFeedback(msg);
    setTimeout(() => setFeedback(null), 3000);
  };

  const handleStart = async () => {
    setActionLoading("start");
    try {
      const result = await apiAction("/api/architect/start", undefined, "/api/architect/start");
      if (result.ok) {
        showFeedback("Architect started successfully");
        setArchitectStatus((s) => s ? { ...s, status: "running" } : s);
      } else {
        showFeedback(`Error: ${result.error}`);
      }
    } catch {
      showFeedback("Error: Failed to connect to API");
    } finally {
      setActionLoading(null);
    }
  };

  const handleStop = async () => {
    setActionLoading("stop");
    try {
      const result = await apiAction("/api/architect/stop", undefined, "/api/architect/stop");
      if (result.ok) {
        showFeedback("Architect stopped");
        setArchitectStatus((s) => s ? { ...s, status: "stopped" } : s);
      } else {
        showFeedback(`Error: ${result.error}`);
      }
    } catch {
      showFeedback("Error: Failed to connect to API");
    } finally {
      setActionLoading(null);
    }
  };

  const handleSendMessage = async () => {
    const msg = chatInput.trim();
    if (!msg || sending) return;

    const userMsg: ChatMessage = { role: "user", content: msg, timestamp: new Date() };
    setMessages((prev) => [...prev, userMsg]);
    setChatInput("");
    setSending(true);

    try {
      const res = await fetch("http://localhost:3001/api/architect/talk", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: msg }),
        signal: AbortSignal.timeout(30000),
      });
      const data = await res.json().catch(() => ({ response: "Failed to parse response" }));
      const archMsg: ChatMessage = {
        role: "architect",
        content: data.response || data.error || "No response",
        timestamp: new Date(),
        error: !!data.error && !data.response,
      };
      setMessages((prev) => [...prev, archMsg]);
    } catch {
      // Fallback to Next.js route
      try {
        const res = await fetch("/api/architect/talk", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ message: msg }),
        });
        const data = await res.json().catch(() => ({ response: "Failed to parse response" }));
        const archMsg: ChatMessage = {
          role: "architect",
          content: data.response || data.error || "No response",
          timestamp: new Date(),
          error: !!data.error && !data.response,
        };
        setMessages((prev) => [...prev, archMsg]);
      } catch {
        setMessages((prev) => [...prev, {
          role: "architect",
          content: "Failed to connect to Architect",
          timestamp: new Date(),
          error: true,
        }]);
      }
    } finally {
      setSending(false);
      inputRef.current?.focus();
    }
  };

  const isRunning = architectStatus?.status === "running";

  return (
    <div className="p-8 space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          {loading ? (
            <Skeleton className="w-3 h-3 rounded-full" />
          ) : (
            <div className={`w-3 h-3 rounded-full ${isRunning ? "bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)] animate-pulse" : "bg-white/20"}`} />
          )}
          <div>
            <h1 className="text-2xl font-heading tracking-wide text-foreground/90">Architect</h1>
            {loading ? (
              <Skeleton className="h-3 w-48 mt-1" />
            ) : (
              <p className="text-xs font-mono text-muted-foreground/40 mt-0.5">
                Orchestration daemon · {isRunning ? "running" : "stopped"}
                {architectStatus?.containerId && ` · ${architectStatus.containerId.slice(0, 12)}`}
                {architectStatus?.uptime && ` · uptime ${architectStatus.uptime}`}
              </p>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          {isRunning ? (
            <>
              <button
                onClick={handleStart}
                disabled={actionLoading !== null}
                className="glass-subtle px-4 py-2 text-[11px] font-mono uppercase tracking-wider text-muted-foreground/50 hover:text-foreground transition-colors disabled:opacity-30"
              >
                {actionLoading === "start" ? "Restarting..." : "Restart"}
              </button>
              <button
                onClick={handleStop}
                disabled={actionLoading !== null}
                className="glass-subtle px-4 py-2 text-[11px] font-mono uppercase tracking-wider text-red-400/50 hover:text-red-400 transition-colors disabled:opacity-30"
              >
                {actionLoading === "stop" ? "Stopping..." : "Stop"}
              </button>
            </>
          ) : (
            <button
              onClick={handleStart}
              disabled={actionLoading !== null}
              className="glass-subtle px-4 py-2 text-[11px] font-mono uppercase tracking-wider text-green-400/50 hover:text-green-400 transition-colors disabled:opacity-30"
            >
              {actionLoading === "start" ? "Starting..." : "Start"}
            </button>
          )}
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
      <div className="flex gap-4 flex-wrap">
        <StatCard
          label="Status"
          value={loading ? "" : isRunning ? "Online" : "Offline"}
          accent={isRunning ? "text-green-400" : "text-red-400/60"}
          loading={loading}
        />
        <StatCard
          label="Worlds"
          value={statusData ? String(statusData.worlds) : "—"}
          sub={statusData ? `${statusData.running ?? 0} running` : undefined}
          loading={loading}
        />
        <StatCard
          label="Agents"
          value={statusData ? String(statusData.agents) : "—"}
          sub="across all worlds"
          loading={loading}
        />
      </div>

      {/* Two-column layout: Chat + Config */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Chat — left 2 columns */}
        <div className="lg:col-span-2">
          <div className="flex items-center gap-3 mb-4">
            <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40">Chat</h2>
            <span className="text-[9px] font-mono text-muted-foreground/20 px-2 py-0.5 rounded bg-white/[0.03] border border-white/[0.05]">
              send spwn commands
            </span>
          </div>

          <div className="glass-subtle rounded-xl overflow-hidden flex flex-col" style={{ height: "420px" }}>
            {/* Messages area */}
            <div className="flex-1 overflow-y-auto p-4 space-y-3">
              {messages.length === 0 && (
                <div className="flex flex-col items-center justify-center h-full text-center">
                  <IconTerminal size={28} className="text-muted-foreground/15 mb-3" />
                  <p className="text-sm text-muted-foreground/30">Talk to the Architect</p>
                  <p className="text-[11px] text-muted-foreground/20 mt-1">
                    Try: <span className="font-mono">ls</span>, <span className="font-mono">agent new atlas</span>, or <span className="font-mono">status</span>
                  </p>
                </div>
              )}
              {messages.map((msg, i) => (
                <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
                  <div className={`max-w-[80%] rounded-xl px-3.5 py-2.5 ${
                    msg.role === "user"
                      ? "bg-white/[0.08] text-foreground/80"
                      : msg.error
                        ? "bg-red-500/10 border border-red-500/15 text-red-400/80"
                        : "bg-white/[0.03] border border-white/[0.06] text-foreground/70"
                  }`}>
                    {msg.role === "architect" ? (
                      <pre className="text-xs font-mono whitespace-pre-wrap break-words leading-relaxed">{msg.content}</pre>
                    ) : (
                      <p className="text-xs font-mono">{msg.content}</p>
                    )}
                    <p className="text-[9px] text-muted-foreground/20 mt-1">
                      {msg.timestamp.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                    </p>
                  </div>
                </div>
              ))}
              {sending && (
                <div className="flex justify-start">
                  <div className="bg-white/[0.03] border border-white/[0.06] rounded-xl px-3.5 py-2.5">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-foreground/30 animate-pulse" />
                      <span className="text-xs text-muted-foreground/40">Processing...</span>
                    </div>
                  </div>
                </div>
              )}
              <div ref={chatEndRef} />
            </div>

            {/* Input area */}
            <div className="border-t border-white/[0.06] p-3 flex gap-2">
              <input
                ref={inputRef}
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    handleSendMessage();
                  }
                }}
                placeholder="Type a spwn command... (e.g. ls, agent new neo)"
                className="flex-1 bg-transparent text-sm font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none"
                disabled={sending}
              />
              <button
                onClick={handleSendMessage}
                disabled={!chatInput.trim() || sending}
                className="p-2 rounded-lg text-muted-foreground/40 hover:text-foreground/70 hover:bg-white/[0.04] transition-colors disabled:opacity-20 disabled:cursor-not-allowed"
              >
                <IconSend size={16} />
              </button>
            </div>
          </div>
        </div>

        {/* Config — right column */}
        <div>
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">Config</h2>
          <div className="glass-subtle p-4 space-y-2">
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Organization</span>
              <span className="text-xs font-mono text-foreground/60">meson</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Runtime</span>
              <span className="text-xs font-mono text-foreground/60">claude-code</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Backend</span>
              <span className="text-xs font-mono text-foreground/60">docker</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Auth</span>
              <span className="text-xs font-mono text-foreground/60">subscription</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[10px] text-muted-foreground/40">Max worlds</span>
              <span className="text-xs font-mono text-foreground/60">10</span>
            </div>
          </div>

          {/* Quick Commands */}
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4 mt-8">Quick Commands</h2>
          <div className="space-y-1.5">
            {[
              { label: "List worlds", cmd: "ls" },
              { label: "List agents", cmd: "agent ls" },
              { label: "System status", cmd: "status" },
            ].map((item) => (
              <button
                key={item.cmd}
                onClick={() => {
                  setChatInput(item.cmd);
                  inputRef.current?.focus();
                }}
                className="w-full glass-subtle px-3 py-2 text-left text-[11px] font-mono text-muted-foreground/40 hover:text-foreground/60 hover:bg-white/[0.04] transition-colors"
              >
                <span className="text-muted-foreground/25">$ spwn </span>{item.cmd}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
