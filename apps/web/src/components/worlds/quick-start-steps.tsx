'use client';

import { IconArrowRight, IconRocket } from '@tabler/icons-react';
import { useId } from 'react';

import { SPAWN_PROGRESS } from '../../api/worlds';
import { useProgressMessages } from '../../hooks/use-progress-messages';
import { focusOnMount } from '../focus-on-mount';
import { ProgressShimmer } from '../progress-shimmer';

const LABEL = 'text-muted-foreground/40 mb-2 block text-[10px] tracking-widest uppercase';
const FIELD =
    'text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-4 py-3 text-sm transition-colors focus:border-white/[0.15] focus:outline-none';
const PRIMARY =
    'text-foreground/70 flex w-full items-center justify-center gap-2 rounded-xl border border-white/[0.08] bg-white/[0.06] py-3 text-sm font-medium transition-all hover:bg-white/[0.1] disabled:cursor-not-allowed disabled:opacity-30';
const SPINNER =
    'border-foreground/30 border-t-foreground/70 h-3.5 w-3.5 animate-spin rounded-full border-2';

interface NameStepProps {
    name: string;
    working: boolean;
    onName: (name: string) => void;
    onCreate: () => void;
}

export function NameStep({ name, working, onName, onCreate }: NameStepProps) {
    const id = useId();
    return (
        <>
            <div>
                <label className={LABEL} htmlFor={id}>
                    Agent name
                </label>
                <input
                    className={FIELD}
                    id={id}
                    onChange={(event) => {
                        onName(event.target.value);
                    }}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter') {
                            onCreate();
                        }
                    }}
                    placeholder="e.g. atlas, neo, morpheus..."
                    ref={focusOnMount}
                    value={name}
                />
                <p className="text-muted-foreground/25 mt-2 text-[10px]">
                    Agents are autonomous AI entities that work inside worlds
                </p>
            </div>
            <button
                className={PRIMARY}
                disabled={name.trim() === '' || working}
                onClick={onCreate}
                type="button"
            >
                {working ? <div className={SPINNER} /> : <IconArrowRight size={16} />}
                {working ? 'Creating...' : 'Create Agent'}
            </button>
        </>
    );
}

interface PurposeStepProps {
    name: string;
    purpose: string;
    onPurpose: (purpose: string) => void;
    onNext: () => void;
}

export function PurposeStep({ name, purpose, onPurpose, onNext }: PurposeStepProps) {
    const id = useId();
    return (
        <>
            <div>
                <label className={LABEL} htmlFor={id}>
                    What should {name} do?
                </label>
                <textarea
                    className={`${FIELD} resize-none`}
                    id={id}
                    onChange={(event) => {
                        onPurpose(event.target.value);
                    }}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter' && !event.shiftKey) {
                            event.preventDefault();
                            onNext();
                        }
                    }}
                    placeholder="e.g. Build a REST API, Manage my infrastructure, Write documentation..."
                    ref={focusOnMount}
                    rows={3}
                    value={purpose}
                />
                <p className="text-muted-foreground/25 mt-2 text-[10px]">
                    Optional - you can always change this later
                </p>
            </div>
            <button className={PRIMARY} onClick={onNext} type="button">
                <IconArrowRight size={16} />
                {purpose.trim() === '' ? 'Skip for now' : 'Continue'}
            </button>
        </>
    );
}

interface WorldStepProps {
    name: string;
    purpose: string;
    workspace: string;
    working: boolean;
    onWorkspace: (workspace: string) => void;
    onSpawn: () => void;
}

function Summary({
    name,
    purpose,
    workspace,
}: Pick<WorldStepProps, 'name' | 'purpose' | 'workspace'>) {
    return (
        <div className="rounded-lg border border-white/[0.05] bg-white/[0.02] px-3 py-3">
            <p className="text-muted-foreground/30 mb-1 text-[10px] tracking-widest uppercase">
                Summary
            </p>
            <div className="text-muted-foreground/40 space-y-0.5 text-[11px]">
                <p>
                    → Agent: <span className="text-foreground/60 font-mono">{name}</span>
                </p>
                {purpose !== '' && (
                    <p>
                        → Purpose: <span className="text-foreground/60">{purpose}</span>
                    </p>
                )}
                <p>
                    → Workspace:{' '}
                    <span className="text-foreground/60 font-mono">
                        {workspace === '' ? `/tmp/spwn-${name.trim()}` : workspace}
                    </span>
                </p>
            </div>
        </div>
    );
}

export function WorldStep({
    name,
    purpose,
    workspace,
    working,
    onWorkspace,
    onSpawn,
}: WorldStepProps) {
    const id = useId();
    const progress = useProgressMessages(working, SPAWN_PROGRESS);
    return (
        <>
            <div>
                <label className={LABEL} htmlFor={id}>
                    Workspace path
                </label>
                <input
                    className={`${FIELD} font-mono`}
                    id={id}
                    onChange={(event) => {
                        onWorkspace(event.target.value);
                    }}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter') {
                            onSpawn();
                        }
                    }}
                    placeholder={`/tmp/spwn-${name.trim() || 'agent'}`}
                    ref={focusOnMount}
                    value={workspace}
                />
                <p className="text-muted-foreground/25 mt-2 text-[10px]">
                    The directory where {name} will work - leave empty for default
                </p>
            </div>
            <Summary name={name} purpose={purpose} workspace={workspace} />
            <button
                className={`${PRIMARY} ${working ? 'animate-pulse' : ''}`}
                disabled={working}
                onClick={onSpawn}
                type="button"
            >
                {working ? <div className={SPINNER} /> : <IconRocket size={16} />}
                {working ? 'Spawning...' : 'New World'}
            </button>
            <ProgressShimmer active={working} message={progress} />
        </>
    );
}
