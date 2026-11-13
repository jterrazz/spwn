import type { MachinesMacView } from '@jterrazz/station-api';

import { Button, Panel, Row } from '../../design/index';
import type { Route } from '../../shell/route';

type StartHereProps = {
    macs: readonly MachinesMacView[];
    go: (to: Route) => void;
    onNewMachine?: (() => void) | undefined;
};

function holdsNoMachine(macs: readonly MachinesMacView[]): boolean {
    return macs.length > 0 && macs.every((mac) => (mac.VMs ?? []).length === 0);
}

// value: the first steps, drawn while machines.list lists no machine on any Mac: a count of the rows it sends.
export function StartHere({ macs, go, onNewMachine }: StartHereProps) {
    if (!holdsNoMachine(macs)) {
        return null;
    }
    return (
        <Panel title="Start here">
            <Row
                sentence="A Linux machine of its own for an agent, made from an image: its tools are installed, and this Mac stays untouched"
                title="Create a workspace"
                trail={
                    <Button
                        disabled={onNewMachine === undefined}
                        name="New machine, a workspace"
                        onClick={onNewMachine}
                    >
                        New machine
                    </Button>
                }
            />
            <Row
                sentence="Its machine file: the apps and settings station keeps on it"
                title="Read what this Mac follows"
                trail={
                    <Button
                        onClick={() => {
                            go({ page: 'machine-files' });
                        }}
                    >
                        Open the file
                    </Button>
                }
            />
            <Row
                sentence="Apps, agent skills and settings, each one a package"
                title="Browse what station can install"
                trail={
                    <Button
                        onClick={() => {
                            go({ page: 'catalog' });
                        }}
                    >
                        Open Catalog
                    </Button>
                }
            />
        </Panel>
    );
}
