import type { MachinefileExplainOutput, MachinefileFileView } from '@jterrazz/station-api';
import { useState } from 'react';

import { useReach, useRead } from '../../dependencies/dependencies';
import type { Reading } from '../../dependencies/dependencies';
import { planFilesOf } from '../../features/machinefile/model';
import { Absent, Unanswered, VerdictHead } from '../../kit/readings';
import { SkeletonHead } from '../../kit/skeleton';
import type { Route } from '../../navigation/route';
import type { Opening } from '../../navigation/window';
import { Caption, Missing, Switch } from '../../ui/index';
import { Page } from '../../ui/page';
import { wordOf } from '../../words/engine-words';
import { UNFLAGGED_FILE_TONES } from '../../words/tones';
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
                            flagged: UNFLAGGED_FILE_TONES.has(candidate.Verdict.Tone)
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

function Waiting({
    reading,
    size,
    what,
}: {
    reading: Reading<unknown>;
    size: 'page' | 'tab';
    what: string;
}) {
    if (reading.kind === 'failed') {
        return <Unanswered error={reading.error} size={size} />;
    }
    return <SkeletonHead size={size} what={what} />;
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
            ? planFilesOf(drift.out.Plan).map(
                  (planFile) => [planFile.name, planFile.identity] as const,
              )
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
                        <SkeletonHead size="tab" what={`This Mac against ${file}`} />
                    )}
                    <Own view={view} />
                </>
            ) : (
                <Text show={setShown} shown={shown} view={view} />
            )}
        </>
    );
}

function openedBy(
    explain: Reading<MachinefileExplainOutput>,
    files: readonly MachinefileFileView[],
): string {
    if (explain.kind === 'read') {
        return explain.out.Name;
    }
    return shelvedOf(files)[0]?.files[0]?.Name ?? '';
}

function Following({ explain }: { explain: Reading<MachinefileExplainOutput> }) {
    if (explain.kind === 'read') {
        return null;
    }
    if (explain.kind === 'failed') {
        return <Unanswered error={explain.error} size="panel" />;
    }
    return <SkeletonHead size="tab" what="the machine file This Mac follows" />;
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
    if (reach.kind === 'unanswered' && (explain.kind !== 'read' || files.kind !== 'read')) {
        return (
            <Page density="compact">
                <Absent reach={reach} />
            </Page>
        );
    }
    if (files.kind !== 'read') {
        return (
            <Page density="compact">
                <Waiting reading={files} size="page" what="the machine files This Mac reads" />
            </Page>
        );
    }
    const all = files.out.Files ?? [];
    const file = chosen ?? opened ?? openedBy(explain, all);
    const view = all.find((candidate) => candidate.Name === file);
    return (
        <Page density="compact">
            <VerdictHead size="page" verdict={files.out.Verdict} />
            <Following explain={explain} />
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
