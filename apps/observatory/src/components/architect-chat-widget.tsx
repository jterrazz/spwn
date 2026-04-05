"use client";

import { useState, useEffect, useRef } from "react";
import { usePathname, useRouter } from "next/navigation";
import {
  IconArrowUp,
  IconChevronDown,
  IconMaximize,
  IconHexagonFilled,
  IconMessageCircle,
} from "@tabler/icons-react";
import { useArchitectChat } from "@/contexts/architect-chat-context";
import { ActivityMessageView } from "@/components/activity-blocks";
import { KnowledgeUpdateCard } from "@/components/knowledge-browser";

function ArchitectGlyph({ isRunning, isActive }: { isRunning: boolean; isActive: boolean }) {
  return (
    <span className="relative inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center self-center overflow-visible">
      <style jsx>{`
        @keyframes architect-rainbow-hue {
          0% { filter: hue-rotate(0deg) brightness(1.2); }
          100% { filter: hue-rotate(360deg) brightness(1.2); }
        }
      `}</style>
      {isRunning && isActive ? (
        <>
          <IconHexagonFilled
            size={14}
            className="absolute inset-0 m-auto blur-[6px] opacity-70 text-pink-400"
            style={{ animation: "architect-rainbow-hue 3s linear infinite" }}
          />
          <IconHexagonFilled
            size={14}
            className="absolute inset-0 m-auto text-pink-400"
            style={{ animation: "architect-rainbow-hue 3s linear infinite" }}
          />
        </>
      ) : (
        <span className="block leading-none translate-y-[0.5px]">
          <IconHexagonFilled size={14} className="text-muted-foreground/45" />
        </span>
      )}
    </span>
  );
}

