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
import { Chat, type ChatBubble } from "@/components/chat";
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
  const panelRef = useRef<HTMLDivElement>(null);

  const {
    messages,
    chatInput,
    setChatInput,
    sending,
    sendMessage,
    isRunning,
  } = useArchitectChat();

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

  // Adapt the architect context's messages into the shared ChatBubble shape.
  const bubbles: ChatBubble[] = messages.map((m) => ({
    role: m.role === "architect" ? "assistant" : "user",
    blocks: m.blocks,
    content: m.content,
    timestamp: m.timestamp,
    error: m.error,
    cost: m.cost,
    duration: m.duration,
  }));

  // Chat is controlled via chatInput/setChatInput — at send time the
  // context's chatInput is already the latest value, so sendMessage()
  // reads it directly. The `text` arg is the same as chatInput here.
  const handleSend = () => {
    void sendMessage();
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

        {/* Messages + input via shared Chat — indexed by the original
            messages array so the extras closure can read stackAction etc. */}
        <Chat
          className="flex-1 px-3 pb-3"
          messages={bubbles}
          onSend={handleSend}
          disabled={sending}
          typingText="Thinking…"
          placeholder="Talk to the Architect..."
          autoFocus={expanded}
          input={chatInput}
          onInputChange={setChatInput}
          emptyState={
            <div className="flex flex-col items-center justify-center text-center">
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
          }
          extras={(_, i) => {
            const raw = messages[i];
            if (!raw) return null;
            return (
              <>
                {raw.stackAction && (
                  <div className="max-w-[85%] mt-1 animate-in slide-in-from-bottom-2 fade-in duration-300">
                    <div className="rounded-lg overflow-hidden border border-blue-500/20 bg-blue-500/[0.06]">
                      <div className="flex items-center gap-1.5 px-2.5 py-1 bg-blue-500/10 border-b border-blue-500/15">
                        <span className="text-[9px]">📋</span>
                        <span className="text-[9px] font-mono uppercase tracking-wider text-blue-400/70">
                          {raw.stackAction.type === "push" ? "Task Pushed" : raw.stackAction.type === "pop" ? "Task Popped" : "Task Updated"}
                        </span>
                      </div>
                      <div className="px-2.5 py-1.5">
                        <p className="text-[11px] font-medium text-blue-200/90">{raw.stackAction.title}</p>
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
