"use client";

import { useState, useEffect, useRef } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet } from "@/lib/api-client";
import { streamChat } from "@/lib/stream-chat";
import type { ActivityBlock } from "@/lib/activity-types";
import { ActivityMessageView } from "@/components/activity-blocks";
import { usePageTitle } from "@/hooks/use-page-title";
import {
  IconMessageCircle,
  IconWorld,
  IconUsers,
  IconClock,
  IconCircleCheck,
  IconChevronDown,
  IconChevronRight,
  IconClipboardList,
  IconArrowUp,
  IconBook2,
} from "@tabler/icons-react";
import { BlueprintBrowser, BlueprintUpdateCard } from "@/components/blueprint-browser";

interface ArchitectStatus {
  status: "running" | "stopped";
  containerId: string | null;
  uptime: string | null;
  error?: string;
  kpis?: {
    worlds: number;
    agents: number;
    tasksPending: number;
    tasksCompleted: number;
  };
}

interface DirectiveActionData {
  type: "add" | "done" | "update";
  title: string;
  priority?: string;
  description?: string;
}

interface BlueprintUpdateData {
  path: string;
  description?: string;
}

interface ChatMessage {
  role: "user" | "architect";
  content: string;
  blocks: ActivityBlock[];
  timestamp: Date;
  error?: boolean;
  cost?: number;
  duration?: number;
  directiveAction?: DirectiveActionData;
  blueprintUpdate?: BlueprintUpdateData;
}

interface Directive {
  text: string;
  done: boolean;
  priority?: "high" | "medium" | "low";
  description?: string;
}

interface DirectivesData {
  inProgress: Directive[];
  backlog: Directive[];
  completed: Directive[];
  raw: string;
}

function StatCard({ label, value, sub, accent, icon, loading: isLoading }: { label: string; value: string; sub?: string; accent?: string; icon?: React.ReactNode; loading?: boolean }) {
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
      <div className="flex items-center gap-2 mb-1">
        {icon && <span className="text-muted-foreground/30">{icon}</span>}
        <p className="text-[10px] uppercase tracking-widest text-muted-foreground/40">{label}</p>
      </div>
      <p className={`text-2xl font-heading ${accent ?? "text-foreground/90"}`}>{value}</p>
      {sub && <p className="text-[11px] font-mono text-muted-foreground/40 mt-0.5">{sub}</p>}
    </div>
  );
}

function PriorityBadge({ priority }: { priority?: "high" | "medium" | "low" }) {
  if (!priority) return null;
  const colors = {
    high: "bg-red-500/15 text-red-400/90 border-red-500/20",
    medium: "bg-amber-500/15 text-amber-400/90 border-amber-500/20",
    low: "bg-green-500/15 text-green-400/90 border-green-500/20",
  };
  return (
    <span className={`text-[9px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded-full border ${colors[priority]}`}>
      {priority}
    </span>
  );
}

function StatusDot({ status }: { status: "inProgress" | "backlog" | "done" }) {
  if (status === "done") {
    return <IconCircleCheck size={16} className="text-green-400/70 flex-shrink-0" />;
  }
  if (status === "inProgress") {
    return (
      <span className="relative flex h-3 w-3 flex-shrink-0">
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-yellow-400/40" />
        <span className="relative inline-flex rounded-full h-3 w-3 bg-yellow-400/70" />
      </span>
    );
  }
  return <span className="w-3 h-3 rounded-full bg-white/15 flex-shrink-0" />;
}

