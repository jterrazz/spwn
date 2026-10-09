'use client';

import { useRouter } from 'next/navigation';

import { SidebarGroup } from '../ui/sidebar';

const STEPS = [
    { after: 'and connect a provider', before: 'Go to', href: '/providers', link: 'Settings' },
    { after: '', before: 'Create an', href: '/agents', link: 'Agent' },
    { after: '', before: 'Spawn a', href: '/', link: 'World' },
];

/** The three steps the sidebar offers while no world exists. */
export function GettingStarted() {
    const router = useRouter();
    return (
        <SidebarGroup>
            <div className="mx-2 space-y-2 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-3">
                <p className="text-muted-foreground/30 text-[10px] tracking-widest uppercase">
                    Getting started
                </p>
                <div className="text-muted-foreground/40 space-y-1.5 text-[11px] leading-relaxed">
                    {STEPS.map((step, index) => (
                        <p key={step.href}>
                            <span className="text-foreground/50 font-mono">{index + 1}.</span>{' '}
                            {step.before}{' '}
                            <button
                                className="text-foreground/60 hover:text-foreground/80 underline decoration-white/10 underline-offset-2"
                                onClick={() => {
                                    router.push(step.href);
                                }}
                                type="button"
                            >
                                {step.link}
                            </button>
                            {step.after === '' ? '' : ` ${step.after}`}
                        </p>
                    ))}
                </div>
            </div>
        </SidebarGroup>
    );
}
