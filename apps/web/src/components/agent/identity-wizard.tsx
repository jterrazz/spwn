'use client';

import { IconX } from '@tabler/icons-react';
import { useId } from 'react';

import { inBackground } from '../../events/in-background';

const LABEL = 'mb-1.5 block text-[10px] tracking-widest text-emerald-300/40 uppercase';
const FIELD =
    'text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-emerald-500/15 bg-white/[0.03] px-3 text-sm transition-colors focus:border-emerald-500/30 focus:outline-none';

interface IdentityWizardProps {
    agent: string;
    onSave: (field: string, content: string) => Promise<boolean>;
    onDismiss: () => void;
}

interface WizardFieldProps {
    label: string;
    placeholder: string;
    padding: string;
    onEnter: (value: string) => Promise<void>;
}

function WizardField({ label, placeholder, padding, onEnter }: WizardFieldProps) {
    const id = useId();
    return (
        <div>
            <label className={LABEL} htmlFor={id}>
                {label}
            </label>
            <input
                className={`${FIELD} ${padding}`}
                id={id}
                onKeyDown={inBackground(async (event: React.KeyboardEvent<HTMLInputElement>) => {
                    const value = event.currentTarget.value.trim();
                    if (event.key === 'Enter' && value !== '') {
                        await onEnter(value);
                    }
                })}
                placeholder={placeholder}
            />
        </div>
    );
}

function traitsOf(value: string): string {
    return value
        .split(',')
        .map((trait) => `- ${trait.trim()}`)
        .join('\n');
}

/** The banner of an agent with no purpose yet: its purpose, profile and traits, each saved on Enter. */
export function IdentityWizard({ agent, onSave, onDismiss }: IdentityWizardProps) {
    return (
        <div className="animate-in fade-in slide-in-from-top-2 rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-5 duration-300">
            <div className="mb-3 flex items-start justify-between">
                <div>
                    <h3 className="font-heading text-sm text-emerald-300">
                        Set up {agent}&apos;s identity
                    </h3>
                    <p className="mt-1 text-[11px] text-emerald-300/50">
                        Give this agent a purpose so it knows what to focus on.
                    </p>
                </div>
                <button
                    aria-label="Dismiss"
                    className="text-emerald-300/30 transition-colors hover:text-emerald-300/60"
                    onClick={onDismiss}
                    type="button"
                >
                    <IconX size={16} />
                </button>
            </div>
            <div className="space-y-3">
                <WizardField
                    label="Purpose"
                    onEnter={async (value) => {
                        if (await onSave('purpose', value)) {
                            onDismiss();
                        }
                    }}
                    padding="py-2.5"
                    placeholder="What is this agent's purpose?"
                />
                <div className="grid grid-cols-2 gap-3">
                    <WizardField
                        label="Profile"
                        onEnter={async (value) => {
                            await onSave('profile', value);
                        }}
                        padding="py-2"
                        placeholder="Describe their profile..."
                    />
                    <WizardField
                        label="Traits"
                        onEnter={async (value) => {
                            await onSave('traits', traitsOf(value));
                        }}
                        padding="py-2"
                        placeholder="e.g. curious, creative, diligent"
                    />
                </div>
                <p className="font-mono text-[10px] text-emerald-300/30">
                    Press Enter in any field to save. Fill purpose to dismiss this banner.
                </p>
            </div>
        </div>
    );
}
