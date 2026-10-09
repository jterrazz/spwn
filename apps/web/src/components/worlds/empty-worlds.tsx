'use client';

import { IconRocket, IconSparkles, IconTerminal2 } from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useCallback, useState } from 'react';

import { apiGet, apiPost } from '../../api/client';
import { errorMessageOf, spawnWorld } from '../../api/worlds';
import { inBackground } from '../../events/in-background';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { Skeleton } from '../ui/skeleton';
import { GalleryCard } from './gallery-card';
import type { GalleryExample } from './gallery-card';

interface EmptyWorldsProps {
    agentCount: number;
    onSpawn: () => void;
    onRefetch: () => void;
}

function spawnRequestOf(example: GalleryExample): Record<string, unknown> {
    const [only] = example.agents;
    if (example.agents.length === 1 && only !== undefined) {
        return { agent: only, config: example.worlds[0] };
    }
    if (example.agents.length > 1) {
        return {
            agents: example.agents.map((name) => ({ name, role: 'worker' })),
            config: example.worlds[0],
        };
    }
    return { config: example.worlds[0] };
}

function useGallery() {
    const [gallery, setGallery] = useState<GalleryExample[] | null>(null);
    const load = useCallback(async () => {
        try {
            const { examples } = await apiGet<{ examples?: GalleryExample[] }>('/api/examples');
            setGallery(examples ?? []);
        } catch {
            setGallery([]);
        }
    }, []);
    useBackgroundTask(load);
    return gallery;
}

function Intro({ agentCount }: { agentCount: number }) {
    return (
        <div className="mb-10 text-center">
            <div className="text-muted-foreground/70 mb-4 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 text-[10px] tracking-wider uppercase">
                <IconSparkles size={11} />
                Start from a template
            </div>
            <h2 className="font-heading text-foreground/90 text-2xl tracking-wide">
                {agentCount > 0
                    ? 'Give your agents a world to work in'
                    : 'Pick a template and spawn in one click'}
            </h2>
            <p className="text-muted-foreground/60 mx-auto mt-2 max-w-lg text-sm">
                {agentCount > 0
                    ? `You have ${agentCount} agent${agentCount > 1 ? 's' : ''} installed. Pick a template to put one to work, or build your own world from scratch.`
                    : 'Each template ships a full world config + pre-written agents with profiles. Clicking Install & spawn copies the files into ~/.spwn, creates a container and drops you straight into a conversation.'}
            </p>
        </div>
    );
}

function Gallery({
    gallery,
    installing,
    onInstall,
}: {
    gallery: GalleryExample[] | null;
    installing: null | string;
    onInstall: (example: GalleryExample) => void;
}) {
    if (gallery === null) {
        return (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {[0, 1, 2, 3, 4].map((index) => (
                    <Skeleton className="h-52 rounded-2xl" key={index} />
                ))}
            </div>
        );
    }
    if (gallery.length === 0) {
        return (
            <p className="text-muted-foreground/60 text-center text-sm">
                No examples bundled in this build.
            </p>
        );
    }
    return (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {gallery.map((example, index) => (
                <GalleryCard
                    busy={installing === example.slug}
                    disabled={installing !== null && installing !== example.slug}
                    example={example}
                    featured={index === 0}
                    key={example.slug}
                    onInstall={() => {
                        onInstall(example);
                    }}
                />
            ))}
        </div>
    );
}

/** The worlds page with no world yet: the gallery of bundled examples, each one click from a running world. */
export function EmptyWorlds({ agentCount, onSpawn, onRefetch }: EmptyWorldsProps) {
    const router = useRouter();
    const gallery = useGallery();
    const [installing, setInstalling] = useState<null | string>(null);
    const [installError, setInstallError] = useState('');

    const install = async (example: GalleryExample) => {
        setInstalling(example.slug);
        setInstallError('');
        try {
            await apiPost(`/api/examples/${example.slug}/install`);
            const id = await spawnWorld(spawnRequestOf(example));
            onRefetch();
            if (id !== '') {
                router.push(`/world/${id}`);
            }
        } catch (error) {
            setInstallError(errorMessageOf(error, 'install failed'));
            setInstalling(null);
        }
    };

    return (
        <div className="flex min-h-[400px] flex-1 items-start justify-center px-4 pt-12 pb-16">
            <div className="w-full max-w-5xl">
                <Intro agentCount={agentCount} />
                <Gallery
                    gallery={gallery}
                    installing={installing}
                    onInstall={inBackground(install)}
                />
                {installError !== '' && (
                    <p className="mt-4 text-center text-xs text-red-300/80">{installError}</p>
                )}
                <div className="mt-10 flex flex-col items-center gap-2">
                    <button
                        className="text-muted-foreground/50 hover:text-foreground/80 inline-flex items-center gap-2 text-[11px] tracking-wider uppercase transition-colors"
                        onClick={onSpawn}
                        type="button"
                    >
                        <IconRocket size={12} />
                        Or build your own world from scratch
                    </button>
                    <div className="text-muted-foreground/30 flex items-center gap-2 font-mono text-[10px]">
                        <IconTerminal2 size={11} />
                        <span>spwn example list</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
