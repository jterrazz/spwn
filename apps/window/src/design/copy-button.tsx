import { useState } from 'react';

import { Button } from './button';
import { looksSecret } from './secret';

export type CopyButtonProps = { text: string; name?: string | undefined; label?: string };

export function CopyButton({ text, name, label = 'Copy' }: CopyButtonProps) {
    const [copied, setCopied] = useState<'copied' | 'idle' | 'refused'>('idle');
    if (looksSecret(text)) {
        return null;
    }
    const copy = async () => {
        try {
            await navigator.clipboard.writeText(text);
            setCopied('copied');
        } catch {
            setCopied('refused');
        }
    };
    const words = { copied: 'Copied', idle: label, refused: 'Select it to copy' } as const;
    return (
        <Button
            name={copied === 'idle' ? name : undefined}
            onClick={() => {
                void copy();
            }}
            variant="link"
        >
            {words[copied]}
        </Button>
    );
}
