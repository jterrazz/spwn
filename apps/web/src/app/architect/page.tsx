'use client';

import { IconPlayerPlay, IconPlayerStop, IconRefresh } from '@tabler/icons-react';

import { ActionButton } from '../../components/action-button';
import { OfflineView } from '../../components/architect/offline-view';
import { RunningView } from '../../components/architect/running-view';
import { StartingView } from '../../components/architect/starting-view';
import { useArchitectControl } from '../../components/architect/use-architect-control';
import type { ArchitectState } from '../../components/architect/use-architect-control';
import { Page } from '../../components/page';
import { PageHeader } from '../../components/page-header';
import { ProgressShimmer } from '../../components/progress-shimmer';
import { Skeleton } from '../../components/ui/skeleton';
import { useArchitectChat } from '../../contexts/architect-chat-context';
import { inBackground } from '../../events/in-background';
import { FeedbackLine, useFeedback } from '../../hooks/use-feedback';
import { usePageTitle } from '../../hooks/use-page-title';
import { useProgressMessages } from '../../hooks/use-progress-messages';

const DESCRIPTIONS: Record<ArchitectState, string> = {
    offline: 'Offline',
    running: 'Running. Talk to it in natural language.',
    starting: 'Starting...',
    stopping: 'Stopping...',
};

const STOP_PROGRESS = [
    { after: 0, text: 'Stopping architect...' },
    { after: 5, text: 'Shutting down container...' },
    { after: 15, text: 'Cleaning up...' },
];

function StoppingView() {
    const message = useProgressMessages(true, STOP_PROGRESS);
    return (
        <div className="-mt-12 flex flex-1 items-center justify-center">
            <div className="flex flex-col items-center text-center">
                <p className="text-muted-foreground/50 mb-4 text-sm">{message}</p>
                <div className="w-48">
                    <ProgressShimmer active message="" />
                </div>
            </div>
        </div>
    );
}

function LoadingView() {
    return (
        <div className="-mt-12 flex flex-1 items-center justify-center">
            <div className="flex flex-col items-center gap-4">
                <Skeleton className="h-16 w-16 rounded-2xl" />
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-56" />
            </div>
        </div>
    );
}

function HeaderActions({
    state,
    busy,
    onStart,
    onStop,
}: {
    state: ArchitectState;
    busy: boolean;
    onStart: () => void;
    onStop: () => void;
}) {
    if (state === 'offline') {
        return (
            <ActionButton
                compact
                disabled={busy}
                icon={<IconPlayerPlay size={16} stroke={2.2} />}
                label="Start"
                onClick={onStart}
            />
        );
    }
    if (state !== 'running') {
        return null;
    }
    return (
        <>
            <ActionButton
                compact
                disabled={busy}
                icon={<IconRefresh size={16} stroke={2.2} />}
                label="Restart"
                onClick={onStart}
            />
            <ActionButton
                compact
                danger
                disabled={busy}
                icon={<IconPlayerStop size={16} stroke={2.2} />}
                label="Stop"
                onClick={onStop}
            />
        </>
    );
}

export default function ArchitectPage() {
    usePageTitle('Architect');
    const { isRunning, loading, setArchitectStatus } = useArchitectChat();
    const { feedback, show } = useFeedback();
    const control = useArchitectControl({
        report: show,
        running: isRunning,
        setStatus: setArchitectStatus,
    });
    const { state } = control;

    return (
        <Page>
            <PageHeader
                actions={
                    <HeaderActions
                        busy={control.busy}
                        onStart={inBackground(control.start)}
                        onStop={inBackground(control.stop)}
                        state={state}
                    />
                }
                description={DESCRIPTIONS[state]}
                title="Architect"
            />
            <FeedbackLine feedback={feedback} />
            {state === 'offline' &&
                (loading ? (
                    <LoadingView />
                ) : (
                    <OfflineView disabled={control.busy} onStart={inBackground(control.start)} />
                ))}
            {state === 'starting' && <StartingView progress={control.progress} />}
            {state === 'stopping' && <StoppingView />}
            {state === 'running' && <RunningView />}
        </Page>
    );
}
