"use client";

import { useState, useEffect } from "react";
import { IconPackage, IconDownload, IconExternalLink } from "@tabler/icons-react";
import { Skeleton } from "@/components/ui/skeleton";

interface Package {
  name: string;
  version: string;
  description: string;
}

export default function MarketplacePage() {
  const [packages, setPackages] = useState<Package[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/packages")
      .then((r) => r.json())
      .then((data) => {
        setPackages(data.packages ?? []);
        if (data.error) setError(data.error);
        setLoading(false);
      })
      .catch(() => {
        setLoading(false);
      });
  }, []);

  return (
    <div className="p-8 space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-heading tracking-wide text-foreground/90">Marketplace</h1>
          <p className="text-xs font-mono text-muted-foreground/40 mt-0.5">
            Installed packages and extensions
          </p>
        </div>
        <a
          href="https://spwn.sh/marketplace"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm bg-white/[0.04] text-foreground/60 hover:text-foreground/80 hover:bg-white/[0.08] border border-white/[0.06] transition-all"
        >
          <IconExternalLink size={16} />
          Browse Marketplace
        </a>
      </div>

      {/* Loading state */}
      {loading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="glass-subtle p-5 flex items-center gap-4">
              <Skeleton className="w-10 h-10 rounded-lg" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-64" />
              </div>
              <Skeleton className="h-8 w-20 rounded-lg" />
            </div>
          ))}
        </div>
      )}

      {/* Installed packages */}
      {!loading && packages.length > 0 && (
        <div>
          <h2 className="text-sm font-heading uppercase tracking-widest text-muted-foreground/40 mb-4">
            Installed ({packages.length})
          </h2>
          <div className="space-y-2">
            {packages.map((pkg) => (
              <div key={pkg.name} className="glass-subtle p-5 flex items-center gap-4">
                <div className="w-10 h-10 rounded-lg bg-white/[0.04] border border-white/[0.06] flex items-center justify-center">
                  <IconPackage size={20} className="text-muted-foreground/40" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono text-foreground/80">{pkg.name}</p>
                    {pkg.version && (
                      <span className="text-[10px] font-mono text-muted-foreground/30 px-1.5 py-0.5 rounded bg-white/[0.04]">
                        v{pkg.version}
                      </span>
                    )}
                  </div>
                  {pkg.description && (
                    <p className="text-xs text-muted-foreground/40 mt-0.5 truncate">{pkg.description}</p>
                  )}
                </div>
                <button className="px-3 py-1.5 rounded-lg text-[11px] text-red-400/50 hover:text-red-400 hover:bg-red-500/10 transition-colors">
                  Uninstall
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Empty state */}
      {!loading && packages.length === 0 && (
        <div className="text-center py-20">
          <div className="w-16 h-16 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mx-auto mb-6">
            <IconPackage size={32} className="text-muted-foreground/20" />
          </div>
          <h2 className="text-lg font-heading text-muted-foreground/50 mb-2">No packages installed</h2>
          <p className="text-sm text-muted-foreground/30 mb-6 max-w-md mx-auto">
            Browse the marketplace to discover configs, agents, playbooks, and extensions for your universe.
          </p>
          <div className="flex gap-3 justify-center">
            <a
              href="https://spwn.sh/marketplace"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm bg-white/[0.06] text-foreground/70 hover:bg-white/[0.1] hover:text-foreground/90 border border-white/[0.08] transition-all"
            >
              <IconExternalLink size={16} />
              Browse Marketplace
            </a>
            <button
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm text-muted-foreground/40 hover:text-foreground/60 hover:bg-white/[0.04] transition-all"
              onClick={() => {/* TODO: install dialog */}}
            >
              <IconDownload size={16} />
              Install from URL
            </button>
          </div>
          {error && (
            <p className="text-[11px] font-mono text-muted-foreground/25 mt-6">
              CLI: {error}
            </p>
          )}
          <div className="mt-8 glass-subtle inline-block px-4 py-2.5 font-mono text-[11px] text-muted-foreground/30">
            spwn get install &lt;package&gt;
          </div>
        </div>
      )}
    </div>
  );
}
