"use client";

import { createContext, useContext, useState, useEffect, useRef, useCallback } from "react";
import { apiGet, goApiUrl } from "@/lib/api-client";
import { streamChat } from "@/lib/stream-chat";
import type { ActivityBlock } from "@/lib/activity-types";

// ── Types ──

export interface ArchitectStatus {
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

interface StackActionData {
  type: "push" | "pop" | "update";
  title: string;
  priority?: string;
  description?: string;
}

export interface KnowledgeUpdateData {
  path: string;
  description?: string;
}

export interface ChatMessage {
  role: "user" | "architect";
  content: string;
  blocks: ActivityBlock[];
  timestamp: Date;
  error?: boolean;
  cost?: number;
  duration?: number;
  stackAction?: StackActionData;
  knowledgeUpdate?: KnowledgeUpdateData;
}

export interface StackItem {
  text: string;
  done: boolean;
  priority?: "high" | "medium" | "low";
  description?: string;
}

export interface StackData {
  focus: StackItem[];
  queued: StackItem[];
  done: StackItem[];
  raw: string;
}

// ── Stack parsing ──

function extractPriority(text: string): { cleanText: string; priority?: "high" | "medium" | "low"; description?: string } {
  let cleanText = text;
  let priority: "high" | "medium" | "low" | undefined;
  let description: string | undefined;

  const priorityMatch = cleanText.match(/\s*[\[(]*\*{0,2}(HIGH|MEDIUM|LOW)\*{0,2}[\])]*\s*/i);
  if (priorityMatch) {
    priority = priorityMatch[1].toLowerCase() as "high" | "medium" | "low";
    cleanText = cleanText.replace(priorityMatch[0], " ").trim();
  }

  const descSep = cleanText.match(/\s+[-—]\s+(.+)$/);
  if (descSep) {
    description = descSep[1];
    cleanText = cleanText.slice(0, cleanText.length - descSep[0].length).trim();
  }

  return { cleanText, priority, description };
}

export function parseStackMd(raw: string): StackData {
  const lines = raw.split("\n");
  const focus: StackItem[] = [];
  const queued: StackItem[] = [];
  const doneItems: StackItem[] = [];

  let section = "queued";

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.toLowerCase().startsWith("## focus")) {
      section = "focus";
      continue;
    }
    if (trimmed.toLowerCase().startsWith("## queued")) {
      section = "queued";
      continue;
    }
    if (trimmed.toLowerCase().startsWith("## done")) {
      section = "done";
      continue;
    }
    if (trimmed.startsWith("#")) continue;

    const checkMatch = trimmed.match(/^-\s*\[([ xX])\]\s*(.+)/);
    if (checkMatch) {
      const isDone = checkMatch[1] !== " ";
      const { cleanText, priority, description } = extractPriority(checkMatch[2]);
      const item: StackItem = { text: cleanText, done: isDone, priority, description };
      if (isDone) {
        doneItems.push(item);
      } else if (section === "focus") {
        focus.push(item);
      } else {
        queued.push(item);
      }
      continue;
    }

    const listMatch = trimmed.match(/^-\s+(.+)/);
    if (listMatch) {
      const { cleanText, priority, description } = extractPriority(listMatch[1]);
      const item: StackItem = { text: cleanText, done: section === "done", priority, description };
      if (section === "focus") {
        focus.push(item);
      } else if (section === "done") {
        doneItems.push(item);
      } else {
        queued.push(item);
      }
    }
  }

  return { focus, queued, done: doneItems, raw };
}

// ── Context ──

interface ArchitectChatContextValue {
  messages: ChatMessage[];
  chatInput: string;
  setChatInput: (value: string) => void;
  sending: boolean;
  sendMessage: () => Promise<void>;
  architectStatus: ArchitectStatus | null;
  isRunning: boolean;
  stack: StackData | null;
  highlightTitle: string | null;
  refreshStatus: () => void;
  setArchitectStatus: React.Dispatch<React.SetStateAction<ArchitectStatus | null>>;
  loading: boolean;
}

const ArchitectChatContext = createContext<ArchitectChatContextValue | null>(null);

export function useArchitectChat() {
  const ctx = useContext(ArchitectChatContext);
  if (!ctx) throw new Error("useArchitectChat must be used within ArchitectChatProvider");
  return ctx;
}

