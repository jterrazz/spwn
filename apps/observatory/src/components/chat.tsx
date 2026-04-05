"use client";

import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { IconArrowUp } from "@tabler/icons-react";

import { ActivityMessageView } from "@/components/activity-blocks";
import type { ActivityBlock } from "@/lib/activity-types";

/**
 * ChatBubble is the minimal shape any caller must provide. The chat does
 * not manage state — the parent owns the message list and passes it in.
 */
export interface ChatBubble {
  role: "user" | "assistant";
  blocks: ActivityBlock[];
  content: string;
  timestamp: Date;
  error?: boolean;
  cost?: number;
  duration?: number;
}

interface ChatProps {
  messages: ChatBubble[];
  onSend: (text: string) => void;
  placeholder?: string;
  disabled?: boolean;
  /** Shown when there are no messages. */
  emptyState?: ReactNode;
  /** Typing indicator text, e.g. "morpheus is thinking...". Omit to hide. */
  typingText?: string;
  /** Rendered immediately after each bubble — used for stack/knowledge cards. */
  extras?: (msg: ChatBubble, index: number) => ReactNode;
  /** Optional label shown under the assistant's bubbles ("morpheus", "architect"). */
  assistantLabel?: string;
  /** Optional label shown under the user's bubbles. Default "you". */
  userLabel?: string;
  /** Focus the input on mount. */
  autoFocus?: boolean;
  /** Extra className on the outer flex column. Use for height constraints. */
  className?: string;
  /**
   * Controlled-input props. When both are passed, the parent owns the
   * input state (lets the architect widget share its draft between
   * collapsed and expanded modes). When omitted, Chat owns state itself.
   */
  input?: string;
  onInputChange?: (value: string) => void;
}

/**
 * Chat is the shared, presentational surface used by both the agent chat
 * tab and the Architect chat widget. It's container-less by design —
 * only the message bubbles and the focused input border carry backgrounds,
 * so the chat blends into whatever page hosts it.
 */
export function Chat({
  messages,
  onSend,
  placeholder = "Message...",
  disabled,
  emptyState,
  typingText,
  extras,
  assistantLabel = "assistant",
  userLabel = "you",
  autoFocus,
  className = "",
  input: controlledInput,
  onInputChange,
}: ChatProps) {
  const [uncontrolledInput, setUncontrolledInput] = useState("");
  const isControlled = controlledInput !== undefined && onInputChange !== undefined;
  const input = isControlled ? controlledInput : uncontrolledInput;
  const setInput = isControlled ? onInputChange : setUncontrolledInput;

  const endRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }, [messages.length, disabled]);

  useEffect(() => {
    if (autoFocus) inputRef.current?.focus();
  }, [autoFocus]);

  const trimmed = input.trim();

  const handleSend = () => {
    if (!trimmed || disabled) return;
    onSend(trimmed);
    setInput("");
    requestAnimationFrame(() => inputRef.current?.focus());
  };

  return (
    <div className={`flex flex-col min-h-0 ${className}`}>
      {/* Messages area — transparent, only bubbles carry backgrounds */}
      <div className="flex-1 overflow-y-auto overflow-x-hidden py-2 pr-1 space-y-3">
        {messages.length === 0 && emptyState && (
          <div className="flex h-full items-center justify-center">{emptyState}</div>
        )}
        {messages.map((msg, i) => (
          <div
            key={i}
            className={`flex flex-col ${msg.role === "user" ? "items-end" : "items-start"}`}
          >
            <div
              className={`max-w-[85%] rounded-2xl px-3.5 py-2.5 transition-colors ${
                msg.role === "user"
                  ? "bg-white/[0.08] text-foreground/85"
                  : msg.error
                    ? "bg-red-500/10 border border-red-500/15 text-red-400/80"
                    : "bg-white/[0.03] border border-white/[0.06] text-foreground/75"
              }`}
            >
              {msg.role === "assistant" && msg.blocks.length > 0 ? (
                <ActivityMessageView
                  message={{
                    role: "agent",
                    blocks: msg.blocks,
                    timestamp: msg.timestamp,
                    cost: msg.cost,
                    duration: msg.duration,
                  }}
                />
              ) : (
                <p className="text-xs whitespace-pre-wrap break-words leading-relaxed">
                  {msg.content || (msg.role === "assistant" ? "…" : "")}
                </p>
              )}
              <p className="text-[9px] text-muted-foreground/25 mt-1">
                {msg.role === "assistant" ? assistantLabel : userLabel}
                {" · "}
                {msg.timestamp.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
              </p>
            </div>
            {extras?.(msg, i)}
          </div>
        ))}
        {disabled && typingText && (
          <div className="flex items-start">
            <div className="rounded-2xl bg-white/[0.03] border border-white/[0.06] px-3.5 py-2.5">
              <div className="flex items-center gap-2">
                <div className="flex gap-1">
                  <span className="w-1.5 h-1.5 rounded-full bg-foreground/30 animate-bounce" style={{ animationDelay: "0ms" }} />
                  <span className="w-1.5 h-1.5 rounded-full bg-foreground/30 animate-bounce" style={{ animationDelay: "150ms" }} />
                  <span className="w-1.5 h-1.5 rounded-full bg-foreground/30 animate-bounce" style={{ animationDelay: "300ms" }} />
                </div>
                <span className="text-xs text-muted-foreground/40">{typingText}</span>
              </div>
            </div>
          </div>
        )}
        <div ref={endRef} />
      </div>

      {/* Input — bare, with only a hairline separator to the scroll area */}
      <div className="mt-3 pt-3 border-t border-white/[0.04] flex items-end gap-2">
        <input
          ref={inputRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleSend();
            }
          }}
          placeholder={placeholder}
          disabled={disabled}
          className="flex-1 bg-transparent text-sm text-foreground/85 placeholder:text-muted-foreground/30 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
        />
        <button
          type="button"
          onClick={handleSend}
          disabled={!trimmed || disabled}
          aria-label="Send"
          className={`shrink-0 flex items-center justify-center w-8 h-8 rounded-full transition-all disabled:opacity-20 disabled:cursor-not-allowed ${
            trimmed
              ? "bg-foreground/85 text-background hover:bg-foreground"
              : "text-muted-foreground/40"
          }`}
        >
          <IconArrowUp size={15} stroke={2.4} />
        </button>
      </div>
    </div>
  );
}

/**
 * Suggestion pills shown inside the empty state. Each click fills the
 * input — onPick is usually just setInput. Kept here so both chats can
 * use the same styling.
 */
export function ChatSuggestions({
  suggestions,
  onPick,
}: {
  suggestions: string[];
  onPick: (text: string) => void;
}) {
  return (
    <div className="flex gap-2 flex-wrap justify-center max-w-sm">
      {suggestions.map((s) => (
        <button
          key={s}
          onClick={() => onPick(s)}
          className="px-3 py-1.5 rounded-full text-[11px] text-muted-foreground/40 bg-white/[0.03] border border-white/[0.06] hover:text-foreground/70 hover:bg-white/[0.06] transition-colors"
        >
          {s}
        </button>
      ))}
    </div>
  );
}
