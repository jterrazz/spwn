import type { PopoverRow } from '../popover';

// value: two Macs and their VMs as a popover lists them, linux-1 the current one.
export const MACHINES: readonly PopoverRow[] = [
    { id: 'macbook-pro', label: 'macbook-pro' },
    { id: 'api', indent: true, label: 'api', state: { tone: 'plain', word: 'Stopped' } },
    { current: true, id: 'linux-1', indent: true, label: 'linux-1' },
    { id: 'linux-2', indent: true, label: 'linux-2', state: { tone: 'new', word: 'Creating' } },
    { id: 'mac-mini', label: 'mac-mini' },
    { id: 'linux-3', indent: true, label: 'linux-3' },
];
