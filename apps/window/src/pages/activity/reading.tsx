import type {
    ActivityReadInput,
    ActivityReport,
    ActivitySample,
    EngineWatchInput,
    Target,
} from '@jterrazz/station-api';
import { useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';

import type { EngineError } from '../../engine/engine-error';
import { keyOf } from '../../engine/frame';
import { useWatch } from '../../engine/react';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { spanOf } from './measures';
import type { Span } from './measures';

export type Heard = { kind: 'failed'; error: EngineError } | { kind: 'read'; out: ActivityReport };

export function useAgain(every: number): number {
    const [tick, setTick] = useState(0);
    useEffect(() => {
        const timer = setInterval(() => {
            setTick((value) => value + 1);
        }, every);
        return () => {
            clearInterval(timer);
        };
    }, [every]);
    return tick;
}

export function useHeard(input: ActivityReadInput) {
    const key = keyOf({ call: 'activity.read', in: input });
    const [last, setLast] = useState<undefined | { key: string; heard: Heard }>(undefined);
    const heard = useCallback(
        (next: Heard) => {
            setLast({ heard: next, key });
        },
        [key],
    );
    return { heard, shown: last?.key === key ? last.heard : undefined };
}

export function Reader({
    input,
    target,
    heard,
}: {
    input: ActivityReadInput;
    target: Target | undefined;
    heard: (heard: Heard) => void;
}) {
    const reading = useWatch('activity.read', input, target);
    useEffect(() => {
        if (reading.kind !== 'reading') {
            heard(reading.kind === 'read' ? { kind: 'read', out: reading.out } : reading);
        }
    }, [reading, heard]);
    return null;
}

export function Held({ name, thisMac }: { name: string; thisMac: boolean }) {
    const machines: EngineWatchInput = { machines: thisMac ? [] : [name] };
    useWatch('engine.watch', machines);
    return null;
}

export function useHistory(found: Found): readonly ActivitySample[] | undefined {
    const vm = found.vm?.Name;
    const input: ActivityReadInput =
        vm === undefined
            ? { history: true }
            : { history: true, machine: { Name: vm, On: found.mac.Name }, of: vm };
    const reading = useWatch('activity.read', input, targetOf({ mac: found.mac }));
    return reading.kind === 'read' ? (reading.out.history ?? []) : undefined;
}

const AGAIN_MS = 5000;

function inputOf(mac: string, vm: string | undefined, span: Span): ActivityReadInput {
    const graph = spanOf(span).ns;
    if (vm === undefined) {
        return { graph };
    }
    return { graph, machine: { Name: vm, On: mac }, of: vm };
}

export function useActivity(
    found: Found,
    span: Span,
): { readers: ReactNode; shown: Heard | undefined } {
    const name = found.vm?.Name ?? found.mac.Name;
    const mac = found.mac.Name;
    const vm = found.vm?.Name;
    const tick = useAgain(AGAIN_MS);
    const onMac = targetOf({ mac: found.mac });
    const inVm = targetOf(found);
    const kept = inputOf(mac, vm, span);
    const own = inputOf(mac, undefined, span);
    const first = useHeard(kept);
    const second = useHeard(own);
    const unkept =
        vm !== undefined && first.shown?.kind === 'read' && first.shown.out.source === '';
    const readers = (
        <>
            <Held name={name} thisMac={found.mac.ThisMac && vm === undefined} />
            <Reader heard={first.heard} input={kept} key={`kept-${String(tick)}`} target={onMac} />
            {unkept ? (
                <Reader
                    heard={second.heard}
                    input={own}
                    key={`own-${String(tick)}`}
                    target={inVm}
                />
            ) : null}
        </>
    );
    return { readers, shown: unkept ? second.shown : first.shown };
}
