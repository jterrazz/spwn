import type {
    ActivityReadInput,
    ActivityReport,
    ActivitySample,
    MachinesMacView,
    Target,
} from '@jterrazz/station-api';
import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

import { spanOf } from '../../../domain/activity/measures';
import type { Span } from '../../../domain/activity/measures';
import type { EngineError } from '../../../domain/engine-error';
import { targetOf } from '../../../domain/machines/machines';
import type { Found } from '../../../domain/machines/machines';
import { keyOf } from '../../../domain/request-key';
import { useWatch } from '../../dependencies/dependencies';
import type { Reading } from '../../dependencies/dependencies';

export type Heard =
    | { kind: 'failed'; error: EngineError }
    | { kind: 'read'; out: ActivityReport; placeholder?: true };

function placeheld(heard: Heard): Heard {
    return heard.kind === 'read' ? { ...heard, placeholder: true } : heard;
}

function heardOf(reading: Reading<ActivityReport>): Heard | undefined {
    if (reading.kind === 'failed') {
        return { error: reading.error, kind: 'failed' };
    }
    if (reading.kind === 'read') {
        return reading.placeholder === true
            ? { kind: 'read', out: reading.out, placeholder: true }
            : { kind: 'read', out: reading.out };
    }
    return undefined;
}

type Last = { key: string; of: string; reading: Reading<ActivityReport> };

function shownOf(last: Last | undefined, key: string, of: string): Heard | undefined {
    const one = last === undefined || last.of !== of ? undefined : heardOf(last.reading);
    return one !== undefined && last?.key !== key ? placeheld(one) : one;
}

// invariant: what was last heard of a machine stays shown while another span of it is read, marked a placeholder
// invariant: until that span answers; a reader of another machine starts from nothing. What is shown is derived from
// invariant: the one reading in the render that reads it, so an answer is drawn in the commit it arrives in.
function useHeard(
    input: ActivityReadInput | undefined,
    target: Target | undefined,
): Heard | undefined {
    const reading = useWatch('activity.read', input, target, { placeholder: true });
    const key = input === undefined ? '' : keyOf({ call: 'activity.read', in: input });
    const of = input?.of ?? '';
    const [last, setLast] = useState<Last>();
    if (
        reading.kind !== 'reading' &&
        (last?.reading !== reading || last.key !== key || last.of !== of)
    ) {
        setLast({ key, of, reading });
    }
    return useMemo(() => shownOf(last, key, of), [last, key, of]);
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

// value: what is shown of `found`'s activity over `span`: the engine keeps it watched while it is read, and a VM the
// value: engine keeps no history of is read from inside it.
export function useActivity(found: Found, span: Span): Heard | undefined {
    const name = found.vm?.Name ?? found.mac.Name;
    const mac = found.mac.Name;
    const vm = found.vm?.Name;
    useWatch('engine.watch', { machines: found.mac.ThisMac && vm === undefined ? [] : [name] });
    const first = useHeard(inputOf(mac, vm, span), targetOf({ mac: found.mac }));
    const unkept = vm !== undefined && first?.kind === 'read' && first.out.source === '';
    const second = useHeard(unkept ? inputOf(mac, undefined, span) : undefined, targetOf(found));
    return unkept ? second : first;
}

type HeardOn = (mac: string, heard: Heard | undefined) => void;

function MacReader({ mac, span, heard }: { mac: MachinesMacView; span: Span; heard: HeardOn }) {
    const shown = useActivity({ mac }, span);
    useEffect(() => {
        heard(mac.Name, shown);
    }, [heard, mac.Name, shown]);
    return null;
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
