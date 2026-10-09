'use client';

import {
    IconArrowRight,
    IconBulb,
    IconCheck,
    IconSparkles,
    IconUser,
    IconWorld,
} from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { apiAction } from '../../api/client';
import { errorMessageOf, spawnWorld } from '../../api/worlds';
import { inBackground } from '../../events/in-background';
import { NameStep, PurposeStep, WorldStep } from './quick-start-steps';

const STEPS = [
    { icon: <IconUser size={14} />, label: 'Create Agent', number: 1 },
    { icon: <IconBulb size={14} />, label: 'Set Purpose', number: 2 },
    { icon: <IconWorld size={14} />, label: 'New World', number: 3 },
];

function stepTone(step: number, current: number): string {
    if (current > step) {
        return 'bg-green-500/15 text-green-400/80 border border-green-500/20';
    }
    return current === step
        ? 'bg-white/[0.08] text-foreground/70 border border-white/[0.12]'
        : 'bg-white/[0.02] text-muted-foreground/25 border border-white/[0.04]';
}

function Stepper({ current }: { current: number }) {
    return (
        <div className="mb-8 flex items-center justify-center gap-2">
            {STEPS.map((step, index) => (
                <div className="flex items-center gap-2" key={step.number}>
                    <div
                        className={`flex items-center gap-1.5 rounded-full px-3 py-1.5 font-mono text-[10px] transition-all ${stepTone(step.number, current)}`}
                    >
                        {current > step.number ? <IconCheck size={10} /> : step.icon}
                        {step.label}
                    </div>
                    {index < STEPS.length - 1 && (
                        <IconArrowRight className="text-muted-foreground/15" size={10} />
                    )}
                </div>
            ))}
        </div>
    );
}

function useQuickStart(onComplete: () => void) {
    const router = useRouter();
    const [step, setStep] = useState(1);
    const [name, setName] = useState('');
    const [workspace, setWorkspace] = useState('');
    const [error, setError] = useState('');
    const [working, setWorking] = useState(false);

    const createAgent = async () => {
        if (name.trim() === '') {
            return;
        }
        setWorking(true);
        setError('');
        const result = await apiAction('/api/agents', { name: name.trim() });
        setWorking(false);
        if (result.ok) {
            setStep(2);
        } else {
            setError(result.error ?? 'Failed to create agent');
        }
    };

    const spawn = async () => {
        setWorking(true);
        setError('');
        const path =
            workspace.trim() ||
            `/tmp/spwn-${name.trim()}-${Math.random().toString(36).slice(2, 6)}`;
        try {
            const id = await spawnWorld({
                agent: name.trim(),
                config: 'default',
                role: 'worker',
                workspaces: [{ name: 'default', path }],
            });
            onComplete();
            if (id !== '') {
                router.push(`/world/${id}`);
            }
        } catch (spawnError) {
            setError(errorMessageOf(spawnError, 'Failed to spawn world'));
            setWorking(false);
        }
    };

    return {
        createAgent,
        error,
        handleName: setName,
        handleWorkspace: setWorkspace,
        name,
        setStep,
        spawn,
        step,
        workspace,
        working,
    };
}

/** The first run: name an agent, give it a purpose, and spawn its first world. */
export function QuickStart({ onComplete }: { onComplete: () => void }) {
    const quickStart = useQuickStart(onComplete);
    const [purpose, setPurpose] = useState('');
    const { error, name, step } = quickStart;

    return (
        <div className="mx-auto w-full max-w-lg px-4">
            <div className="mb-8 text-center">
                <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.08] bg-gradient-to-br from-blue-500/20 to-purple-500/20">
                    <IconSparkles className="text-blue-400/60" size={28} />
                </div>
                <h2 className="font-heading text-foreground/90 text-xl">Get started</h2>
                <p className="text-muted-foreground/40 mt-1 font-mono text-xs">
                    Create an agent, give it a purpose, and spawn a world.
                </p>
            </div>
            <Stepper current={step} />
            <div className="glass-subtle space-y-4 rounded-2xl p-6">
                {step === 1 && (
                    <NameStep
                        name={name}
                        onCreate={inBackground(quickStart.createAgent)}
                        onName={quickStart.handleName}
                        working={quickStart.working}
                    />
                )}
                {step === 2 && (
                    <PurposeStep
                        name={name}
                        onNext={() => {
                            quickStart.setStep(3);
                        }}
                        onPurpose={setPurpose}
                        purpose={purpose}
                    />
                )}
                {step === 3 && (
                    <WorldStep
                        name={name}
                        onSpawn={inBackground(quickStart.spawn)}
                        onWorkspace={quickStart.handleWorkspace}
                        purpose={purpose}
                        working={quickStart.working}
                        workspace={quickStart.workspace}
                    />
                )}
                {error !== '' && (
                    <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 font-mono text-xs text-red-400">
                        {error}
                    </div>
                )}
            </div>
        </div>
    );
}