function DirectiveCard({ item, status, isHighlighted }: { item: Directive; status: "inProgress" | "backlog" | "done"; isHighlighted: boolean }) {
  return (
    <div
      className={`group rounded-lg border px-3 py-2.5 transition-all duration-500 ${
        isHighlighted
          ? "animate-in slide-in-from-right-4 fade-in duration-500 bg-blue-500/8 border-blue-500/20 shadow-[0_0_12px_rgba(59,130,246,0.08)]"
          : status === "done"
            ? "bg-white/[0.01] border-white/[0.04] opacity-60"
            : "bg-white/[0.03] border-white/[0.07] hover:border-white/[0.12] hover:bg-white/[0.05]"
      }`}
    >
      <div className="flex items-start gap-2.5">
        <div className="mt-0.5">
          <StatusDot status={status} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className={`text-xs font-medium leading-tight ${status === "done" ? "line-through text-muted-foreground/30" : "text-foreground/80"}`}>
              {item.text}
            </span>
            <PriorityBadge priority={item.priority} />
          </div>
          {item.description && (
            <p className="text-[11px] text-muted-foreground/35 mt-1 leading-relaxed">{item.description}</p>
          )}
        </div>
      </div>
    </div>
  );
}

function DirectivesPanel({ directives, highlightTitle }: { directives: DirectivesData | null; highlightTitle: string | null }) {
  const [showCompleted, setShowCompleted] = useState(false);

  if (!directives) {
    return (
      <div className="glass-subtle rounded-xl p-5 space-y-3">
        <div className="flex items-center gap-2.5">
          <IconClipboardList size={18} className="text-muted-foreground/30" />
          <h3 className="text-sm font-heading tracking-wide text-muted-foreground/50">Directives</h3>
        </div>
        <div className="space-y-2">
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-3/4 rounded-lg" />
        </div>
      </div>
    );
  }

  const isEmpty = directives.inProgress.length === 0 && directives.backlog.length === 0 && directives.completed.length === 0;
  const totalActive = directives.inProgress.length + directives.backlog.length;

  return (
    <div className="glass-subtle rounded-xl p-5 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <IconClipboardList size={18} className="text-muted-foreground/40" />
          <h3 className="text-sm font-heading tracking-wide text-foreground/60">Directives</h3>
        </div>
        <span className="text-[9px] font-mono text-muted-foreground/25 px-2 py-0.5 rounded-full bg-white/[0.03] border border-white/[0.05]">
          {totalActive > 0 ? `${totalActive} active` : "managed by architect"}
        </span>
      </div>

      {/* Empty state */}
      {isEmpty && (
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <div className="w-10 h-10 rounded-full bg-white/[0.04] flex items-center justify-center mb-3">
            <IconClipboardList size={20} className="text-muted-foreground/20" />
          </div>
          <p className="text-xs text-muted-foreground/35 font-medium">No directives yet</p>
          <p className="text-[11px] text-muted-foreground/20 mt-1">Talk to the Architect to get started</p>
        </div>
      )}

      {/* In Progress */}
      {directives.inProgress.length > 0 && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="w-1.5 h-1.5 rounded-full bg-yellow-400/60" />
            <p className="text-[10px] uppercase tracking-wider font-medium text-yellow-400/60">In Progress</p>
            <span className="text-[9px] font-mono text-yellow-400/30 ml-auto">{directives.inProgress.length}</span>
          </div>
          <div className="space-y-1.5">
            {directives.inProgress.map((item, i) => (
              <DirectiveCard
                key={`ip-${i}`}
                item={item}
                status="inProgress"
                isHighlighted={!!highlightTitle && item.text.includes(highlightTitle)}
              />
            ))}
          </div>
        </div>
      )}

      {/* Backlog */}
      {directives.backlog.length > 0 && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="w-1.5 h-1.5 rounded-full bg-white/20" />
            <p className="text-[10px] uppercase tracking-wider font-medium text-muted-foreground/35">Backlog</p>
            <span className="text-[9px] font-mono text-muted-foreground/20 ml-auto">{directives.backlog.length}</span>
          </div>
          <div className="space-y-1.5">
            {directives.backlog.map((item, i) => (
              <DirectiveCard
                key={`bl-${i}`}
                item={item}
                status="backlog"
                isHighlighted={!!highlightTitle && item.text.includes(highlightTitle)}
              />
            ))}
          </div>
        </div>
      )}

      {/* Completed (collapsible) */}
      {directives.completed.length > 0 && (
        <div className="pt-1 border-t border-white/[0.05]">
          <button
            onClick={() => setShowCompleted(!showCompleted)}
            className="flex items-center gap-2 w-full text-[10px] uppercase tracking-wider text-muted-foreground/25 hover:text-muted-foreground/40 transition-colors py-1"
          >
            {showCompleted ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
            <span>Completed</span>
            <span className="ml-auto text-[9px] font-mono px-1.5 py-0.5 rounded-full bg-green-500/10 text-green-400/40 border border-green-500/10">
              {directives.completed.length}
            </span>
          </button>
          {showCompleted && (
            <div className="space-y-1.5 mt-2">
              {directives.completed.map((item, i) => (
                <DirectiveCard
                  key={`done-${i}`}
                  item={item}
                  status="done"
                  isHighlighted={false}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function extractPriority(text: string): { cleanText: string; priority?: "high" | "medium" | "low"; description?: string } {
  let cleanText = text;
  let priority: "high" | "medium" | "low" | undefined;
  let description: string | undefined;

  // Extract priority markers like [HIGH], [MEDIUM], [LOW], (high), (medium), (low), or **HIGH**
  const priorityMatch = cleanText.match(/\s*[\[(]*\*{0,2}(HIGH|MEDIUM|LOW)\*{0,2}[\])]*\s*/i);
  if (priorityMatch) {
    priority = priorityMatch[1].toLowerCase() as "high" | "medium" | "low";
    cleanText = cleanText.replace(priorityMatch[0], " ").trim();
  }

  // Extract description after " - " or " — " separator
  const descSep = cleanText.match(/\s+[-—]\s+(.+)$/);
  if (descSep) {
    description = descSep[1];
    cleanText = cleanText.slice(0, cleanText.length - descSep[0].length).trim();
  }

  return { cleanText, priority, description };
}

function parseDirectivesMd(raw: string): DirectivesData {
  const lines = raw.split("\n");
  const inProgress: Directive[] = [];
  const backlog: Directive[] = [];
  const completed: Directive[] = [];

  let section = "backlog";

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.toLowerCase().startsWith("## in progress") || trimmed.toLowerCase().startsWith("## in-progress")) {
      section = "inProgress";
      continue;
    }
    if (trimmed.toLowerCase().startsWith("## backlog")) {
      section = "backlog";
      continue;
    }
    if (trimmed.toLowerCase().startsWith("## completed") || trimmed.toLowerCase().startsWith("## done")) {
      section = "completed";
      continue;
    }
    if (trimmed.startsWith("#")) continue;

    // Parse checkbox items
    const checkMatch = trimmed.match(/^-\s*\[([ xX])\]\s*(.+)/);
    if (checkMatch) {
      const done = checkMatch[1] !== " ";
      const { cleanText, priority, description } = extractPriority(checkMatch[2]);
      const item: Directive = { text: cleanText, done, priority, description };
      if (done) {
        completed.push(item);
      } else if (section === "inProgress") {
        inProgress.push(item);
      } else {
        backlog.push(item);
      }
      continue;
    }

    // Parse plain list items
    const listMatch = trimmed.match(/^-\s+(.+)/);
    if (listMatch) {
      const { cleanText, priority, description } = extractPriority(listMatch[1]);
      const item: Directive = { text: cleanText, done: section === "completed", priority, description };
      if (section === "inProgress") {
        inProgress.push(item);
      } else if (section === "completed") {
        completed.push(item);
      } else {
        backlog.push(item);
      }
    }
  }

  return { inProgress, backlog, completed, raw };
}

export default function ArchitectPage() {
  usePageTitle("Architect");
  const [architectStatus, setArchitectStatus] = useState<ArchitectStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [directives, setDirectives] = useState<DirectivesData | null>(null);

  // Tab state
  const [activeTab, setActiveTab] = useState<"chat" | "blueprint" | "tasks">("chat");

  // Chat state
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [sending, setSending] = useState(false);
  const [highlightTitle, setHighlightTitle] = useState<string | null>(null);
  const chatEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const fetchDirectives = () => {
    apiGet<{ content: string }>("/api/architect/directives")
      .then((data) => {
        setDirectives(parseDirectivesMd(data.content));
      })
      .catch(() => {
        // No directives available
      });
  };

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
    fetchDirectives();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (messages.length > 0) {
      chatEndRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  }, [messages]);

  const showFeedback = (msg: string) => {
    setFeedback(msg);
    setTimeout(() => setFeedback(null), 3000);
  };

  const handleStart = async () => {
    setActionLoading("start");
    try {
      const res = await fetch("http://localhost:3001/api/architect/start", { method: "POST" });
      if (res.ok) {
        showFeedback("Architect started successfully");
        setArchitectStatus((s) => s ? { ...s, status: "running" } : s);
      } else {
        const data = await res.json().catch(() => ({ error: "Unknown error" }));
        showFeedback(`Error: ${data.error}`);
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
      const res = await fetch("http://localhost:3001/api/architect/stop", { method: "POST" });
      if (res.ok) {
        showFeedback("Architect stopped");
        setArchitectStatus((s) => s ? { ...s, status: "stopped" } : s);
      } else {
        const data = await res.json().catch(() => ({ error: "Unknown error" }));
        showFeedback(`Error: ${data.error}`);
      }
    } catch {
      showFeedback("Error: Failed to connect to API");
    } finally {
      setActionLoading(null);
    }
  };

  const handleTalkResponse = (data: { response?: string; error?: string; directiveAction?: DirectiveActionData; blueprintUpdate?: BlueprintUpdateData }) => {
    const text = data.response || data.error || "No response";
    const archMsg: ChatMessage = {
      role: "architect",
      content: text,
      blocks: [{ type: data.error && !data.response ? "error" as const : "text" as const, content: text }],
      timestamp: new Date(),
      error: !!data.error && !data.response,
      directiveAction: data.directiveAction,
      blueprintUpdate: data.blueprintUpdate,
    };
    setMessages((prev) => [...prev, archMsg]);

    // If there's a directive action, refresh directives and highlight the item
    if (data.directiveAction) {
      fetchDirectives();
      setHighlightTitle(data.directiveAction.title);
      setTimeout(() => setHighlightTitle(null), 3000);
    }
  };

  const doTalk = async (msg: string) => {
    const msgIndex = messages.length + 1;
    setMessages((prev) => [...prev, {
      role: "architect" as const, content: "", blocks: [], timestamp: new Date(),
    }]);

    await streamChat({
      url: "http://localhost:3001/api/architect/talk",
      fallbackUrl: "/api/architect/talk",
      body: { message: msg },
      onBlocks: (newBlocks) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[msgIndex];
          if (last && last.role === "architect") {
            const allBlocks = [...last.blocks, ...newBlocks];
            const textContent = allBlocks
              .filter((b) => b.type === "text")
              .map((b) => (b as { content: string }).content)
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
          if (last && last.role === "architect") {
            updated[msgIndex] = { ...last, cost: meta.cost, duration: meta.duration };
          }
          return updated;
        });
      },
      onError: (error) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[msgIndex];
          if (last && last.role === "architect") {
            updated[msgIndex] = {
              ...last,
              blocks: [...last.blocks, { type: "error" as const, content: error }],
              content: error,
              error: true,
            };
          }
          return updated;
        });
      },
    });
  };

  const handleSendMessage = async () => {
    const msg = chatInput.trim();
    if (!msg || sending) return;

    const userMsg: ChatMessage = { role: "user", content: msg, blocks: [{ type: "text", content: msg }], timestamp: new Date() };
    setMessages((prev) => [...prev, userMsg]);
    setChatInput("");
    setSending(true);

    try {
      // Check if architect is running
      let running = architectStatus?.status === "running";

      if (!running) {
        // Show starting message
        setMessages((prev) => [...prev, {
          role: "architect",
          content: "Starting Architect...",
          blocks: [{ type: "text" as const, content: "Starting Architect..." }],
          timestamp: new Date(),
        }]);

        // Start it (try Go API, fallback to Next.js)
        try {
          await fetch("http://localhost:3001/api/architect/start", { method: "POST" });
        } catch {
          try {
            await fetch("/api/architect/start", { method: "POST" });
          } catch {
            setMessages((prev) => [...prev, {
              role: "architect",
              content: "Failed to auto-start Architect. Please start it manually.",
              blocks: [{ type: "error" as const, content: "Failed to auto-start Architect. Please start it manually." }],
              timestamp: new Date(),
              error: true,
            }]);
            setSending(false);
            inputRef.current?.focus();
            return;
          }
        }

        // Wait for it to be ready (poll every 2s, max 30s)
        for (let i = 0; i < 15; i++) {
          await new Promise((resolve) => setTimeout(resolve, 2000));
          try {
            const statusRes = await fetch("http://localhost:3001/api/architect/status");
            const statusData = await statusRes.json();
            if (statusData.status === "running") {
              running = true;
              setArchitectStatus((s) => s ? { ...s, status: "running" } : { status: "running", containerId: null, uptime: null });
              break;
            }
          } catch {
            // Ignore polling errors, keep trying
          }
        }

        if (!running) {
          setMessages((prev) => [...prev, {
            role: "architect",
            content: "Architect failed to start after 30s. Please try starting it manually.",
            blocks: [{ type: "error" as const, content: "Architect failed to start after 30s." }],
            timestamp: new Date(),
            error: true,
          }]);
          setSending(false);
          inputRef.current?.focus();
          return;
        }
      }

      // Now talk
      await doTalk(msg);
    } catch (e: unknown) {
      const errMsg = e instanceof Error ? e.message : "Unknown error";
      setMessages((prev) => [...prev, {
        role: "architect",
        content: `Error: ${errMsg}`,
        blocks: [{ type: "error" as const, content: errMsg }],
        timestamp: new Date(),
        error: true,
      }]);
    } finally {
      setSending(false);
      inputRef.current?.focus();
    }
  };

  const isRunning = architectStatus?.status === "running";
  const kpis = architectStatus?.kpis;

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

      {/* KPI Cards */}
      <div className="flex gap-4 flex-wrap">
        <StatCard
          label="Worlds"
          value={loading ? "" : String(kpis?.worlds ?? 0)}
          sub="running"
          accent="text-blue-400"
          icon={<IconWorld size={14} />}
          loading={loading}
        />
        <StatCard
          label="Agents"
          value={loading ? "" : String(kpis?.agents ?? 0)}
          sub="managed"
          accent="text-purple-400"
          icon={<IconUsers size={14} />}
          loading={loading}
        />
        <StatCard
          label="Directives"
          value={loading ? "" : String(kpis?.tasksPending ?? 0)}
          sub="pending"
          accent="text-yellow-400"
          icon={<IconClock size={14} />}
          loading={loading}
        />
        <StatCard
          label="Done"
          value={loading ? "" : String(kpis?.tasksCompleted ?? 0)}
          sub="completed"
          accent="text-green-400"
          icon={<IconCircleCheck size={14} />}
          loading={loading}
        />
      </div>

      {/* Tab bar */}
      <div className="flex items-center gap-1 border-b border-white/[0.06] pb-0">
        {([
          { key: "chat", label: "Chat", icon: <IconMessageCircle size={14} /> },
          { key: "blueprint", label: "Blueprint", icon: <IconBook2 size={14} /> },
          { key: "tasks", label: "Directives", icon: <IconClipboardList size={14} /> },
        ] as const).map(({ key, label, icon }) => (
          <button
            key={key}
            onClick={() => setActiveTab(key)}
            className={`flex items-center gap-2 px-4 py-2.5 text-xs font-medium transition-colors border-b-2 -mb-[1px] ${
              activeTab === key
                ? "text-foreground/80 border-foreground/50"
                : "text-muted-foreground/40 border-transparent hover:text-muted-foreground/60"
            }`}
          >
            <span className="opacity-60">{icon}</span>
            {label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {activeTab === "chat" && (
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Chat Interface (2/3 width) */}
        <div className="lg:col-span-2">

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
                  {!isRunning && (
                    <p className="text-[10px] text-yellow-400/40 mt-2 font-mono">
                      Architect is offline — it will auto-start when you send a message
                    </p>
                  )}
                </div>
              )}
              {messages.map((msg, i) => (
                <div key={i} className={`flex flex-col ${msg.role === "user" ? "items-end" : "items-start"}`}>
                  <div className={`max-w-[80%] rounded-xl px-3.5 py-2.5 ${
                    msg.role === "user"
                      ? "bg-white/[0.08] text-foreground/80"
                      : msg.error
                        ? "bg-red-500/10 border border-red-500/15 text-red-400/80"
                        : "bg-white/[0.03] border border-white/[0.06] text-foreground/70"
                  }`}>
                    {msg.role === "architect" && msg.blocks.length > 0 ? (
                      <ActivityMessageView message={{ role: "agent", blocks: msg.blocks, timestamp: msg.timestamp, cost: msg.cost, duration: msg.duration }} />
                    ) : msg.role === "architect" ? (
                      <pre className="text-xs font-mono whitespace-pre-wrap break-words leading-relaxed">{msg.content}</pre>
                    ) : (
                      <p className="text-xs">{msg.content}</p>
                    )}
                    <p className="text-[9px] text-muted-foreground/20 mt-1">
                      {msg.timestamp.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                    </p>
                  </div>
                  {msg.directiveAction && (
                    <div className="max-w-[80%] mt-1.5 animate-in slide-in-from-bottom-2 fade-in duration-300">
                      <div className="rounded-lg overflow-hidden border border-blue-500/20 bg-blue-500/[0.06]">
                        <div className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-500/10 border-b border-blue-500/15">
                          <span className="text-[10px]">📋</span>
                          <span className="text-[10px] font-mono uppercase tracking-wider text-blue-400/70">
                            {msg.directiveAction.type === "add" ? "Directive Issued" : msg.directiveAction.type === "done" ? "Directive Resolved" : "Directive Updated"}
                          </span>
                        </div>
                        <div className="px-3 py-2">
                          <p className="text-xs font-medium text-blue-200/90">{msg.directiveAction.title}</p>
                          <div className="flex items-center gap-2 mt-1">
                            {msg.directiveAction.priority && (
                              <span className={`text-[9px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded-full border ${
                                msg.directiveAction.priority.toUpperCase() === "HIGH"
                                  ? "bg-red-500/15 text-red-400/80 border-red-500/20"
                                  : msg.directiveAction.priority.toUpperCase() === "MEDIUM"
                                    ? "bg-amber-500/15 text-amber-400/80 border-amber-500/20"
                                    : "bg-green-500/15 text-green-400/80 border-green-500/20"
                              }`}>
                                {msg.directiveAction.priority}
                              </span>
                            )}
                            {msg.directiveAction.description && (
                              <span className="text-[10px] text-blue-400/40">{msg.directiveAction.description}</span>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                  {msg.blueprintUpdate && (
                    <BlueprintUpdateCard
                      path={msg.blueprintUpdate.path}
                      description={msg.blueprintUpdate.description}
                    />
                  )}
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
                className={`p-2 rounded-lg transition-all disabled:opacity-20 disabled:cursor-not-allowed ${
                  chatInput.trim()
                    ? "bg-white/[0.08] text-foreground/70 hover:bg-white/[0.12]"
                    : "text-muted-foreground/40"
                }`}
              >
                <IconArrowUp size={16} />
              </button>
            </div>
          </div>
        </div>

        {/* Directives sidebar (1/3 width) */}
        <div>
          <DirectivesPanel directives={directives} highlightTitle={highlightTitle} />
        </div>
      </div>
      )}

      {activeTab === "blueprint" && (
        <BlueprintBrowser compact />
      )}

      {activeTab === "tasks" && (
        <DirectivesPanel directives={directives} highlightTitle={highlightTitle} />
      )}
    </div>
  );
}
