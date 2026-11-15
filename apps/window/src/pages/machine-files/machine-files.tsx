import type { MachinefileFileView } from '@jterrazz/station-api';
import { useState } from 'react';

import { Caption, Missing, Switch } from '../../design/index';
import { wordOf } from '../../engine/engine-words';
import { useReach, useRead } from '../../engine/react';
import type { Reading } from '../../engine/react';
import type { Route } from '../../shell/route';
import type { Opening } from '../../shell/window';
import {
    Absent,
    NotGiven,
    NoVerdict,
    Reading as ReadingVerdict,
    Unanswered,
} from '../kit/readings';
import { layersOf } from '../packages/model';
import { Page } from '../page';
import { Chapters } from './chapters';
import { FileHead } from './file-head';
import type { Identities, Shown } from './file-head';
import { Own, Text, Users } from './file-panels';
import { shelvedOf } from './model';

import styles from './machine-files.module.css';

const RECIPE = 'A machine file is the recipe a machine is made from';

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
                        <NotGiven
                            title="Machines that follow it"
                            what="The machines made from it, and whether each matches it"
                        />
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
            <NoVerdict line={RECIPE} size="page" />
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
