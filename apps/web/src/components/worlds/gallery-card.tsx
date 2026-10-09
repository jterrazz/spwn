'use client';

import {
    IconArrowRight,
    IconBriefcase,
    IconBuildingFactory2,
    IconFlask,
    IconLoader2,
    IconRobot,
    IconRocket,
    IconUser,
    IconUsers,
    IconWorld,
} from '@tabler/icons-react';

export interface GalleryExample {
    slug: string;
    name: string;
    tagline: string;
    description: string;
    agents: string[];
    worlds: string[];
    command?: string;
}

interface Theme {
    icon: React.ReactNode;
    accent: string;
    gradient: string;
}

const THEMES: Record<string, Theme> = {
    macrohard: {
        accent: 'text-sky-400/80',
        gradient: 'from-sky-500/15 to-indigo-500/10',
        icon: <IconUsers size={18} />,
    },
    matrix: {
        accent: 'text-green-400/80',
        gradient: 'from-green-500/15 to-emerald-500/10',
        icon: <IconRobot size={18} />,
    },
    'paperclip-factory': {
        accent: 'text-blue-400/80',
        gradient: 'from-blue-500/15 to-cyan-500/10',
        icon: <IconBuildingFactory2 size={18} />,
    },
    'research-lab': {
        accent: 'text-purple-400/80',
        gradient: 'from-purple-500/15 to-pink-500/10',
        icon: <IconFlask size={18} />,
    },
    startup: {
        accent: 'text-amber-400/80',
        gradient: 'from-amber-500/15 to-orange-500/10',
        icon: <IconBriefcase size={18} />,
    },
};

const PLAIN: Theme = {
    accent: 'text-blue-400/80',
    gradient: 'from-blue-500/10 to-purple-500/10',
    icon: <IconWorld size={18} />,
};

const FEATURED_BUTTON =
    'text-foreground/95 border-white/[0.15] bg-white/[0.08] hover:border-white/[0.25] hover:bg-white/[0.14]';
const BUTTON =
    'text-foreground/90 border-white/[0.10] bg-white/[0.06] hover:border-white/[0.18] hover:bg-white/[0.10]';

interface GalleryCardProps {
    example: GalleryExample;
    featured: boolean;
    busy: boolean;
    disabled: boolean;
    onInstall: () => void;
}

function CardHeading({ example }: { example: GalleryExample }) {
    const theme = THEMES[example.slug] ?? PLAIN;
    return (
        <div className="flex items-start gap-3">
            <div
                className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-white/[0.08] bg-gradient-to-br ${theme.gradient}`}
            >
                <span className={theme.accent}>{theme.icon}</span>
            </div>
            <div className="min-w-0 flex-1">
                <h3 className="font-heading text-foreground/95 text-sm tracking-wide">
                    {example.name}
                </h3>
                <p className="text-muted-foreground/60 text-[11px]">{example.tagline}</p>
            </div>
        </div>
    );
}

function CardRoster({ example }: { example: GalleryExample }) {
    const command = example.command?.split('\n')[0] ?? '';
    return (
        <>
            <div className="mt-3 flex flex-wrap gap-1.5">
                {example.agents.map((agent) => (
                    <span
                        className="text-muted-foreground/70 inline-flex items-center gap-1 rounded-full border border-white/[0.08] bg-white/[0.04] px-2 py-0.5 font-mono text-[10px]"
                        key={agent}
                    >
                        <IconUser className="opacity-50" size={9} />
                        {agent}
                    </span>
                ))}
            </div>
            {command !== '' && (
                <div className="mt-3 rounded-lg border border-white/[0.06] bg-white/[0.03] px-3 py-1.5">
                    <code className="text-muted-foreground/50 font-mono text-[10px] leading-relaxed">
                        $ {command}
                    </code>
                </div>
            )}
        </>
    );
}

function InstallLabel({ busy }: { busy: boolean }) {
    if (busy) {
        return (
            <>
                <IconLoader2 className="animate-spin" size={13} />
                Spawning…
            </>
        );
    }
    return (
        <>
            <IconRocket size={13} />
            Install &amp; spawn
            <IconArrowRight className="ml-0.5 opacity-60" size={12} />
        </>
    );
}

/** One bundled example of the gallery: what it is, who it ships, and the button that installs and spawns it. */
export function GalleryCard({ example, featured, busy, disabled, onInstall }: GalleryCardProps) {
    const firstParagraph = example.description.split('\n\n')[0] ?? example.description;
    const span = featured ? 'sm:col-span-2 lg:col-span-2' : '';
    const hover = disabled
        ? 'opacity-50'
        : 'hover:border-white/[0.15] hover:bg-white/[0.04] hover:shadow-lg hover:shadow-white/[0.02]';

    return (
        <div
            className={`group relative flex h-full flex-col rounded-2xl border border-white/[0.08] bg-white/[0.02] p-5 transition-all duration-200 ${span} ${hover}`}
        >
            <CardHeading example={example} />
            <p
                className={`text-muted-foreground/70 mt-3 text-[11px] leading-relaxed ${featured ? 'line-clamp-4' : 'line-clamp-3'}`}
            >
                {firstParagraph}
            </p>
            <CardRoster example={example} />
            <div className="flex-1" />
            <button
                className={`mt-4 inline-flex items-center justify-center gap-1.5 rounded-lg border px-3 py-2 text-xs font-medium transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-60 ${featured ? FEATURED_BUTTON : BUTTON}`}
                disabled={disabled || busy}
                onClick={onInstall}
                type="button"
            >
                <InstallLabel busy={busy} />
            </button>
        </div>
    );
}
