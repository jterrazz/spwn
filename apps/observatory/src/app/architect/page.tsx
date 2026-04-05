"use client";

import { useState, useEffect } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { goApiUrl } from "@/lib/api-client";
import { Chat, type ChatBubble } from "@/components/chat";
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
  IconBook2,
} from "@tabler/icons-react";
import { KnowledgeBrowser, KnowledgeUpdateCard } from "@/components/knowledge-browser";
import { useArchitectChat, type StackData, type StackItem } from "@/contexts/architect-chat-context";
import { PageHeader } from "@/components/page-header";
import { Page } from "@/components/page";

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

function StatusDot({ status }: { status: "focus" | "queued" | "done" }) {
  if (status === "done") {
    return <IconCircleCheck size={16} className="text-green-400/70 flex-shrink-0" />;
  }
  if (status === "focus") {
    return (
      <span className="relative flex h-3 w-3 flex-shrink-0">
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-yellow-400/40" />
        <span className="relative inline-flex rounded-full h-3 w-3 bg-yellow-400/70" />
      </span>
    );
  }
  return <span className="w-3 h-3 rounded-full bg-white/15 flex-shrink-0" />;
}

function StackCard({ item, status, isHighlighted }: { item: StackItem; status: "focus" | "queued" | "done"; isHighlighted: boolean }) {
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

function StackPanel({ stack, highlightTitle }: { stack: StackData | null; highlightTitle: string | null }) {
  const [showCompleted, setShowCompleted] = useState(false);

  if (!stack) {
    return (
      <div className="glass-subtle rounded-xl p-5 space-y-3">
        <div className="flex items-center gap-2.5">
          <IconClipboardList size={18} className="text-muted-foreground/30" />
          <h3 className="text-sm font-heading tracking-wide text-muted-foreground/50">Stack</h3>
        </div>
        <div className="space-y-2">
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-3/4 rounded-lg" />
        </div>
      </div>
    );
  }

  const isEmpty = stack.focus.length === 0 && stack.queued.length === 0 && stack.done.length === 0;
  const totalActive = stack.focus.length + stack.queued.length;

  return (
    <div className="glass-subtle rounded-xl p-5 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <IconClipboardList size={18} className="text-muted-foreground/40" />
          <h3 className="text-sm font-heading tracking-wide text-foreground/60">Stack</h3>
        </div>
        <span className="text-[9px] font-mono text-muted-foreground/25 px-2 py-0.5 rounded-full bg-white/[0.03] border border-white/[0.05]">
          {totalActive > 0 ? `${totalActive} active` : "managed by architect"}
        </span>
      </div>

      {isEmpty && (
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <div className="w-10 h-10 rounded-full bg-white/[0.04] flex items-center justify-center mb-3">
            <IconClipboardList size={20} className="text-muted-foreground/20" />
          </div>
          <p className="text-xs text-muted-foreground/35 font-medium">No tasks yet</p>
          <p className="text-[11px] text-muted-foreground/20 mt-1">Talk to the Architect to get started</p>
        </div>
      )}

      {stack.focus.length > 0 && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="w-1.5 h-1.5 rounded-full bg-yellow-400/60" />
            <p className="text-[10px] uppercase tracking-wider font-medium text-yellow-400/60">Focus</p>
            <span className="text-[9px] font-mono text-yellow-400/30 ml-auto">{stack.focus.length}</span>
          </div>
          <div className="space-y-1.5">
            {stack.focus.map((item, i) => (
              <StackCard
                key={`ip-${i}`}
                item={item}
                status="focus"
                isHighlighted={!!highlightTitle && item.text.includes(highlightTitle)}
              />
            ))}
          </div>
        </div>
      )}

      {stack.queued.length > 0 && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="w-1.5 h-1.5 rounded-full bg-white/20" />
            <p className="text-[10px] uppercase tracking-wider font-medium text-muted-foreground/35">Queued</p>
            <span className="text-[9px] font-mono text-muted-foreground/20 ml-auto">{stack.queued.length}</span>
          </div>
          <div className="space-y-1.5">
            {stack.queued.map((item, i) => (
              <StackCard
                key={`bl-${i}`}
                item={item}
                status="queued"
                isHighlighted={!!highlightTitle && item.text.includes(highlightTitle)}
              />
            ))}
          </div>
        </div>
      )}

      {stack.done.length > 0 && (
        <div className="pt-1 border-t border-white/[0.05]">
          <button
            onClick={() => setShowCompleted(!showCompleted)}
            className="flex items-center gap-2 w-full text-[10px] uppercase tracking-wider text-muted-foreground/25 hover:text-muted-foreground/40 transition-colors py-1"
          >
            {showCompleted ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
            <span>Done</span>
            <span className="ml-auto text-[9px] font-mono px-1.5 py-0.5 rounded-full bg-green-500/10 text-green-400/40 border border-green-500/10">
              {stack.done.length}
            </span>
          </button>
          {showCompleted && (
            <div className="space-y-1.5 mt-2">
              {stack.done.map((item, i) => (
                <StackCard
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

export default function ArchitectPage() {
  usePageTitle("Architect");

  const {
    messages,
    chatInput,
    setChatInput,
    sending,
    sendMessage,
    architectStatus,
    isRunning,
    stack,
    highlightTitle,
    setArchitectStatus,
    loading,
  } = useArchitectChat();

  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"chat" | "knowledge" | "tasks">("chat");

  // Map architect messages into the shared chat bubble shape.
  const bubbles: ChatBubble[] = messages.map((m) => ({
    role: m.role === "architect" ? "assistant" : "user",
    blocks: m.blocks,
    content: m.content,
    timestamp: m.timestamp,
    error: m.error,
    cost: m.cost,
    duration: m.duration,
  }));

  const showFeedback = (msg: string) => {
    setFeedback(msg);
    setTimeout(() => setFeedback(null), 3000);
  };

  const handleStart = async () => {
    setActionLoading("start");
    try {
      const res = await fetch(goApiUrl("/api/architect/start"), { method: "POST" });
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
      const res = await fetch(goApiUrl("/api/architect/stop"), { method: "POST" });
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

  const handleSendMessage = () => {
    void sendMessage();
  };

  const kpis = architectStatus?.kpis;

  return (
    <Page>
      <PageHeader
        title="Architect"
        description="Your always-on world builder — creates agents, spawns worlds, manages tasks."
        actions={
          isRunning ? (
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
          )
        }
      />

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
          label="Stack"
          value={loading ? "" : String(kpis?.tasksPending ?? 0)}
          sub="pending"
          accent="text-yellow-400"
          icon={<IconClock size={14} />}
          loading={loading}
        />
        <StatCard
          label="Done"
          value={loading ? "" : String(kpis?.tasksCompleted ?? 0)}
          sub="done"
          accent="text-green-400"
          icon={<IconCircleCheck size={14} />}
          loading={loading}
        />
      </div>

      {/* Tab bar */}
      <div className="flex items-center gap-1 border-b border-white/[0.06] pb-0">
        {([
          { key: "chat", label: "Chat", icon: <IconMessageCircle size={14} /> },
          { key: "knowledge", label: "Knowledge", icon: <IconBook2 size={14} /> },
          { key: "tasks", label: "Stack", icon: <IconClipboardList size={14} /> },
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

          <Chat
            className="h-[480px]"
            messages={bubbles}
            onSend={handleSendMessage}
            disabled={sending}
            typingText="Architect is thinking…"
            placeholder="Talk to the Architect..."
            autoFocus
            input={chatInput}
            onInputChange={setChatInput}
            emptyState={
              <div className="flex flex-col items-center justify-center text-center">
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
            }
            extras={(_, i) => {
              const raw = messages[i];
              if (!raw) return null;
              return (
                <>
                  {raw.stackAction && (
                    <div className="max-w-[80%] mt-1.5 animate-in slide-in-from-bottom-2 fade-in duration-300">
                      <div className="rounded-lg overflow-hidden border border-blue-500/20 bg-blue-500/[0.06]">
                        <div className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-500/10 border-b border-blue-500/15">
                          <span className="text-[10px]">📋</span>
                          <span className="text-[10px] font-mono uppercase tracking-wider text-blue-400/70">
                            {raw.stackAction.type === "push" ? "Task Pushed" : raw.stackAction.type === "pop" ? "Task Popped" : "Task Updated"}
                          </span>
                        </div>
                        <div className="px-3 py-2">
                          <p className="text-xs font-medium text-blue-200/90">{raw.stackAction.title}</p>
                          <div className="flex items-center gap-2 mt-1">
                            {raw.stackAction.priority && (
                              <span className={`text-[9px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded-full border ${
                                raw.stackAction.priority.toUpperCase() === "HIGH"
                                  ? "bg-red-500/15 text-red-400/80 border-red-500/20"
                                  : raw.stackAction.priority.toUpperCase() === "MEDIUM"
                                    ? "bg-amber-500/15 text-amber-400/80 border-amber-500/20"
                                    : "bg-green-500/15 text-green-400/80 border-green-500/20"
                              }`}>
                                {raw.stackAction.priority}
                              </span>
                            )}
                            {raw.stackAction.description && (
                              <span className="text-[10px] text-blue-400/40">{raw.stackAction.description}</span>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                  {raw.knowledgeUpdate && (
                    <KnowledgeUpdateCard
                      path={raw.knowledgeUpdate.path}
                      description={raw.knowledgeUpdate.description}
                    />
                  )}
                </>
              );
            }}
          />
        </div>

        {/* Stack sidebar (1/3 width) */}
        <div>
          <StackPanel stack={stack} highlightTitle={highlightTitle} />
        </div>
      </div>
      )}

      {activeTab === "knowledge" && (
        <KnowledgeBrowser compact architectMode />
      )}

      {activeTab === "tasks" && (
        <StackPanel stack={stack} highlightTitle={highlightTitle} />
      )}
    </Page>
  );
}
