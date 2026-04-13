"use client";

import { useEffect } from "react";

export default function WorldError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("World error:", error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh]">
      <div className="text-center">
        <div className="w-14 h-14 rounded-2xl bg-red-500/10 border border-red-500/20 flex items-center justify-center mx-auto mb-4">
          <span className="text-xl">🌍</span>
        </div>
        <h2 className="text-lg font-heading text-foreground/80 mb-2">
          World unavailable
        </h2>
        <p className="text-sm text-muted-foreground/40 font-mono mb-6 max-w-sm">
          {error.message || "Failed to load world data"}
        </p>
        <div className="flex items-center gap-3 justify-center">
          <button
            onClick={reset}
            className="px-5 py-2.5 rounded-xl text-sm bg-white/[0.04] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.06] transition-all"
          >
            Retry
          </button>
          <a
            href="/"
            className="px-5 py-2.5 rounded-xl text-sm text-muted-foreground/40 hover:text-foreground/60 transition-colors"
          >
            Back to worlds
          </a>
        </div>
      </div>
    </div>
  );
}
