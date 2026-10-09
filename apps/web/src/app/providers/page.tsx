'use client';

import { IconAlertTriangle, IconPlugConnected, IconTerminal2 } from '@tabler/icons-react';
import { useState } from 'react';

import { Page } from '../../components/page';
import { PageHeader } from '../../components/page-header';
import { ConfigureModal } from '../../components/providers/configure-modal';
import { ProviderRow } from '../../components/providers/provider-row';
import { useProviders } from '../../components/providers/use-providers';
import { Skeleton } from '../../components/ui/skeleton';
import { inBackground } from '../../events/in-background';
import { FeedbackLine, useFeedback } from '../../hooks/use-feedback';
import { usePageTitle } from '../../hooks/use-page-title';

function Loading() {
    return (
        <div className="space-y-4">
            {[1, 2, 3].map((index) => (
                <div className="flex items-center gap-4 py-5" key={index}>
                    <Skeleton className="h-9 w-9 rounded-lg" />
                    <div className="flex-1">
                        <Skeleton className="h-4 w-28" />
                        <Skeleton className="mt-1.5 h-2.5 w-20" />
                    </div>
                    <Skeleton className="h-7 w-20 rounded-lg" />
                </div>
            ))}
        </div>
    );
}

function NoProviders() {
    return (
        <div className="-mt-12 flex flex-1 items-center justify-center">
            <div className="flex max-w-md flex-col items-center text-center">
                <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.08] bg-white/[0.04]">
                    <IconPlugConnected className="text-muted-foreground/20" size={28} />
                </div>
                <h2 className="font-heading text-foreground/70 mb-2 text-lg tracking-wide">
                    No providers detected
                </h2>
                <p className="text-muted-foreground/40 mb-6 text-sm leading-relaxed">
                    Agents need an AI provider to think. Add an API key or sign in with a
                    subscription.
                </p>
                <div className="text-muted-foreground/25 flex items-center gap-2 font-mono text-[11px]">
                    <IconTerminal2 size={13} />
                    <span>export ANTHROPIC_API_KEY=sk-...</span>
                </div>
            </div>
        </div>
    );
}

function Failure({ message }: { message: string }) {
    return (
        <div className="flex items-start gap-2 rounded-lg border border-red-500/15 bg-red-500/10 px-4 py-3">
            <IconAlertTriangle className="mt-0.5 shrink-0 text-red-400/60" size={14} />
            <p className="font-mono text-xs text-red-400/70">{message}</p>
        </div>
    );
}

export default function ProvidersPage() {
    usePageTitle('Settings');
    const { feedback, show } = useFeedback();
    const { check, checking, failure, loading, providers, reconnect, reset } = useProviders(show);
    const [configuring, setConfiguring] = useState<null | string>(null);

    return (
        <Page>
            <PageHeader description="Connect your AI providers to power agents." title="Settings" />
            <FeedbackLine feedback={feedback} />
            {!loading && failure !== '' && <Failure message={failure} />}
            {loading && <Loading />}
            {!loading && providers.length === 0 && failure === '' && <NoProviders />}
            {!loading && providers.length > 0 && (
                <div className="divide-y divide-white/[0.06]">
                    {providers.map((provider) => (
                        <ProviderRow
                            checking={checking === provider.provider}
                            key={provider.provider}
                            onCheck={inBackground(async () => {
                                await check(provider.provider);
                            })}
                            onConfigure={() => {
                                setConfiguring(provider.provider);
                            }}
                            onReconnect={inBackground(async () => {
                                await reconnect(provider.provider);
                            })}
                            onReset={inBackground(async () => {
                                await reset(provider.provider);
                            })}
                            provider={provider}
                        />
                    ))}
                </div>
            )}
            {configuring !== null && (
                <ConfigureModal
                    onClose={() => {
                        setConfiguring(null);
                    }}
                    provider={configuring}
                />
            )}
        </Page>
    );
}