export function ArchitectChatProvider({ children }: { children: React.ReactNode }) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [sending, setSending] = useState(false);
  const [architectStatus, setArchitectStatus] = useState<ArchitectStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [stack, setStack] = useState<StackData | null>(null);
  const [highlightTitle, setHighlightTitle] = useState<string | null>(null);

  const fetchStack = useCallback(() => {
    apiGet<{ content: string }>("/api/architect/stack")
      .then((data) => {
        setStack(parseStackMd(data.content));
      })
      .catch(() => {});
  }, []);

  const refreshStatus = useCallback(() => {
    apiGet<ArchitectStatus>("/api/architect/status")
      .catch(() => ({ status: "stopped" as const, containerId: null, uptime: null }))
      .then((archStatus) => {
        setArchitectStatus(archStatus);
        setLoading(false);
      });
  }, []);

  // Load conversation history on mount
  useEffect(() => {
    apiGet<{ sessions: { id: string; messages: { role: string; content: string; timestamp: string; type: string; toolName?: string; cost?: number; durationMs?: number }[]; startedAt: string; cost?: number }[] }>("/api/architect/history")
      .then((data) => {
        if (!data.sessions || data.sessions.length === 0) return;
        const historyMsgs: ChatMessage[] = [];
        for (let si = 0; si < data.sessions.length; si++) {
          const session = data.sessions[si];
          if (si > 0) {
            historyMsgs.push({
              role: "architect",
              content: `── Session ${session.id.slice(0, 8)} · ${session.startedAt ? new Date(session.startedAt).toLocaleString() : "unknown"} ──`,
              blocks: [{ type: "text" as const, content: `── Previous session ──` }],
              timestamp: session.startedAt ? new Date(session.startedAt) : new Date(),
            });
          }
          for (const msg of session.messages) {
            if (msg.type === "text" && msg.role === "user") {
              historyMsgs.push({
                role: "user",
                content: msg.content,
                blocks: [{ type: "text" as const, content: msg.content }],
                timestamp: msg.timestamp ? new Date(msg.timestamp) : new Date(),
              });
            } else if (msg.type === "text" && msg.role === "assistant") {
              historyMsgs.push({
                role: "architect",
                content: msg.content,
                blocks: [{ type: "text" as const, content: msg.content }],
                timestamp: msg.timestamp ? new Date(msg.timestamp) : new Date(),
              });
            } else if (msg.type === "tool_use") {
              historyMsgs.push({
                role: "architect",
                content: `🔧 ${msg.toolName || "tool"}`,
                blocks: [{ type: "tool_use", tool: msg.toolName || "tool", input: {}, id: `hist-${si}-${historyMsgs.length}` } as ActivityBlock],
                timestamp: msg.timestamp ? new Date(msg.timestamp) : new Date(),
              });
            } else if (msg.type === "result" && msg.cost) {
              const lastArch = [...historyMsgs].reverse().find(m => m.role === "architect");
              if (lastArch) {
                lastArch.cost = msg.cost;
                lastArch.duration = msg.durationMs;
              }
            }
          }
        }
        if (historyMsgs.length > 0) {
          setMessages(historyMsgs);
        }
      })
      .catch(() => {});
  }, []);

  // Status polling
  useEffect(() => {
    refreshStatus();
    fetchStack();
    const interval = setInterval(refreshStatus, 10000);
    return () => clearInterval(interval);
  }, [refreshStatus, fetchStack]);

  const doTalk = useCallback(async (msg: string) => {
    // Use a ref-like approach: capture current length before adding placeholder
    let msgIndex: number;
    setMessages((prev) => {
      msgIndex = prev.length;
      return [...prev, {
        role: "architect" as const, content: "", blocks: [], timestamp: new Date(),
      }];
    });

    await streamChat({
      url: goApiUrl("/api/architect/talk"),
      body: { message: msg },
      onBlocks: (newBlocks) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[msgIndex!];
          if (last && last.role === "architect") {
            const allBlocks = [...last.blocks, ...newBlocks];
            const textContent = allBlocks
              .filter((b) => b.type === "text")
              .map((b) => (b as { content: string }).content)
              .join("");
            updated[msgIndex!] = { ...last, blocks: allBlocks, content: textContent };
          }
          return updated;
        });
      },
      onDone: (meta) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[msgIndex!];
          if (last && last.role === "architect") {
            updated[msgIndex!] = { ...last, cost: meta.cost, duration: meta.duration };
          }
          return updated;
        });
        // Refresh stack after response completes
        fetchStack();
      },
      onError: (error) => {
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[msgIndex!];
          if (last && last.role === "architect") {
            updated[msgIndex!] = {
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
  }, [fetchStack]);

  const sendMessage = useCallback(async () => {
    const msg = chatInput.trim();
    if (!msg || sending) return;

    const userMsg: ChatMessage = { role: "user", content: msg, blocks: [{ type: "text", content: msg }], timestamp: new Date() };
    setMessages((prev) => [...prev, userMsg]);
    setChatInput("");
    setSending(true);

    try {
      let running = architectStatus?.status === "running";

      if (!running) {
        setMessages((prev) => [...prev, {
          role: "architect",
          content: "Starting Architect...",
          blocks: [{ type: "text" as const, content: "Starting Architect..." }],
          timestamp: new Date(),
        }]);

        try {
          await fetch(goApiUrl("/api/architect/start"), { method: "POST" });
        } catch {
          try {
            await fetch(goApiUrl("/api/architect/start"), { method: "POST" });
          } catch {
            setMessages((prev) => [...prev, {
              role: "architect",
              content: "Failed to auto-start Architect. Please start it manually.",
              blocks: [{ type: "error" as const, content: "Failed to auto-start Architect. Please start it manually." }],
              timestamp: new Date(),
              error: true,
            }]);
            setSending(false);
            return;
          }
        }

        for (let i = 0; i < 15; i++) {
          await new Promise((resolve) => setTimeout(resolve, 2000));
          try {
            const statusRes = await fetch(goApiUrl("/api/architect/status"));
            const statusData = await statusRes.json();
            if (statusData.status === "running") {
              running = true;
              setArchitectStatus((s) => s ? { ...s, status: "running" } : { status: "running", containerId: null, uptime: null });
              break;
            }
          } catch {}
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
          return;
        }
      }

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
    }
  }, [chatInput, sending, architectStatus, doTalk]);

  const isRunning = architectStatus?.status === "running";

  return (
    <ArchitectChatContext.Provider
      value={{
        messages,
        chatInput,
        setChatInput,
        sending,
        sendMessage,
        architectStatus,
        isRunning,
        stack,
        highlightTitle,
        refreshStatus,
        setArchitectStatus,
        loading,
      }}
    >
      {children}
    </ArchitectChatContext.Provider>
  );
}
