import type { ComponentProps } from 'react';

import type { QrCode } from './qr-code';
import type { Story } from './story';

export const QR_CODE_STORIES: readonly Story<ComponentProps<typeof QrCode>>[] = [
    { name: 'an address', props: { text: 'http://gate4.local:3000' } },
    { name: 'long text', props: { text: `https://gate4.tail1234.ts.net/${'path/'.repeat(20)}` } },
];
