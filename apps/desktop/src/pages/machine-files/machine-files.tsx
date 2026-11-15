import type { MachinefileFileView } from '@jterrazz/station-api';
import { useState } from 'react';

import { useReach, useRead } from '../../presentation/dependencies/dependencies';
import type { Reading } from '../../presentation/dependencies/dependencies';
import type { Route } from '../../presentation/navigation/route';
import type { Opening } from '../../presentation/navigation/window';
import { Caption, Missing, Switch } from '../../presentation/ui/index';
import { isCalm } from '../../presentation/ui/state-pill';
import { wordOf } from '../../presentation/words/engine-words';
import { Absent, Reading as ReadingVerdict, Unanswered, VerdictHead } from '../kit/readings';
import { layersOf } from '../packages/model';
import { Page } from '../page';
import { Chapters } from './chapters';
import { FileHead } from './file-head';
import type { Identities, Shown } from './file-head';
import { Followers, Own, Text, Users } from './file-panels';
import { shelvedOf } from './model';

import styles from './machine-files.module.css';

type Go = (to: Route) => void;

function Tabs({
    files,
    file,
    choose,
    identities,
}: {
    files: readonly MachinefileFileView[];
    file: string;
    choose: (file: string) => void;
    identities: Identities;
}) {
    return (
        <div className={styles.tabs}>
            {shelvedOf(files).map(({ shelf, files: shelved }) => (
                <div className={styles.shelf} key={shelf}>
                    <Caption>{wordOf('MachinefileShelf', shelf).Word}</Caption>
                    <Switch
                        label={wordOf('MachinefileShelf', shelf).Word}
                        onChange={choose}
                        options={shelved.map((candidate) => ({
                            flagged: isCalm(candidate.Verdict.Tone)
                                ? undefined
                                : candidate.Verdict.Sentence,
                            id: candidate.Name,
                            identity: identities.get(candidate.Name),
                            label: candidate.Name,
                        }))}
                        surface="bare"
                        value={file}
                    />
                </div>
            ))}
        </div>
    );
}

function Waiting({ reading, what }: { reading: Reading<unknown>; what: string }) {
    if (reading.kind === 'failed') {
        return <Unanswered error={reading.error} size="page" />;
    }
    return <ReadingVerdict size="page" what={what} />;
}

function File({
    files,
    view,
    choose,
    go,
    onNewMachine,
}: {
    files: readonly MachinefileFileView[];
    view: MachinefileFileView;
    choose: (file: string) => void;
    go: Go;
    onNewMachine?: ((opening: Opening) => void) | undefined;
}) {
    const file = view.Name;
    const [shown, setShown] = useState<Shown>('fields');
    const drift = useRead('machinefile.drift', { Machinefile: file });
    const identities: Identities = new Map(
        drift.kind === 'read'
            ? layersOf(drift.out.Plan).map((layer) => [layer.name, layer.identity] as const)
            : [],
    );
    return (
        <>
            <Tabs choose={choose} file={file} files={files} identities={identities} />
            <FileHead
                choose={choose}
                identities={identities}
                onNewMachine={onNewMachine}
                show={setShown}
                shown={shown}
                view={view}
            />
            {shown === 'fields' ? (
                <>
                    {view.Head === undefined ? (
                        <Users choose={choose} files={files} name={file} />
                    ) : (
                        <Followers go={go} view={view} />
                    )}
                    {drift.kind === 'read' ? (
                        <Chapters file={file} go={go} plan={drift.out.Plan} />
                    ) : drift.kind === 'failed' ? (
                        <Unanswered error={drift.error} size="tab" />
                    ) : (
                        <ReadingVerdict size="tab" what={`This Mac against ${file}`} />
                    )}
                    <Own view={view} />
                </>
            ) : (
                <Text show={setShown} shown={shown} view={view} />
            )}
        </>
    );
}

export function MachineFilesPage({
    go,
    opened,
    onNewMachine,
}: {
    go: Go;
    opened?: string | undefined;
    onNewMachine?: ((opening: Opening) => void) | undefined;
}) {
    const reach = useReach();
    const explain = useRead('machinefile.explain', {});
    const files = useRead('machinefile.files', {});
    const [chosen, setChosen] = useState<string | undefined>(undefined);
    if (reach.kind === 'unanswered') {
        return (
            <Page>
                <Absent reach={reach} />
            </Page>
        );
    }
    if (explain.kind !== 'read') {
        return (
            <Page>
                <Waiting reading={explain} what="the machine file This Mac follows" />
            </Page>
        );
    }
    if (files.kind !== 'read') {
        return (
            <Page>
                <Waiting reading={files} what="the machine files This Mac reads" />
            </Page>
        );
    }
    const all = files.out.Files ?? [];
    const file = chosen ?? opened ?? explain.out.Name;
    const view = all.find((candidate) => candidate.Name === file);
    return (
        <Page>
            <VerdictHead size="page" verdict={files.out.Verdict} />
            {view === undefined ? (
                <Missing>{`No machine file named ${file}`}</Missing>
            ) : (
                <File
                    choose={setChosen}
                    files={all}
                    go={go}
                    key={file}
                    onNewMachine={onNewMachine}
                    view={view}
                />
            )}
        </Page>
    );
}
