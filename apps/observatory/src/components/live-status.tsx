"use client";

import { useState, useEffect } from "react";

export function LiveStatus() {
  const [isLive, setIsLive] = useState(true);

  useEffect(() => {
    const check = () => {
      fetch("/api/status")
        .then((r) => {
          setIsLive(r.ok);
        })
        .catch(() => {
          setIsLive(false);
        });
    };

    check();
    const interval = setInterval(check, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex items-center gap-1.5" title={isLive ? "API responding" : "API disconnected"}>
      <div
        className={`w-1.5 h-1.5 rounded-full transition-colors ${
          isLive
            ? "bg-green-500 shadow-[0_0_4px_rgba(34,197,94,0.6)]"
            : "bg-red-400 shadow-[0_0_4px_rgba(248,113,113,0.5)]"
        }`}
      />
      <span className={`text-[9px] font-mono uppercase tracking-wider ${
        isLive ? "text-green-500/60" : "text-red-400/60"
      }`}>
        {isLive ? "live" : "disconnected"}
      </span>
    </div>
  );
}
