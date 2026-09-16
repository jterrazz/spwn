'use client';

import { IconDownload, IconExternalLink, IconPackage } from '@tabler/icons-react';
import { useEffect, useState } from 'react';

import { goApiUrl } from '@/api/client';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { Skeleton } from '@/components/ui/skeleton';
import { usePageTitle } from '@/hooks/use-page-title';

interface Dependency {
    name: string;
    version: string;
    description: string;
}

export default function MarketplacePage() {
    usePageTitle('Marketplace');
    const [packages, setPackages] = useState<Dependency[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<null | string>(null);

    useEffect(() => {
        fetch(goApiUrl('/api/dependencies'))
            .then(async (r) => await r.json())
            .then((data) => {
                setPackages(data.dependencies ?? []);
                if (data.error) {
                    setError(data.error);
                }
                setLoading(false);
            })
            .catch(() => {
                setLoading(false);
            });
    }, []);

    return (
        <Page>
            {/* Header */}
            <PageHeader
                actions={
                    <a
                        className="text-foreground/60 hover:text-foreground/80 flex items-center gap-2 rounded-xl border border-white/[0.06] bg-white/[0.04] px-4 py-2 text-sm transition-all hover:bg-white/[0.08]"
                        href="https://spwn.sh/marketplace"
                        rel="noopener noreferrer"
                        target="_blank"
                    >
                        <IconExternalLink size={16} />
                        Browse Marketplace
                    </a>
                }
                description="Browse and install dependencies to extend your worlds."
                title="Marketplace"
            />

            {/* Loading state */}
            {loading && (
                <div className="space-y-3">
                    {[1, 2, 3].map((i) => (
                        <div className="glass-subtle flex items-center gap-4 p-5" key={i}>
                            <Skeleton className="h-10 w-10 rounded-lg" />
                            <div className="flex-1 space-y-2">
                                <Skeleton className="h-4 w-40" />
                                <Skeleton className="h-3 w-64" />
                            </div>
                            <Skeleton className="h-8 w-20 rounded-lg" />
                        </div>
                    ))}
                </div>
            )}

            {/* Installed dependencies */}
            {!loading && packages.length > 0 && (
                <div>
                    <h2 className="font-heading text-muted-foreground/40 mb-4 text-sm tracking-widest uppercase">
                        Installed ({packages.length})
                    </h2>
                    <div className="space-y-2">
                        {packages.map((pkg) => (
                            <div
                                className="glass-subtle flex items-center gap-4 p-5"
                                key={pkg.name}
                            >
                                <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-white/[0.06] bg-white/[0.04]">
                                    <IconPackage className="text-muted-foreground/40" size={20} />
                                </div>
                                <div className="min-w-0 flex-1">
                                    <div className="flex items-center gap-2">
                                        <p className="text-foreground/80 font-mono text-sm">
                                            {pkg.name}
                                        </p>
                                        {pkg.version && (
                                            <span className="text-muted-foreground/30 rounded bg-white/[0.04] px-1.5 py-0.5 font-mono text-[10px]">
                                                v{pkg.version}
                                            </span>
                                        )}
                                    </div>
                                    {pkg.description && (
                                        <p className="text-muted-foreground/40 mt-0.5 truncate text-xs">
                                            {pkg.description}
                                        </p>
                                    )}
                                </div>
                                <button className="rounded-lg px-3 py-1.5 text-[11px] text-red-400/50 transition-colors hover:bg-red-500/10 hover:text-red-400">
                                    Uninstall
                                </button>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Empty state */}
            {!loading && packages.length === 0 && (
                <div className="py-20 text-center">
                    <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.06] bg-white/[0.03]">
                        <IconPackage className="text-muted-foreground/20" size={32} />
                    </div>
                    <h2 className="font-heading text-muted-foreground/50 mb-2 text-lg">
                        No dependencies installed
                    </h2>
                    <p className="text-muted-foreground/30 mx-auto mb-6 max-w-md text-sm">
                        Browse the marketplace to discover configs, agents, playbooks, and
                        extensions for your universe.
                    </p>
                    <div className="flex justify-center gap-3">
                        <a
                            className="text-foreground/70 hover:text-foreground/90 flex items-center gap-2 rounded-xl border border-white/[0.08] bg-white/[0.06] px-5 py-2.5 text-sm transition-all hover:bg-white/[0.1]"
                            href="https://spwn.sh/marketplace"
                            rel="noopener noreferrer"
                            target="_blank"
                        >
                            <IconExternalLink size={16} />
                            Browse Marketplace
                        </a>
                        <button
                            className="text-muted-foreground/40 hover:text-foreground/60 flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm transition-all hover:bg-white/[0.04]"
                            disabled
                        >
                            <IconDownload size={16} />
                            Install from URL
                        </button>
                    </div>
                    {error && (
                        <p className="text-muted-foreground/25 mt-6 font-mono text-[11px]">
                            CLI: {error}
                        </p>
                    )}
                    <div className="glass-subtle text-muted-foreground/30 mt-8 inline-block px-4 py-2.5 font-mono text-[11px]">
                        spwn install &lt;package&gt;
                    </div>
                </div>
            )}
        </Page>
    );
}
