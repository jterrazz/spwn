import type { CallName, Target } from '@jterrazz/station-api';

import { EXPOSED_ROUTE } from '../../engine/provisional';
import { useGesture } from '../../engine/use-gesture';
import type { Gesture } from '../../engine/use-gesture';

const PORT_REREADS: readonly CallName[] = ['ports.list', 'overview.read'];

type PortOf = { vm: string; port: number; target: Target | undefined };

export function usePortShare({ vm, port, target }: PortOf): Gesture<unknown> {
    return useGesture('ports.share', {
        input: { Port: String(port), VM: vm },
        rereads: PORT_REREADS,
        subject: `${vm}:${port}`,
        target,
    });
}

type DeclaredOf = { machine: string; port: number; target: Target | undefined };

export function usePortDeclare({ machine, port, target }: DeclaredOf): Gesture<unknown> {
    return useGesture('ports.declare', {
        input: { Machine: machine, Port: String(port), Via: EXPOSED_ROUTE },
        rereads: PORT_REREADS,
        subject: `${machine}:${port}`,
        target,
    });
}

export function usePortStop({ vm, port, target }: PortOf): Gesture<unknown> {
    return useGesture('ports.stop-sharing', {
        input: { Port: String(port), VM: vm },
        rereads: PORT_REREADS,
        subject: `${vm}:${port}`,
        target,
    });
}
