"use client";

import { useState, useEffect, useRef } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiAction } from "@/lib/api-client";
import { usePageTitle } from "@/hooks/use-page-title";
import { IconSend, IconMessageCircle } from "@tabler/icons-react";

interface ArchitectStatus {
  status: "running" | "stopped";
  containerId: string | null;
  uptime: string | null;
  error?: string;
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
      apiGet<ArchitectStatus>("/api/architect/status", "/api/architect/status")
        .catch(() => ({ status: "stopped" as const, containerId: null, uptime: null }))
        .then((archStatus) => {
          setArchitectStatus(archStatus);
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
        signal: AbortSignal.timeout(60000),
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
          content: "Failed to connect to Architect. Make sure the container is running.",
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
                {isRunning ? "Online" : "Offline"}
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

      {/* Status Cards */}
      <div className="flex gap-4 flex-wrap">
        <StatCard
          label="Status"
          value={loading ? "" : isRunning ? "Online" : "Offline"}
          accent={isRunning ? "text-green-400" : "text-red-400/60"}
          loading={loading}
        />
        <StatCard
          label="Container"
          value={loading ? "" : architectStatus?.containerId ? architectStatus.containerId.slice(0, 12) : "—"}
          sub="spwn-architect"
          loading={loading}
        />
        <StatCard
          label="Uptime"
          value={loading ? "" : architectStatus?.uptime ?? "—"}
          sub={isRunning ? "since last start" : undefined}
          loading={loading}
        />
      </div>

      {/* Chat Interface */}
      <div>
        <div className="flex items-center gap-3 mb-4">
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40">Chat</h2>
          <span className="text-[9px] font-mono text-muted-foreground/20 px-2 py-0.5 rounded bg-white/[0.03] border border-white/[0.05]">
            natural language
          </span>
        </div>

        <div className="glass-subtle rounded-xl overflow-hidden flex flex-col" style={{ height: "480px" }}>
          {/* Messages area */}
          <div className="flex-1 overflow-y-auto p-4 space-y-3">
            {messages.length === 0 && (
              <div className="flex flex-col items-center justify-center h-full text-center">
                <IconMessageCircle size={28} className="text-muted-foreground/15 mb-3" />
                <p className="text-sm text-muted-foreground/30">Talk to the Architect</p>
                <p className="text-[11px] text-muted-foreground/20 mt-1 max-w-sm">
                  Ask anything in natural language. For example: &quot;Create a new agent for the API project&quot; or &quot;What agents are running?&quot;
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
                    <p className="text-xs">{msg.content}</p>
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
                    <span className="text-xs text-muted-foreground/40">Architect is thinking...</span>
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
              placeholder="Talk to the Architect..."
              className="flex-1 bg-transparent text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none"
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
    </div>
  );
}
