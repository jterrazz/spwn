"use client";

import { useState, useEffect } from "react";
import { IconArrowUp, IconX } from "@tabler/icons-react";
import type { VersionInfo } from "@/hooks/use-version";

const DISMISS_KEY = "spwn-upgrade-dismissed";

interface UpgradeBannerProps {
  version: VersionInfo;
}

export function UpgradeBanner({ version }: UpgradeBannerProps) {
  const [dismissed, setDismissed] = useState(true); // start hidden to avoid flash

  useEffect(() => {
    // Check if this specific version was already dismissed
    const dismissedVersion = localStorage.getItem(DISMISS_KEY);
    setDismissed(dismissedVersion === version.latest);
  }, [version.latest]);

  if (!version.updateAvailable || dismissed) {
    return null;
  }

  const handleDismiss = () => {
    localStorage.setItem(DISMISS_KEY, version.latest);
    setDismissed(true);
  };

  return (
    <div className="mx-3 mb-3 rounded-lg border border-amber-500/20 bg-amber-500/[0.06] p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 text-amber-400">
          <IconArrowUp size={14} className="shrink-0 mt-0.5" />
          <span className="text-[11px] font-medium">
            spwn v{version.latest} available
          </span>
        </div>
        <button
          onClick={handleDismiss}
          className="text-muted-foreground/30 hover:text-muted-foreground/60 transition-colors"
        >
          <IconX size={12} />
        </button>
      </div>
      <p className="text-[10px] text-muted-foreground/50 mt-1.5 ml-[22px]">
        Current: v{version.current}
      </p>
      <div className="mt-2 ml-[22px] space-y-1">
        <p className="text-[10px] text-muted-foreground/40">Run in terminal:</p>
        <code className="block text-[10px] font-mono text-amber-400/70 bg-black/20 rounded px-2 py-1">
          $ spwn upgrade
        </code>
        <p className="text-[10px] text-muted-foreground/40 mt-1.5">Then restart:</p>
        <code className="block text-[10px] font-mono text-amber-400/70 bg-black/20 rounded px-2 py-1">
          $ spwn web
        </code>
      </div>
    </div>
  );
}
