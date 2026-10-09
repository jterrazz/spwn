import { IconBrandDocker, IconKey, IconMessage, IconWorld } from '@tabler/icons-react';

import type { DockerStatus } from '../../contexts/docker-context';

export interface OnboardingStatus {
    completed: boolean;
    hasDocker: boolean;
    hasAuth: boolean;
    hasWorlds: boolean;
    hasAgents: boolean;
    docker: DockerStatus;
}

export type StepId = 'auth' | 'chat' | 'docker' | 'world';

const ORDER: StepId[] = ['docker', 'auth', 'world', 'chat'];

export const STEPS: {
    id: StepId;
    title: string;
    subtitle: string;
    icon: typeof IconBrandDocker;
}[] = [
    {
        icon: IconBrandDocker,
        id: 'docker',
        subtitle: 'spwn runs every agent inside an isolated Docker container.',
        title: 'Connect Docker',
    },
    {
        icon: IconKey,
        id: 'auth',
        subtitle: 'Sign in to Anthropic or OpenAI so your agents can think.',
        title: 'Connect a model provider',
    },
    {
        icon: IconWorld,
        id: 'world',
        subtitle: 'A world is an isolated workspace where one or more agents live.',
        title: 'Spawn your first world',
    },
    {
        icon: IconMessage,
        id: 'chat',
        subtitle: 'Send your first message and watch it work.',
        title: 'Talk to an agent',
    },
];

export function completed(step: StepId, status: OnboardingStatus): boolean {
    const done: Record<StepId, boolean> = {
        auth: status.hasAuth,
        chat: false,
        docker: status.hasDocker,
        world: status.hasWorlds,
    };
    return done[step];
}

export function nextStep(step: StepId): StepId {
    return ORDER[Math.min(ORDER.indexOf(step) + 1, ORDER.length - 1)] ?? step;
}

/** The step to show after a status arrives: past the open one once it is done, else the first one not done. */
export function stepAfter(open: StepId, status: OnboardingStatus): StepId {
    if (open !== 'docker' && completed(open, status)) {
        return nextStep(open);
    }
    return ORDER.find((step) => !completed(step, status)) ?? 'chat';
}
