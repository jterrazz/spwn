import type { MachinesMacView } from '@jterrazz/station-api';
import { useState } from 'react';
import type { RefObject } from 'react';

import { machineNamesOf } from '../../domain/machines/machines';
import { identityOf, MachineIcon, Popover } from '../ui/index';
import type { PopoverRow } from '../ui/index';
import { machineKindOf } from '../words/provisional';
import { macState, vmState } from '../words/shown';

export type Opened = { anchor: RefObject<HTMLElement | null>; id: string; onClose: () => void };

type Fleet = {
    macs: readonly MachinesMacView[];
    current: string | undefined;
    mac?: string | undefined;
};

export function matches(label: string, query: string): boolean {
    return label.toLowerCase().includes(query.trim().toLowerCase());
}

// value: the machine list, as the sidebar draws it: each Mac, then its VMs under it; for one Mac, its VMs alone.
export function machineRows({ macs, current, mac }: Fleet): readonly PopoverRow[] {
    const listed = machineNamesOf(macs);
    const shown = mac === undefined ? macs : macs.filter((one) => one.Name === mac);
    return shown.flatMap((one) => [
        ...(mac === undefined
            ? [
                  {
                      current: one.Name === current,
                      id: one.Name,
                      label: one.Name,
                      lead: <MachineIcon kind="mac" />,
                      state: macState(one),
                  },
              ]
            : []),
        ...(one.VMs ?? []).map((vm) => ({
            current: vm.Name === current,
            id: vm.Name,
            indent: mac === undefined,
            label: vm.Name,
            lead: <MachineIcon identity={identityOf(vm.Name, listed)} kind={machineKindOf(vm)} />,
            state: vmState(vm),
        })),
    ]);
}

type MachinePopoverProps = Fleet & Opened & { onChoose: (machine: string) => void };

// invariant: a filter lists what it keeps flat, as a Mac it hides would leave its VMs under nothing.
export function MachinePopover({ anchor, id, onClose, onChoose, ...fleet }: MachinePopoverProps) {
    const [query, setQuery] = useState('');
    const rows = machineRows(fleet);
    const kept =
        query.trim() === ''
            ? rows
            : rows
                  .filter((row) => matches(row.label, query))
                  .map((row) => ({ ...row, indent: false }));
    const label = fleet.mac === undefined ? 'Machines' : `Machines on ${fleet.mac}`;
    return (
        <Popover
            anchor={anchor}
            empty="No machine matches"
            filter={{
                label: 'Filter machines',
                onChange: setQuery,
                placeholder: 'Filter machines',
                value: query,
            }}
            id={id}
            label={label}
            onChoose={onChoose}
            onClose={onClose}
            sections={[{ label: fleet.mac ?? 'Machines', rows: kept }]}
        />
    );
}
