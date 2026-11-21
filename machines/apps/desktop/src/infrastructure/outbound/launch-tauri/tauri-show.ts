import type { Ipc } from '../engine-tauri/tauri';

export function tauriShow(ipc: Pick<Ipc, 'invoke'>): () => Promise<void> {
    return async () => {
        await Promise.allSettled([ipc.invoke('window_ready')]);
    };
}

export const NO_SHOW = async (): Promise<void> => {
    await Promise.resolve();
};
