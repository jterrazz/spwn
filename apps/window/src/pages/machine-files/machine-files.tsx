import type { MachinefileCallsExplainOut, MachinefilePlan } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { Icon } from '../../design/icon';
import { Button, Facts, Missing, Panel, Switch, Value } from '../../design/index';
import type { Identity } from '../../design/index';
import { makesAVm } from '../../engine/provisional';
import { useReach, useRead, useWatch } from '../../engine/react';
import type { Reading } from '../../engine/react';
import { machineKindWord } from '../../engine/shown';
import type { Route } from '../../shell/route';
import type { Opening } from '../../shell/window';
import {
    Absent,
    NotGiven,
    NoVerdict,
    Reading as ReadingVerdict,
    Unanswered,
} from '../kit/readings';
import { LayerList } from '../packages/layers';
import { categoriesOf, countOn, layersOf, onShelf } from '../packages/model';
import { Page } from '../page';
import { catalogRouteOf, CHAPTERS, descriptionsOf, extendedBy, factsOf, filesOf } from './model';
import type { Chapter, Extended } from './model';

import styles from './machine-files.module.css';

const RECIPE = 'A machine file is the recipe a machine is made from';

type Go = (to: Route) => void;

function Chapters({ plan, file, go }: { plan: MachinefilePlan; file: string; go: Go }) {
    const [chapter, setChapter] = useState<Chapter>('Apps');
    const apps = useWatch('apps.list', {});
    const reports = apps.kind === 'read' ? (apps.out ?? []) : [];
    const categories = categoriesOf(reports);
    const layers = layersOf(plan);
    return (
        <Panel
            action={
                <Switch
                    label="Chapter"
                    onChange={setChapter}
                    options={CHAPTERS.map((id) => ({
                        count: id === 'Access' ? undefined : countOn(layers, id, categories),
                        id,
                        label: id,
                    }))}
                    surface="panel"
                    value={chapter}
                />
            }
            note="the boxed values are the ones you set"
            title="What it puts on a machine"
        >
            {chapter === 'Access' ? (
                <Missing>The accounts it shares and the ports it opens</Missing>
            ) : (
                <LayerList
                    layers={onShelf(layers, chapter, categories)}
                    look={{
                        descriptions: descriptionsOf(reports),
                        file,
                        kind: 'file',
                        open: (name) => {
                            go(catalogRouteOf(chapter, name));
                        },
                    }}
                    shelf={chapter}
                />
            )}
        </Panel>
    );
}

type Identities = ReadonlyMap<string, Identity>;

function Files({
    explain,
    file,
    choose,
    identities,
}: {
    explain: MachinefileCallsExplainOut;
    file: string;
    choose: (file: string) => void;
    identities: Identities;
}) {
    const files = filesOf(explain);
    if (files.length < 2) {
        return null;
    }
    return (
        <Switch
            label="Machine files"
            onChange={choose}
            options={files.map((candidate) => ({
                id: candidate.Name,
                identity: identities.get(candidate.Name),
                label: candidate.Name,
            }))}
            surface="bare"
            value={file}
        />
    );
}

function Extends({
    files,
    choose,
}: {
    files: readonly Extended[];
    choose: (file: string) => void;
}) {
    return (
        <span className={styles.extends}>
            {files.map((extended) => (
                <button
                    className={styles.extended}
                    key={extended.name}
                    onClick={() => {
                        choose(extended.name);
                    }}
                    type="button"
                >
                    <Value identity={extended.identity} look="raised">
                        {extended.name}
                    </Value>
                </button>
            ))}
        </span>
    );
}

function Head({
    explain,
    file,
    choose,
    identities,
    onNewMachine,
}: {
    explain: MachinefileCallsExplainOut;
    file: string;
    choose: (file: string) => void;
    identities: Identities;
    onNewMachine?: ((opening: Opening) => void) | undefined;
}) {
    const origin = filesOf(explain).find((candidate) => candidate.Name === file);
    const path = origin?.File ?? '';
    const named = path === '' ? file : (path.split('/').at(-1) ?? file);
    const { makes, system } = factsOf(explain, file);
    const extended = extendedBy(explain, file).map((one) => ({
        ...one,
        identity: identities.get(one.name) ?? one.identity,
    }));
    const facts: { label: string; value: ReactNode }[] = [
        { label: 'Makes', value: makes === '' ? null : `a ${machineKindWord(makes)}` },
        { label: 'System', value: system === '' ? null : system },
        {
            label: 'Extends',
            value: extended.length === 0 ? null : <Extends choose={choose} files={extended} />,
        },
    ].filter((fact) => fact.value !== null);
    return (
        <section aria-label={named} className={styles.file}>
            <div className={styles.head}>
                <span className={styles.icon}>
                    <Icon name="file" />
                </span>
                <div className={styles.named}>
                    <h2 className={styles.name}>{named}</h2>
                    {path === '' ? null : <span className={styles.path}>{path}</span>}
                </div>
                {makesAVm(makes) && onNewMachine !== undefined ? (
                    <span className={styles.actions}>
                        <Button
                            onClick={() => {
                                onNewMachine({ from: file });
                            }}
                            variant="primary"
                        >
                            New machine
                        </Button>
                    </span>
                ) : null}
            </div>
            {facts.length === 0 ? null : <Facts facts={facts} />}
        </section>
    );
}

function Waiting({ reading, what }: { reading: Reading<unknown>; what: string }) {
    if (reading.kind === 'failed') {
        return <Unanswered error={reading.error} size="page" />;
    }
    return <ReadingVerdict size="page" what={what} />;
}

function File({
    bound,
    file,
    choose,
    go,
    onNewMachine,
}: {
    bound: MachinefileCallsExplainOut;
    file: string;
    choose: (file: string) => void;
    go: Go;
    onNewMachine?: ((opening: Opening) => void) | undefined;
}) {
    const input = { Machinefile: file };
    const drift = useRead('machinefile.drift', input);
    const own = useRead('machinefile.explain', input);
    const identities: Identities = new Map(
        drift.kind === 'read'
            ? layersOf(drift.out.Plan).map((layer) => [layer.name, layer.identity] as const)
            : [],
    );
    return (
        <>
            <Files choose={choose} explain={bound} file={file} identities={identities} />
            <Head
                choose={choose}
                explain={own.kind === 'read' ? own.out : bound}
                file={file}
                identities={identities}
                onNewMachine={onNewMachine}
            />
            <NotGiven
                title="Machines that follow it"
                what="The machines made from it, and whether each matches it"
            />
            {drift.kind === 'read' ? (
                <Chapters file={file} go={go} plan={drift.out.Plan} />
            ) : drift.kind === 'failed' ? (
                <Unanswered error={drift.error} size="tab" />
            ) : (
                <ReadingVerdict size="tab" what={`This Mac against ${file}`} />
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
    const file = chosen ?? opened ?? explain.out.Name;
    return (
        <Page>
            <NoVerdict line={RECIPE} size="page" />
            <File
                bound={explain.out}
                choose={setChosen}
                file={file}
                go={go}
                onNewMachine={onNewMachine}
            />
        </Page>
    );
}
