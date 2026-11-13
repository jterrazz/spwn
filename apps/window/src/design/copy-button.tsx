import { useEffect, useState } from 'react';

import { Button } from './button';
import { looksSecret } from './secret';

export type CopyButtonProps = {
    text: string;
    name?: string | undefined;
    label?: string;
    variant?: 'inline' | 'link';
};

type Said = 'copied' | 'idle' | 'refused';

const REST_AFTER_MS = 1500;

export function CopyButton({ text, name, label = 'Copy', variant = 'link' }: CopyButtonProps) {
    const [said, setSaid] = useState<Said>('idle');
    useEffect(() => {
        if (said === 'idle') {
            return;
        }
        const rest = setTimeout(() => {
            setSaid('idle');
        }, REST_AFTER_MS);
        return () => {
            clearTimeout(rest);
        };
    }, [said]);
    if (looksSecret(text)) {
        return null;
    }
    const copy = async () => {
        try {
            await navigator.clipboard.writeText(text);
            setSaid('copied');
        } catch {
            setSaid('refused');
        }
    };
    const words = { copied: 'Copied', idle: label, refused: 'Select it to copy' } as const;
    return (
        <Button
            name={said === 'idle' ? name : undefined}
            onClick={() => {
                void copy();
            }}
            variant={variant}
        >
            {words[said]}
        </Button>
    );
}