export function ArchitectChatWidget() {
  const pathname = usePathname();
  const router = useRouter();
  const [expanded, setExpanded] = useState(false);
  const chatEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  const {
    messages,
    chatInput,
    setChatInput,
    sending,
    sendMessage,
    isRunning,
  } = useArchitectChat();

  // Auto-scroll when new messages arrive
  useEffect(() => {
    if (expanded && messages.length > 0) {
      chatEndRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  }, [messages, expanded]);

  // Focus input when expanding
  useEffect(() => {
    if (expanded) {
      inputRef.current?.focus();
    }
  }, [expanded]);

  // Close when clicking outside the expanded panel
  useEffect(() => {
    if (!expanded) return;

    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (panelRef.current?.contains(target)) return;
      setExpanded(false);
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("touchstart", handlePointerDown, { passive: true });
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("touchstart", handlePointerDown);
    };
  }, [expanded]);

  // Hide on architect page — the full page takes over
  if (pathname === "/architect" || pathname.startsWith("/architect/")) {
    return null;
  }

  const handleSend = () => {
    sendMessage();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  // ── Expanded panel ──
  if (expanded) {
    return (
      <div ref={panelRef} className="fixed bottom-4 right-4 z-[200] w-[420px] h-[540px] flex flex-col rounded-2xl border border-white/[0.08] bg-background/95 backdrop-blur-xl shadow-2xl shadow-black/30 animate-in slide-in-from-bottom-4 fade-in duration-200 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-white/[0.06]">
          <div className="flex items-center gap-1 rounded-full border border-foreground/[0.08] dark:border-white/[0.1] bg-foreground/[0.04] dark:bg-white/[0.05] px-2.5 py-1.5 shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)] backdrop-blur-md">
            <ArchitectGlyph isRunning={isRunning} isActive={sending} />
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={() => { setExpanded(false); router.push("/architect"); }}
              className="w-7 h-7 flex items-center justify-center rounded-md text-muted-foreground/30 hover:text-foreground/60 transition-colors"
              title="Open full page"
            >
              <IconMaximize size={14} />
            </button>
            <button
              onClick={() => setExpanded(false)}
              className="w-7 h-7 flex items-center justify-center rounded-md text-muted-foreground/30 hover:text-foreground/60 transition-colors"
              title="Minimize"
            >
              <IconChevronDown size={14} />
            </button>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-3 space-y-2.5">
          {messages.length === 0 && (
            <div className="flex flex-col items-center justify-center h-full text-center">
              <IconMessageCircle size={24} className="text-muted-foreground/15 mb-2" />
              <p className="text-xs text-muted-foreground/30">Talk to the Architect</p>
              <p className="text-[10px] text-muted-foreground/20 mt-1 max-w-[260px]">
                Ask anything — create agents, manage worlds, or check status.
              </p>
              {!isRunning && (
                <p className="text-[9px] text-yellow-400/40 mt-2 font-mono">
                  Architect is offline — it will auto-start when you send a message
                </p>
              )}
            </div>
          )}
          {messages.map((msg, i) => (
            <div key={i} className={`flex flex-col ${msg.role === "user" ? "items-end" : "items-start"}`}>
              <div className={`max-w-[85%] rounded-xl px-3 py-2 ${
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
                  {msg.timestamp.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                </p>
              </div>
              {msg.stackAction && (
                <div className="max-w-[85%] mt-1 animate-in slide-in-from-bottom-2 fade-in duration-300">
                  <div className="rounded-lg overflow-hidden border border-blue-500/20 bg-blue-500/[0.06]">
                    <div className="flex items-center gap-1.5 px-2.5 py-1 bg-blue-500/10 border-b border-blue-500/15">
                      <span className="text-[9px]">📋</span>
                      <span className="text-[9px] font-mono uppercase tracking-wider text-blue-400/70">
                        {msg.stackAction.type === "push" ? "Task Pushed" : msg.stackAction.type === "pop" ? "Task Popped" : "Task Updated"}
                      </span>
                    </div>
                    <div className="px-2.5 py-1.5">
                      <p className="text-[11px] font-medium text-blue-200/90">{msg.stackAction.title}</p>
                    </div>
                  </div>
                </div>
              )}
              {msg.knowledgeUpdate && (
                <KnowledgeUpdateCard
                  path={msg.knowledgeUpdate.path}
                  description={msg.knowledgeUpdate.description}
                />
              )}
            </div>
          ))}
          {sending && (
            <div className="flex justify-start">
              <div className="bg-white/[0.03] border border-white/[0.06] rounded-xl px-3 py-2">
                <div className="flex items-center gap-2">
                  <div className="w-1.5 h-1.5 rounded-full bg-foreground/30 animate-pulse" />
                  <span className="text-[11px] text-muted-foreground/40">Thinking...</span>
                </div>
              </div>
            </div>
          )}
          <div ref={chatEndRef} />
        </div>

        {/* Input */}
        <div className="border-t border-white/[0.06] p-2.5 flex gap-2">
          <input
            ref={inputRef}
            value={chatInput}
            onChange={(e) => setChatInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Talk to the Architect..."
            className="flex-1 bg-transparent text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none"
            disabled={sending}
          />
          <button
            onClick={handleSend}
            disabled={!chatInput.trim() || sending}
            className={`p-1.5 rounded-lg transition-all disabled:opacity-20 disabled:cursor-not-allowed ${
              chatInput.trim()
                ? "bg-white/[0.08] text-foreground/70 hover:bg-white/[0.12]"
                : "text-muted-foreground/40"
            }`}
          >
            <IconArrowUp size={14} />
          </button>
        </div>
      </div>
    );
  }

  // ── Collapsed bar ──
  return (
    <>
      <div className="fixed bottom-4 right-4 z-[200] animate-in fade-in duration-200">
        <div className="flex items-center gap-2 rounded-full border border-foreground/[0.08] dark:border-white/[0.1] bg-foreground/[0.04] dark:bg-white/[0.05] backdrop-blur-md shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_1px_2px_rgba(0,0,0,0.04)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_1px_2px_rgba(0,0,0,0.18)] px-2.5 py-1.5 w-[300px]">
        <button
          onClick={() => setExpanded(true)}
          className="flex h-[30px] items-center justify-center rounded-full border border-transparent px-2 shrink-0"
          title={isRunning ? "Architect alive" : "Architect offline"}
        >
          <ArchitectGlyph isRunning={isRunning} isActive={sending} />
        </button>
        <input
          value={chatInput}
          onChange={(e) => setChatInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              if (chatInput.trim()) {
                handleSend();
                setExpanded(true);
              }
            }
          }}
          onFocus={() => setExpanded(true)}
          placeholder="Ask the Architect..."
          className="flex-1 bg-transparent text-[13px] text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none min-w-0"
          disabled={sending}
        />
        {chatInput.trim() ? (
          <button
            onClick={() => { handleSend(); setExpanded(true); }}
            className="p-1.5 rounded-full bg-white/[0.08] text-foreground/70 hover:bg-white/[0.12] transition-all shrink-0"
          >
            <IconArrowUp size={14} />
          </button>
        ) : null}
        </div>
      </div>
    </>
  );
}
