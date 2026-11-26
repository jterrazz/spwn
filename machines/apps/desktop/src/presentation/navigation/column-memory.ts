import type { Pin } from './pin';
import type { Memory } from './route-memory';

const KEY = 'station.window.column';

export function recalledPin(memory: Memory | undefined): Pin | undefined {
    try {
        const text = memory?.getItem(KEY) ?? null;
        return text === 'pinned' || text === 'hidden' ? text : undefined;
    } catch {
        return undefined;
    }
}

export function rememberPin(memory: Memory | undefined, pin: Pin): void {
    try {
        memory?.setItem(KEY, pin);
    } catch {
        // hazard: a webview that refuses storage still pins; the next launch follows the width again.
    }
}
