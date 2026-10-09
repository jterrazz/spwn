'use client';

import { IconCheck } from '@tabler/icons-react';

import { AuthStep } from './auth-step';
import { ChatStep, DockerStep, WorldStep } from './step-views';
import { completed } from './steps';
import type { OnboardingStatus, StepId, STEPS } from './steps';

interface StepRowProps {
    step: (typeof STEPS)[number];
    index: number;
    status: null | OnboardingStatus;
    open: boolean;
    refreshing: boolean;
    onOpen: () => void;
    onAdvance: () => void;
    onRefresh: () => void;
}

function tones(done: boolean, open: boolean): { row: string; circle: string; label: string } {
    if (open) {
        return {
            circle: 'border-white/20 bg-white/[0.06] text-foreground/80',
            label: 'Active',
            row: 'border-white/15 bg-white/[0.04]',
        };
    }
    if (done) {
        return {
            circle: 'border-emerald-400/40 bg-emerald-500/15 text-emerald-300',
            label: 'Done',
            row: 'border-emerald-500/20 bg-emerald-500/[0.03]',
        };
    }
    return {
        circle: 'border-white/10 bg-white/[0.02] text-muted-foreground/50',
        label: 'Pending',
        row: 'border-white/[0.06] bg-white/[0.015]',
    };
}

function StepBody({
    id,
    status,
    refreshing,
    onAdvance,
    onRefresh,
}: Omit<StepRowProps, 'index' | 'onOpen' | 'open' | 'step'> & { id: StepId }) {
    switch (id) {
        case 'docker': {
            return (
                <DockerStep
                    onAdvance={onAdvance}
                    onRefresh={onRefresh}
                    refreshing={refreshing}
                    status={status}
                />
            );
        }
        case 'auth': {
            return <AuthStep onAdvance={onAdvance} status={status} />;
        }
        case 'world': {
            return <WorldStep onAdvance={onAdvance} status={status} />;
        }
        default: {
            return <ChatStep />;
        }
    }
}

/** One step of the first-run setup: its number, title and state, and its body while it is open. */
export function StepRow({
    step,
    index,
    status,
    open,
    refreshing,
    onOpen,
    onAdvance,
    onRefresh,
}: StepRowProps) {
    const done = status !== null && completed(step.id, status);
    const tone = tones(done && !open, open);
    const Icon = step.icon;
    return (
        <li className={`overflow-hidden rounded-xl border transition-colors ${tone.row}`}>
            <button
                className="flex w-full items-center gap-4 px-4 py-3 text-left"
                onClick={onOpen}
                type="button"
            >
                <div
                    className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full border ${done ? 'border-emerald-400/40 bg-emerald-500/15 text-emerald-300' : tone.circle}`}
                >
                    {done ? <IconCheck size={14} /> : <Icon size={14} />}
                </div>
                <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                        <span className="text-muted-foreground/40 text-[10px] tabular-nums">
                            {String(index + 1).padStart(2, '0')}
                        </span>
                        <span
                            className={`text-sm font-medium ${done ? 'text-emerald-200/90' : 'text-foreground/90'}`}
                        >
                            {step.title}
                        </span>
                    </div>
                    <p className="text-muted-foreground/60 mt-0.5 text-xs">{step.subtitle}</p>
                </div>
                <span className="text-muted-foreground/40 text-[10px] tracking-wider uppercase">
                    {done ? 'Done' : tone.label}
                </span>
            </button>
            {open && (
                <div className="border-t border-white/[0.06] px-4 py-4">
                    <StepBody
                        id={step.id}
                        onAdvance={onAdvance}
                        onRefresh={onRefresh}
                        refreshing={refreshing}
                        status={status}
                    />
                </div>
            )}
        </li>
    );
}
