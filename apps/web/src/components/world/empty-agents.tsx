'use client';

import {
    IconArrowRight,
    IconLoader2,
    IconSparkles,
    IconUserPlus,
    IconWorld,
} from '@tabler/icons-react';
import { useCallback, useState } from 'react';

import { apiGet, apiPost } from '../../api/client';
import { errorMessageOf } from '../../api/worlds';
import { inBackground } from '../../events/in-background';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { useList } from '../../hooks/use-list';
import { Skeleton } from '../ui/skeleton';
import type { GalleryExample } from '../worlds/gallery-card';

const CARD =
    'group flex rounded-lg border border-white/[0.08] bg-white/[0.02] px-3 py-2.5 text-left transition-colors hover:border-white/[0.16] hover:bg-white/[0.05] disabled:cursor-not-allowed disabled:opacity-50';

function useExamples() {
    const [examples, setExamples] = useState<GalleryExample[] | null>(null);
    const load = useCallback(async () => {
        try {
            const gallery = await apiGet<{ examples?: GalleryExample[] }>('/api/examples');
            setExamples(gallery.examples ?? []);
        } catch {
            setExamples([]);
        }
    }, []);
    useBackgroundTask(load);
    return examples;
}

function Spinner({ size }: { size: number }) {
    return <IconLoader2 className="text-muted-foreground/60 shrink-0 animate-spin" size={size} />;
}

function InstalledAgents({
    names,
    busy,
    onDeploy,
}: {
    names: string[];
    busy: null | string;
    onDeploy: (name: string) => void;
}) {
    if (names.length === 0) {
        return (
            <div className="text-muted-foreground/60 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5 text-[11px]">
                You don&apos;t have any agents installed yet. Pick one from the gallery below to
                install and deploy in one click.
            </div>
        );
    }
    return (
        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {names.map((name) => (
                <button
                    className={`${CARD} items-center justify-between`}
                    disabled={busy !== null}
                    key={name}
                    onClick={() => {
                        onDeploy(name);
                    }}
                    type="button"
                >
                    <div className="min-w-0">
                        <div className="text-foreground/90 truncate font-mono text-[12px]">
                            {name}
                        </div>
                        <div className="text-muted-foreground/40 text-[10px] tracking-wider uppercase">
                            worker
                        </div>
                    </div>
                    {busy === name ? (
                        <Spinner size={13} />
                    ) : (
                        <span className="text-muted-foreground/40 group-hover:text-foreground/80 inline-flex shrink-0 items-center gap-1 text-[10px] tracking-wider uppercase">
                            Deploy
                            <IconArrowRight size={11} />
                        </span>
                    )}
                </button>
            ))}
        </div>
    );
}

function TemplateGallery({
    examples,
    busy,
    onInstall,
}: {
    examples: GalleryExample[];
    busy: null | string;
    onInstall: (example: GalleryExample) => void;
}) {
    if (examples.length === 0) {
        return (
            <p className="text-muted-foreground/60 col-span-full text-[11px]">
                No examples bundled in this build.
            </p>
        );
    }
    return (
        <>
            {examples.map((example) => (
                <button
                    className={`${CARD} items-start gap-2.5`}
                    disabled={busy !== null}
                    key={example.slug}
                    onClick={() => {
                        onInstall(example);
                    }}
                    type="button"
                >
                    <IconWorld className="mt-0.5 shrink-0 text-blue-400/70" size={14} />
                    <div className="min-w-0 flex-1">
                        <div className="text-foreground/95 truncate text-[12px] font-medium">
                            {example.name}
                        </div>
                        <div className="text-muted-foreground/60 truncate text-[10px]">
                            {example.tagline}
                        </div>
                        <div className="mt-1 flex flex-wrap gap-1">
                            {example.agents.slice(0, 3).map((agent) => (
                                <span
                                    className="text-muted-foreground/60 rounded border border-white/[0.06] bg-white/[0.03] px-1 py-0.5 font-mono text-[9px]"
                                    key={agent}
                                >
                                    {agent}
                                </span>
                            ))}
                        </div>
                    </div>
                    {busy === example.slug && <Spinner size={12} />}
                </button>
            ))}
        </>
    );
}

function useDeployHere(worldId: string, onDeployed: () => void) {
    const [busy, setBusy] = useState<null | string>(null);
    const [problem, setProblem] = useState('');

    const run = async (key: string, task: () => Promise<void>) => {
        setBusy(key);
        setProblem('');
        try {
            await task();
            onDeployed();
        } catch (error) {
            setProblem(errorMessageOf(error, 'deploy failed'));
        }
        setBusy(null);
    };
    const deploy = async (name: string) => {
        await run(name, async () => {
            await apiPost(`/api/worlds/${worldId}/agents`, { name, role: 'worker' });
        });
    };
    const install = async (example: GalleryExample) => {
        await run(example.slug, async () => {
            const [primary] = example.agents;
            if (primary === undefined || primary === '') {
                throw new Error('template has no agents');
            }
            await apiPost(`/api/examples/${example.slug}/install`);
            await apiPost(`/api/worlds/${worldId}/agents`, { name: primary, role: 'worker' });
        });
    };
    return { busy, deploy, install, problem };
}

function galleryLabel(open: boolean, noneInstalled: boolean): string {
    if (open) {
        return 'Hide template gallery';
    }
    return noneInstalled ? 'Install one from a template' : 'Or install a new one from a template';
}

/** A world with no agent yet: deploy one of the project's agents, or install one from a template, without restarting the world. */
export function EmptyAgents({ worldId, onDeployed }: { worldId: string; onDeployed: () => void }) {
    const installed = useList<{ name: string }>('/api/agents').map((agent) => agent.name);
    const examples = useExamples();
    const { busy, deploy, install, problem } = useDeployHere(worldId, onDeployed);
    const [galleryOpen, setGalleryOpen] = useState(false);

    return (
        <div className="rounded-2xl border border-white/[0.06] bg-white/[0.01] px-5 py-6">
            <div className="mb-4 flex items-start gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-white/[0.08] bg-gradient-to-br from-blue-500/10 to-purple-500/10">
                    <IconUserPlus className="text-blue-400/80" size={16} />
                </div>
                <div className="min-w-0">
                    <h3 className="text-foreground/95 text-sm font-medium">This world is empty</h3>
                    <p className="text-muted-foreground/60 mt-0.5 text-[11px]">
                        Pick one of your agents to deploy here, or install a fresh one from the
                        gallery. Deployment is hot - no container restart.
                    </p>
                </div>
            </div>
            {problem !== '' && <p className="mb-3 text-[11px] text-red-300/80">{problem}</p>}
            {examples === null ? (
                <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                    {[0, 1, 2].map((index) => (
                        <Skeleton className="h-16 rounded-lg" key={index} />
                    ))}
                </div>
            ) : (
                <InstalledAgents busy={busy} names={installed} onDeploy={inBackground(deploy)} />
            )}
            <div className="mt-5">
                <button
                    className="text-muted-foreground/50 hover:text-foreground/80 inline-flex items-center gap-1.5 text-[11px] tracking-wider uppercase transition-colors"
                    onClick={() => {
                        setGalleryOpen(!galleryOpen);
                    }}
                    type="button"
                >
                    <IconSparkles size={11} />
                    {galleryLabel(galleryOpen, installed.length === 0)}
                </button>
                {galleryOpen && (
                    <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                        <TemplateGallery
                            busy={busy}
                            examples={examples ?? []}
                            onInstall={inBackground(install)}
                        />
                    </div>
                )}
            </div>
        </div>
    );
}
