import type {
    ActivityReadInput,
    ActivityReport,
    ActivitySample,
    EngineWatchInput,
    MachinesMacView,
    Target,
} from '@jterrazz/station-api';
import { useCallback, useEffect, useState } from 'react';
import type { ReactElement, ReactNode } from 'react';

import { spanOf } from '../../../domain/activity/measures';
import type { Span } from '../../../domain/activity/measures';
import type { EngineError } from '../../../domain/engine-error';
import { targetOf } from '../../../domain/machines/machines';
import type { Found } from '../../../domain/machines/machines';
import { keyOf } from '../../../domain/request-key';
import { useWatch } from '../../dependencies/dependencies';

export type Heard =
    | { kind: 'failed'; error: EngineError }
    | { kind: 'read'; out: ActivityReport; placeholder?: true };

function placeheld(heard: Heard): Heard {
    return heard.kind === 'read' ? { ...heard, placeholder: true } : heard;
}

// invariant: what was last heard of a machine stays shown while another span of it is read, marked a placeholder
// invariant: until that span answers; a reader of another machine starts from nothing.
export function useHeard(input: ActivityReadInput) {
    const key = keyOf({ call: 'activity.read', in: input });
    const of = input.of ?? '';
    const [last, setLast] = useState<undefined | { key: string; of: string; heard: Heard }>(
        undefined,
    );
    const heard = useCallback(
        (next: Heard) => {
            setLast({ heard: next, key, of });
        },
        [key, of],
    );
    if (last === undefined || last.of !== of) {
        return { heard, shown: undefined };
    }
    return { heard, shown: last.key === key ? last.heard : placeheld(last.heard) };
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
    const reading = useWatch('activity.read', input, target, { placeholder: true });
    useEffect(() => {
        if (reading.kind === 'failed') {
            heard(reading);
        } else if (reading.kind === 'read') {
            heard(
                reading.placeholder === true
                    ? { kind: 'read', out: reading.out, placeholder: true }
                    : { kind: 'read', out: reading.out },
            );
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
): { readers: ReactElement; shown: Heard | undefined } {
    const name = found.vm?.Name ?? found.mac.Name;
    const mac = found.mac.Name;
    const vm = found.vm?.Name;
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
            <Reader heard={first.heard} input={kept} target={onMac} />
            {unkept ? <Reader heard={second.heard} input={own} target={inVm} /> : null}
        </>
    );
    return { readers, shown: unkept ? second.shown : first.shown };
}

type HeardOn = (mac: string, heard: Heard | undefined) => void;

function MacReader({ mac, span, heard }: { mac: MachinesMacView; span: Span; heard: HeardOn }) {
    const { readers, shown } = useActivity({ mac }, span);
    useEffect(() => {
        heard(mac.Name, shown);
    }, [heard, mac.Name, shown]);
    return readers;
}

function withHeard(
    kept: ReadonlyMap<string, Heard>,
    mac: string,
    heard: Heard | undefined,
): ReadonlyMap<string, Heard> {
    if (kept.get(mac) === heard) {
        return kept;
    }
    const next = new Map(kept);
    if (heard === undefined) {
        next.delete(mac);
    } else {
        next.set(mac, heard);
    }
    return next;
}

export function useActivities(
    macs: readonly MachinesMacView[],
    span: Span,
): { readers: ReactNode; shown: ReadonlyMap<string, Heard> } {
    const [shown, setShown] = useState<ReadonlyMap<string, Heard>>(new Map());
    const heard = useCallback<HeardOn>((mac, one) => {
        setShown((kept) => withHeard(kept, mac, one));
    }, []);
    const readers = macs.map((mac) => (
        <MacReader heard={heard} key={mac.Name} mac={mac} span={span} />
    ));
    return { readers, shown };
}
