"use client";

import { useState, useEffect, useRef } from "react";
import { useProgressMessages } from "@/hooks/use-progress-messages";
import { ProgressShimmer } from "@/components/progress-shimmer";
import { Skeleton } from "@/components/ui/skeleton";
import { goApiUrl } from "@/lib/api-client";
import { Chat, type ChatBubble } from "@/components/chat";
import { usePageTitle } from "@/hooks/use-page-title";
import {
  IconMessageCircle,
  IconPlayerPlay,
  IconRefresh,
  IconPlayerStop,
  IconHexagonFilled,
  IconTerminal2,
} from "@tabler/icons-react";
import { useArchitectChat } from "@/contexts/architect-chat-context";
import { PageHeader } from "@/components/page-header";
import { Page } from "@/components/page";
import { ActionButton } from "@/components/action-button";
import { MetricGrid } from "@/components/ds";

// ── Architect States ────────────────────────────────────────────────────

type ArchitectState = "offline" | "starting" | "running" | "stopping";

function deriveState(isRunning: boolean, actionLoading: string | null): ArchitectState {
  if (actionLoading === "start") return "starting";
  if (actionLoading === "stop") return "stopping";
  return isRunning ? "running" : "offline";
}

// ── Offline State ───────────────────────────────────────────────────────

function OfflineView({ onStart, disabled }: { onStart: () => void; disabled: boolean }) {
  return (
    <div className="flex-1 flex items-center justify-center -mt-12">
      <div className="flex flex-col items-center text-center max-w-md">
        <div className="w-16 h-16 rounded-2xl bg-white/[0.04] border border-white/[0.08] flex items-center justify-center mb-6">
          <IconHexagonFilled size={28} className="text-muted-foreground/20" />
        </div>

        <h2 className="text-lg font-heading tracking-wide text-foreground/70 mb-2">
          Architect is offline
        </h2>
        <p className="text-sm text-muted-foreground/40 mb-8 leading-relaxed">
          The Architect runs in the background and manages everything for you: creating agents, spawning worlds, and keeping track of tasks. Start it to begin working.
        </p>

        <button
          onClick={onStart}
          disabled={disabled}
          className="group flex items-center gap-3 px-6 py-3 rounded-xl bg-white/[0.06] border border-white/[0.10] hover:bg-white/[0.10] hover:border-white/[0.16] transition-all duration-200 disabled:opacity-40"
        >
          <IconPlayerPlay size={18} className="text-green-400/80 group-hover:text-green-400" />
          <span className="text-sm font-medium text-foreground/70 group-hover:text-foreground/90">Start Architect</span>
        </button>

        <div className="mt-6 flex items-center gap-2 text-[11px] text-muted-foreground/25 font-mono">
          <IconTerminal2 size={13} />
          <span>spwn architect start</span>
        </div>
      </div>
    </div>
  );
}

// ── Starting State ──────────────────────────────────────────────────────

function StartingView({ progressMessage }: { progressMessage: string }) {
  return (
    <div className="flex-1 flex items-center justify-center -mt-12">
      <div className="flex flex-col items-center text-center max-w-md">
        <div className="w-16 h-16 rounded-2xl bg-white/[0.04] border border-white/[0.08] flex items-center justify-center mb-6 relative">
          <IconHexagonFilled size={28} className="text-yellow-400/40 animate-pulse" />
        </div>

        <h2 className="text-lg font-heading tracking-wide text-foreground/70 mb-2">
          Starting Architect
        </h2>
        <p className="text-sm text-muted-foreground/40 mb-4 leading-relaxed">
          {progressMessage}
        </p>

        <div className="w-48">
          <ProgressShimmer active message="" />
        </div>
      </div>
    </div>
  );
}

