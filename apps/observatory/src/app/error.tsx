"use client";

import { useEffect } from "react";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("Observatory error:", error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center min-h-screen">
      <div className="text-center">
        <div className="w-16 h-16 rounded-2xl bg-red-500/10 border border-red-500/20 flex items-center justify-center mx-auto mb-5">
          <span className="text-2xl">⚠</span>
        </div>
        <h2 className="text-lg font-heading text-foreground/80 mb-2">
          Something went wrong
        </h2>
        <p className="text-sm text-muted-foreground/40 font-mono mb-6 max-w-md">
          {error.message || "An unexpected error occurred"}
        </p>
        <div className="flex items-center gap-3 justify-center">
          <button
            onClick={reset}
            className="px-5 py-2.5 rounded-xl text-sm bg-white/[0.04] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.06] transition-all"
          >
            Try again
          </button>
          <a
            href="/"
            className="px-5 py-2.5 rounded-xl text-sm text-muted-foreground/40 hover:text-foreground/60 transition-colors"
          >
            Go home
          </a>
        </div>
      </div>
    </div>
  );
}
