import type { ProgressStep } from '../hooks/use-progress-messages';
import { apiPost } from './client';
import { textOf } from './fields';

// value: a first spawn builds the world's image, minutes on a cold machine
const SPAWN_BUDGET_MS = 600_000;

export const SPAWN_PROGRESS: ProgressStep[] = [
    { after: 0, text: 'Creating world...' },
    { after: 5, text: 'Building Docker image (first run could take a few minutes)...' },
    { after: 30, text: 'Still building... installing dependencies...' },
    { after: 60, text: 'Almost there...' },
];

/** Spawns a world as the API's POST /api/worlds describes it, and answers its id ('' when the API names none). */
export async function spawnWorld(request: Record<string, unknown>): Promise<string> {
    const world = await apiPost<unknown>('/api/worlds', request, {
        signal: AbortSignal.timeout(SPAWN_BUDGET_MS),
    });
    return textOf(world, 'id');
}

export function errorMessageOf(error: unknown, fallback: string): string {
    return error instanceof Error ? error.message : fallback;
}