// ── Main Page ───────────────────────────────────────────────────────────

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
    highlightTitle,
    setArchitectStatus,
    loading,
  } = useArchitectChat();

  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [startPolling, setStartPolling] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const state = deriveState(isRunning, actionLoading);

  const startProgressMessage = useProgressMessages(state === "starting", [
    { after: 0, text: "Sending start signal..." },
    { after: 3, text: "Building Docker image (first run may take a few minutes)..." },
    { after: 30, text: "Installing tools and dependencies..." },
    { after: 60, text: "Almost ready..." },
    { after: 120, text: "Still building... large images take time on first run." },
  ]);

  const stopProgressMessage = useProgressMessages(state === "stopping", [
    { after: 0, text: "Stopping architect..." },
    { after: 5, text: "Shutting down container..." },
    { after: 15, text: "Cleaning up..." },
  ]);

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
    setTimeout(() => setFeedback(null), 4000);
  };

  // Poll for architect status after starting
  useEffect(() => {
    if (!startPolling) return;
    pollRef.current = setInterval(async () => {
      try {
        const res = await fetch(goApiUrl("/api/architect/status"));
        if (res.ok) {
          const data = await res.json();
          if (data.status === "running") {
            setStartPolling(false);
            setActionLoading(null);
            setArchitectStatus((s) => s ? { ...s, status: "running" } : s);
            showFeedback("Architect started");
          }
        }
      } catch {
        // ignore polling errors
      }
    }, 3000);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [startPolling, setArchitectStatus]);

  const handleStart = async () => {
    setActionLoading("start");
    try {
      const res = await fetch(goApiUrl("/api/architect/start"), { method: "POST" });
      if (res.ok) {
        setStartPolling(true);
      } else {
        const data = await res.json().catch(() => ({ error: "Unknown error" }));
        showFeedback(`Error: ${data.error}`);
        setActionLoading(null);
      }
    } catch {
      showFeedback("Error: Failed to connect to API");
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
        description={
          state === "running"
            ? "Running. Talk to it in natural language."
            : state === "starting"
              ? "Starting..."
              : state === "stopping"
                ? "Stopping..."
                : "Offline"
        }
        actions={
          state === "running" ? (
            <>
              <ActionButton
                compact
                onClick={handleStart}
                disabled={actionLoading !== null}
                label="Restart"
                icon={<IconRefresh size={16} stroke={2.2} />}
              />
              <ActionButton
                compact
                danger
                onClick={handleStop}
                disabled={actionLoading !== null}
                label="Stop"
                icon={<IconPlayerStop size={16} stroke={2.2} />}
              />
            </>
          ) : state === "offline" ? (
            <ActionButton
              compact
              onClick={handleStart}
              disabled={actionLoading !== null}
              label="Start"
              icon={<IconPlayerPlay size={16} stroke={2.2} />}
            />
          ) : null
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

      {/* ── Offline ── */}
      {state === "offline" && !loading && (
        <OfflineView onStart={handleStart} disabled={actionLoading !== null} />
      )}

      {/* ── Starting ── */}
      {state === "starting" && (
        <StartingView progressMessage={startProgressMessage} />
      )}

      {/* ── Stopping ── */}
      {state === "stopping" && (
        <div className="flex-1 flex items-center justify-center -mt-12">
          <div className="flex flex-col items-center text-center">
            <p className="text-sm text-muted-foreground/50 mb-4">{stopProgressMessage}</p>
            <div className="w-48">
              <ProgressShimmer active message="" />
            </div>
          </div>
        </div>
      )}

      {/* ── Loading (initial) ── */}
      {loading && state === "offline" && (
        <div className="flex-1 flex items-center justify-center -mt-12">
          <div className="flex flex-col items-center gap-4">
            <Skeleton className="h-16 w-16 rounded-2xl" />
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-3 w-56" />
          </div>
        </div>
      )}

      {/* ── Running ── */}
      {state === "running" && (
        <>
          {/* KPI Metrics */}
          <MetricGrid columns={2} className="w-fit gap-x-10" items={[
            { label: "Worlds", value: kpis?.worlds ?? 0 },
            { label: "Agents", value: kpis?.agents ?? 0 },
          ]} />

          {/* Chat */}
          <Chat
            className="h-[480px]"
            messages={bubbles}
            onSend={handleSendMessage}
            disabled={sending}
            typingText="Architect is thinking..."
            placeholder="Talk to the Architect..."
            autoFocus
            input={chatInput}
            onInputChange={setChatInput}
            emptyState={
              <div className="flex flex-col items-center justify-center text-center">
                <IconMessageCircle size={28} className="text-muted-foreground/15 mb-3" />
                <p className="text-sm text-muted-foreground/30">Ask anything</p>
                <p className="text-[11px] text-muted-foreground/20 mt-1 max-w-sm">
                  &quot;Create an agent for the API project&quot;, &quot;What&apos;s running?&quot;, &quot;Spawn a world for the frontend repo&quot;
                </p>
              </div>
            }
          />
        </>
      )}
    </Page>
  );
}
