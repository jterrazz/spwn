import type { MachinesSshOutput, MachinesVM } from '@jterrazz/station-api';

import { shownOrWithheld } from '../../design/secret';
import { Openable } from '../../engine/opener';
import { useRead } from '../../engine/react';
import type { Reading } from '../../engine/react';
import { SSH_WORDS } from '../../engine/words';
import { OpenButton } from '../kit/opens';

export function terminalHeld(vm: MachinesVM, ssh: Reading<MachinesSshOutput>): string {
    if (!vm.SSH) {
        return `SSH ${SSH_WORDS.silent} on ${vm.Name}`;
    }
    if (ssh.kind === 'reading') {
        return `Reading the command that reaches ${vm.Name}…`;
    }
    if (ssh.kind === 'failed') {
        return shownOrWithheld(ssh.error.message);
    }
    return `The engine gives no command that reaches ${vm.Name}`;
}

// invariant: the command is the engine's own, read on this Mac's engine, the one whose Mac opens the terminal:
// invariant: `machines.ssh` answers how this Mac reaches the VM, wherever it runs.
export function TerminalButton({ vm }: { vm: MachinesVM }) {
    const ssh = useRead('machines.ssh', { Name: vm.Name });
    const openable = vm.SSH && ssh.kind === 'read' ? Openable.ofSsh(ssh.out) : undefined;
    return (
        <OpenButton
            held={terminalHeld(vm, ssh)}
            label="Open in Terminal"
            openable={openable}
            variant="primary"
            yields
        />
    );
}
