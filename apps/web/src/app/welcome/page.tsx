'use client';

import { IconCheck, IconLoader2, IconRocket } from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { apiGet, apiPost } from '../../api/client';
import { optional } from '../../api/optional';
import { StepRow } from '../../components/welcome/step-row';
import { nextStep, stepAfter, STEPS } from '../../components/welcome/steps';
import type { OnboardingStatus, StepId } from '../../components/welcome/steps';
import { inBackground } from '../../events/in-background';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { usePageTitle } from '../../hooks/use-page-title';

function useOnboarding() {
    const router = useRouter();
    const [status, setStatus] = useState<null | OnboardingStatus>(null);
    const [open, setOpen] = useState<StepId>('docker');
    const [refreshing, setRefreshing] = useState(false);

    const fetchStatus = useCallback(async () => {
        const next = await optional(apiGet<OnboardingStatus>('/api/system/onboarding'));
        if (next !== undefined) {
            setStatus(next);
            setOpen((previous) => stepAfter(previous, next));
        }
    }, []);
    useBackgroundTask(fetchStatus, 4000);

    useEffect(() => {
        if (status?.completed === true) {
            router.replace('/');
        }
    }, [status?.completed, router]);

    const refresh = async () => {
        setRefreshing(true);
        await fetchStatus();
        setRefreshing(false);
    };
    return { open, refresh, refreshing, setOpen, status };
}

function Intro() {
    return (
        <header className="mb-10 text-center">
            <div className="text-muted-foreground/70 mb-3 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 text-[10px] tracking-wider uppercase">
                <IconRocket size={11} />
                First-run setup
            </div>
            <h1 className="font-heading text-foreground/90 text-3xl tracking-wide">
                Welcome to spwn
            </h1>
            <p className="text-muted-foreground/70 mx-auto mt-2 max-w-lg text-sm">
                The control plane for AI agents. Four quick steps and you&apos;re ready to spawn
                your first world.
            </p>
        </header>
    );
}

function Footer({ ready, onSkip }: { ready: boolean; onSkip: () => void }) {
    const router = useRouter();
    const [finishing, setFinishing] = useState(false);
    const finish = async () => {
        setFinishing(true);
        try {
            await apiPost('/api/system/onboarding/complete');
            router.replace('/');
        } finally {
            setFinishing(false);
        }
    };
    return (
        <div className="mt-8 flex items-center justify-between border-t border-white/[0.06] pt-6">
            <button
                className="text-muted-foreground/60 hover:text-foreground/80 text-xs"
                onClick={onSkip}
                type="button"
            >
                Skip for now
            </button>
            <button
                className="disabled:text-muted-foreground/40 inline-flex items-center gap-2 rounded-lg border border-emerald-400/30 bg-emerald-500/10 px-4 py-2 text-xs font-medium text-emerald-100 transition-colors hover:bg-emerald-500/20 disabled:cursor-not-allowed disabled:border-white/10 disabled:bg-white/[0.02]"
                disabled={!ready || finishing}
                onClick={inBackground(finish)}
                type="button"
            >
                {finishing ? (
                    <IconLoader2 className="animate-spin" size={12} />
                ) : (
                    <IconCheck size={12} />
                )}
                Finish setup
            </button>
        </div>
    );
}

export default function WelcomePage() {
    usePageTitle('Welcome');
    const router = useRouter();
    const { open, refresh, refreshing, setOpen, status } = useOnboarding();
    const ready = status?.hasDocker === true && status.hasAuth && status.hasWorlds;

    return (
        <div className="min-h-full overflow-y-auto px-6 py-10">
            <div className="mx-auto max-w-3xl">
                <Intro />
                <ol className="space-y-3">
                    {STEPS.map((step, index) => (
                        <StepRow
                            index={index}
                            key={step.id}
                            onAdvance={() => {
                                setOpen(nextStep(step.id));
                            }}
                            onOpen={() => {
                                setOpen(step.id);
                            }}
                            onRefresh={inBackground(refresh)}
                            open={open === step.id}
                            refreshing={refreshing}
                            status={status}
                            step={step}
                        />
                    ))}
                </ol>
                <Footer
                    onSkip={() => {
                        router.replace('/');
                    }}
                    ready={ready}
                />
            </div>
        </div>
    );
}
