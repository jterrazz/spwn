"use client";

import { useState, useEffect } from "react";
import { isGoApiAvailable, onConnectionStatusChange, getConnectionStatus, type ConnectionStatus } from "@/lib/api-client";

const STATUS_CONFIG: Record<ConnectionStatus, { dot: string; label: string; labelColor: string }> = {
  connected: {
    dot: "bg-green-500 shadow-[0_0_4px_rgba(34,197,94,0.6)]",
    label: "Connected",
    labelColor: "text-green-500/60",
  },
  degraded: {
    dot: "bg-yellow-500 shadow-[0_0_4px_rgba(234,179,8,0.6)]",
    label: "Degraded",
    labelColor: "text-yellow-500/60",
  },
  disconnected: {
    dot: "bg-red-400 shadow-[0_0_4px_rgba(248,113,113,0.5)]",
    label: "Disconnected",
    labelColor: "text-red-400/60",
  },
};

export function LiveStatus() {
  const [status, setStatus] = useState<ConnectionStatus>(getConnectionStatus());

  useEffect(() => {
    // Subscribe to status changes from the API client
    const unsub = onConnectionStatusChange(setStatus);

    // Also do periodic direct checks
    const check = async () => {
      const goUp = await isGoApiAvailable();
      if (goUp) {
        setStatus("connected");
        return;
      }
      // Fall back to Next.js API route check
      try {
        const res = await fetch("/api/status");
        setStatus(res.ok ? "degraded" : "disconnected");
      } catch {
        setStatus("disconnected");
      }
    };

    check();
    const interval = setInterval(check, 10000);
    return () => {
      clearInterval(interval);
      unsub();
    };
  }, []);

  const config = STATUS_CONFIG[status];
  const title = status === "connected"
    ? "Go API responding"
    : status === "degraded"
      ? "Using fallback (exec mode)"
      : "API disconnected";

  return (
    <div className="flex items-center gap-1.5" title={title}>
      <div
        className={`w-1.5 h-1.5 rounded-full transition-colors ${config.dot}`}
      />
      <span className={`text-[9px] font-mono uppercase tracking-wider ${config.labelColor}`}>
        {config.label}
      </span>
    </div>
  );
}
