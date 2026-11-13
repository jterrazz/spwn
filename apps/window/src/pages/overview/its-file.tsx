import { Button, MissingMark, Panel, Row } from '../../design/index';
import { useRead } from '../../engine/react';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route, Tab } from '../../shell/route';
import { Unanswered } from '../kit/readings';

type Go = (to: Route) => void;

const CHAPTERS: readonly { tab: Tab; about: string }[] = [
    { about: 'what is installed, and its settings', tab: 'Packages' },
    { about: 'which ports are shared, and with whom', tab: 'Ports' },
    { about: 'which accounts it borrows, and for how long', tab: 'Accounts' },
];

export function FileChapters({ vm, file, go }: { vm: string; file: string; go: Go }) {
    return (
        <Panel
            action={
                <Button
                    onClick={() => {
                        go(machineRoute(vm, 'Packages'));
                    }}
                >
                    Plan
                </Button>
            }
            note={file}
            title="Its file"
        >
            {CHAPTERS.map((chapter) => (
                <Row
                    key={chapter.tab}
                    onOpen={() => {
                        go(machineRoute(vm, chapter.tab));
                    }}
                    sentence={chapter.about}
                    size="slim"
                    title={chapter.tab}
                    trail={<MissingMark />}
                />
            ))}
        </Panel>
    );
}

export function ItsFile({ found, go }: { found: Found; go: Go }) {
    const vm = found.vm?.Name ?? found.mac.Name;
    const drift = useRead('machinefile.drift', {}, targetOf(found));
    if (drift.kind === 'failed') {
        return (
            <Panel title="Its file">
                <Unanswered error={drift.error} size="panel" />
            </Panel>
        );
    }
    return <FileChapters file={drift.kind === 'read' ? drift.out.Plan.Name : ''} go={go} vm={vm} />;
}
