import { BUTTON_VARIANTS } from './button';
import type { ButtonProps } from './button';
import type { Story } from './story';

const WORDS = {
    danger: 'Delete workspace…',
    destroy: 'Delete',
    inline: 'QR code',
    link: '3 more can wait',
    plain: 'Stop',
    primary: 'Open in Terminal',
} as const;

export const BUTTON_STORIES: readonly Story<ButtonProps>[] = [
    ...BUTTON_VARIANTS.map((variant) => ({
        name: variant,
        props: { children: WORDS[variant], variant },
    })),
    ...BUTTON_VARIANTS.map((variant) => ({
        name: `${variant}, disabled`,
        props: { children: WORDS[variant], disabled: true, variant },
    })),
    { name: 'plain, busy', props: { busy: true, children: 'Stop' } },
    { name: 'primary, busy', props: { busy: true, children: 'Create', variant: 'primary' } },
    { name: 'link, busy', props: { busy: true, children: 'Dismiss', variant: 'link' } },
    {
        name: 'a primary that yields',
        props: { children: 'New machine', variant: 'primary', yields: true },
    },
    { name: 'long text', props: { children: 'Free what can be rebuilt on every machine' } },
];
