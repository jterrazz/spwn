import type { MachinefileChapter, MachinefileDrift } from '@jterrazz/station-api';

import { Button, Panel, Row, StatePill } from '../../design/index';
import { useRead } from '../../engine/react';
import { verdictShown } from '../../engine/shown';
import { driftAsked } from '../../model/machines';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route, Tab } from '../../shell/route';
import { Unanswered } from '../kit/readings';

import styles from './overview.module.css';

type Go = (to: Route) => void;

// value: the mockup's three rows, one per tab; Packages reads the apps chapter, as the Packages tab's head does.
const FILE_ROWS: readonly { chapter: MachinefileChapter; tab: Tab; about: string }[] = [
    { about: 'what is installed, and its settings', chapter: 'apps', tab: 'Packages' },
    { about: 'which ports are shared, and with whom', chapter: 'ports', tab: 'Ports' },
    { about: 'which accounts it borrows, and for how long', chapter: 'accounts', tab: 'Accounts' },
];

function stateOf(drift: MachinefileDrift | undefined, chapter: MachinefileChapter) {
    const read = (drift?.Chapters ?? []).find((one) => one.Chapter === chapter);
    return read === undefined ? null : { ...verdictShown(read.Verdict), calm: 'pill' as const };
}

export function FileChapters({
    vm,
    drift,
    go,
}: {
    vm: string;
    drift: MachinefileDrift | undefined;
    go: Go;
}) {
    return (
        <Panel
            action={
                <span className={styles.fileHead}>
                    {drift === undefined ? null : <StatePill {...verdictShown(drift.Verdict)} />}
                    <Button
                        onClick={() => {
                            go(machineRoute(vm, 'Packages'));
                        }}
                    >
                        Plan
                    </Button>
                </span>
            }
            note={drift?.Plan.Name ?? ''}
            title="Its file"
        >
            {FILE_ROWS.map(({ chapter, tab, about }) => (
                <Row
                    key={chapter}
                    onOpen={() => {
                        go(machineRoute(vm, tab));
                    }}
                    sentence={about}
                    size="slim"
                    state={stateOf(drift, chapter)}
                    title={tab}
                />
            ))}
        </Panel>
    );
}

export function ItsFile({ found, go }: { found: Found; go: Go }) {
    const vm = found.vm?.Name ?? found.mac.Name;
    const drift = useRead('machinefile.drift', driftAsked(found));
    if (drift.kind === 'failed') {
        return (
            <Panel title="Its file">
                <Unanswered error={drift.error} size="panel" />
            </Panel>
        );
    }
    return <FileChapters drift={drift.kind === 'read' ? drift.out : undefined} go={go} vm={vm} />;
}
