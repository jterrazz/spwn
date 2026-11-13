import type { MachinefileChapter, MachinefileDrift } from '@jterrazz/station-api';

import { Button, Panel, Row, StatePill } from '../../design/index';
import { useRead } from '../../engine/react';
import { chapterWord, verdictShown } from '../../engine/shown';
import { driftAsked } from '../../model/machines';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route, Tab } from '../../shell/route';
import { Unanswered } from '../kit/readings';

import styles from './overview.module.css';

type Go = (to: Route) => void;

const CHAPTER_TABS: readonly { chapter: MachinefileChapter; tab: Tab; about: string }[] = [
    { about: 'what is installed', chapter: 'apps', tab: 'Packages' },
    { about: 'its settings', chapter: 'settings', tab: 'Packages' },
    { about: 'which ports are shared, and with whom', chapter: 'ports', tab: 'Ports' },
    { about: 'which accounts it borrows, and for how long', chapter: 'accounts', tab: 'Accounts' },
];

function routeOf(chapter: MachinefileChapter): { tab: Tab; about: string } {
    return CHAPTER_TABS.find((one) => one.chapter === chapter) ?? { about: '', tab: 'Packages' };
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
            {(drift === undefined
                ? CHAPTER_TABS.map(({ chapter }) => ({ chapter, state: null }))
                : (drift.Chapters ?? []).map((one) => ({
                      chapter: one.Chapter,
                      state: verdictShown(one.Verdict),
                  }))
            ).map(({ chapter, state }) => {
                const { tab, about } = routeOf(chapter);
                return (
                    <Row
                        key={chapter}
                        onOpen={() => {
                            go(machineRoute(vm, tab));
                        }}
                        sentence={about}
                        size="slim"
                        state={state}
                        title={chapterWord(chapter)}
                    />
                );
            })}
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
