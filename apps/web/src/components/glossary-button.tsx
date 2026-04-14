'use client';

import { IconHelpCircle } from '@tabler/icons-react';
import { useState } from 'react';

import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';

interface Term {
    word: string;
    short: string;
    long: string;
}

const TERMS: Term[] = [
    {
        word: 'Universe',
        short: 'The shared reality.',
        long: 'The set of physics, constants and resource limits applied to every world inside an organization. Configured in universe.yaml.',
    },
    {
        word: 'World',
        short: 'An isolated workspace.',
        long: 'A long-lived Docker container running one or more agents on a project. Worlds are created by you or by the architect.',
    },
    {
        word: 'Agent',
        short: 'A persistent worker.',
        long: 'An AI agent composed from tools, skills, and a profile, with persistent memory and identity. Lives inside a world and can run tasks, talk with you, and collaborate with other agents.',
    },
    {
        word: 'Architect',
        short: 'The always-on orchestrator.',
        long: 'A daemon that listens on channels (CLI, chat, etc.), creates and destroys worlds on demand, and coordinates agents across them.',
    },
    {
        word: 'Runtime',
        short: 'How an agent thinks.',
        long: 'The CLI or SDK that drives an agent - Claude Code, Codex, Aider, etc. Runtimes are swappable adapters; the rest of spwn does not care which one you use.',
    },
    {
        word: 'Provider',
        short: 'Where the model runs.',
        long: 'The company hosting the underlying LLM (Anthropic, OpenAI, …). Credentials are managed once and bind-mounted into every container.',
    },
    {
        word: 'Tool',
        short: 'A composable capability.',
        long: 'A piece of an image - a binary, configuration, or skill - that can be installed into a world. Tools live under @spwn/* in the catalog.',
    },
    {
        word: 'Skill',
        short: 'A reusable playbook.',
        long: 'A markdown file describing how to perform a task. Skills are bundled inside tools and copied into the world so any agent can read them.',
    },
    {
        word: 'Profile',
        short: 'A reusable personality template.',
        long: 'Role, tone, purpose, behavior - authored in markdown and attached to agents. Profiles are first-class composable blocks alongside tools and skills.',
    },
];

/**
 * Floating ? button + glossary modal. Mounted globally so users can ask
 * "what is an architect?" from any screen without leaving the page.
 */
export function GlossaryButton() {
    const [open, setOpen] = useState(false);
    return (
        <>
            <button
                aria-label="Open glossary"
                className="fixed bottom-4 left-4 z-50 inline-flex h-9 w-9 items-center justify-center rounded-full border border-white/10 bg-black/40 text-muted-foreground/70 backdrop-blur-md transition-colors hover:border-white/20 hover:text-foreground"
                onClick={() => setOpen(true)}
                title="Glossary"
                type="button"
            >
                <IconHelpCircle size={16} />
            </button>
            <Dialog onOpenChange={setOpen} open={open}>
                <DialogContent className="max-w-xl">
                    <DialogHeader>
                        <DialogTitle>Glossary</DialogTitle>
                        <DialogDescription>
                            The vocabulary spwn uses across the CLI and the desktop app.
                        </DialogDescription>
                    </DialogHeader>
                    <dl className="mt-2 max-h-[60vh] space-y-3 overflow-y-auto pr-2">
                        {TERMS.map((t) => (
                            <div
                                className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5"
                                key={t.word}
                            >
                                <dt className="flex items-baseline gap-2">
                                    <span className="text-sm font-medium text-foreground/95">
                                        {t.word}
                                    </span>
                                    <span className="text-[11px] text-muted-foreground/60">
                                        {t.short}
                                    </span>
                                </dt>
                                <dd className="mt-1 text-[11px] leading-relaxed text-muted-foreground/70">
                                    {t.long}
                                </dd>
                            </div>
                        ))}
                    </dl>
                </DialogContent>
            </Dialog>
        </>
    );
}
