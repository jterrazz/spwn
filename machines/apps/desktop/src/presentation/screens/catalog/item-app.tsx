import type { AppsReport, SettingsRow } from '@jterrazz/station-api';

import { sentence } from '../../../domain/format';
import { useRead } from '../../dependencies/dependencies';
import { AppIcon, Button, Facts, MissingMark, Panel, Row, StateMark, Value } from '../../ui/index';
import { countOf, detailOf, wordOf } from '../../words/engine-words';
import type { PlanLine } from '../../words/provisional';
import { appUpdate, kindChipWord, settingState } from '../../words/shown';
import { APP_STATE_TONES } from '../../words/tones';
import { APP_STATE_WORDS } from '../../words/words';
import { installLinesOf, usedBy } from './item-model';
import {
    factOf,
    FileLinks,
    Installs,
    ItemChips,
    ItemHead,
    MachineChips,
    originFact,
    Rail,
    Two,
} from './item-parts';
import type { ItemProps } from './item-parts';
import { categoryWord } from './shelves';

import styles from './item.module.css';

function Tags({ names }: { names: readonly string[] }) {
    return names.length === 0 ? null : (
        <span className={styles.chips}>
            {names.map((name) => (
                <Value key={name} look="raised">
                    {name}
                </Value>
            ))}
        </span>
    );
}

function HowInstalled({ report, go, routeOf }: ItemProps & { report: AppsReport }) {
    const { App } = report;
    const needs = (App.Needs ?? []).map((need) => wordOf('CatalogNeed', need).Word);
    const provides = App.Provides ?? [];
    const requires = App.Requires ?? [];
    const facts = [
        {
            label: 'Puts on your path',
            value: provides.length === 0 ? null : <Tags names={provides} />,
        },
        {
            label: 'Installs first',
            value:
                requires.length === 0 ? null : (
                    <ItemChips go={go} names={requires} routeOf={routeOf} />
                ),
        },
        { label: 'Asks you for', value: needs.length === 0 ? null : needs.join(', ') },
        { label: 'Opens as', value: App.Bundle === '' ? null : <Value>{App.Bundle}</Value> },
    ].filter((fact) => fact.value !== null);
    return (
        <Panel padded title="How it is installed">
            <Installs lines={installLinesOf(App.Installs)} />
            {facts.length === 0 ? null : <Facts facts={facts} layout="line" />}
        </Panel>
    );
}

function Version({ report, line }: { report: AppsReport; line: PlanLine | undefined }) {
    const update = appUpdate(report);
    const waiting = line?.Waiting ?? '';
    const rule = line?.Step.Rule ?? '';
    const here =
        report.Version === '' ? (
            <StateMark tone={APP_STATE_TONES[report.State]} word={APP_STATE_WORDS[report.State]} />
        ) : (
            <Value look="raised">{report.Version}</Value>
        );
    const newest = report.Newest === '' ? <MissingMark /> : <Value>{report.Newest}</Value>;
    return (
        <Panel padded title="Version">
            <Facts
                facts={[
                    { label: 'On your machines', value: here },
                    { label: 'Rule', value: rule === '' ? <MissingMark /> : rule },
                    {
                        label: 'Newest',
                        value: update ? (
                            <StateMark
                                tone={update.tone}
                                word={waiting === '' ? update.word : waiting}
                            />
                        ) : (
                            newest
                        ),
                    },
                    { label: 'Last moved', value: <MissingMark /> },
                ]}
            />
        </Panel>
    );
}

function ComesWith({
    report,
    settings,
    go,
    routeOf,
}: ItemProps & { report: AppsReport; settings: readonly SettingsRow[] }) {
    const then = report.App.Then ?? [];
    if (then.length === 0) {
        return null;
    }
    return (
        <Panel note="proposed once it is installed" title="Settings it comes with">
            <div className={styles.rows}>
                {then.map((name) => {
                    const row = settings.find((one) => one.Name === name);
                    return (
                        <Row
                            key={name}
                            lead={<AppIcon kind="setting" name={name} />}
                            onOpen={() => {
                                go(routeOf(name));
                            }}
                            sentence={row?.Package.Description}
                            size="slim"
                            state={row === undefined ? null : settingState(row.State)}
                            tag={kindChipWord('setting')}
                            tagLook="type"
                            title={name}
                        />
                    );
                })}
            </div>
        </Panel>
    );
}

function OnDisk({ report, go }: Pick<ItemProps, 'go'> & { report: AppsReport }) {
    const paths = report.App.Disk.Paths ?? [];
    return paths.length === 0 ? null : (
        <Panel
            action={
                <Button
                    onClick={() => {
                        go({ page: 'storage' });
                    }}
                    variant="link"
                >
                    Storage
                </Button>
            }
            title="What it keeps on disk"
        >
            <div className={styles.rows}>
                {paths.map((stored) => (
                    <Row
                        key={stored.Path}
                        kind={wordOf('CatalogLevel', stored.Level).Word}
                        sentence={sentence([detailOf('CatalogLevel', stored.Level), stored.Note])}
                        size="slim"
                        title={stored.Path}
                    />
                ))}
            </div>
        </Panel>
    );
}

function kindName(report: AppsReport): string {
    return wordOf('CatalogAppKind', report.App.AppKind).Word;
}

function declaredFact(go: ItemProps['go'], declared: readonly string[]) {
    return factOf(
        'Declared in',
        declared.length === 0 ? null : <FileLinks go={go} names={declared} />,
    );
}

function carriedFact(props: ItemProps, carried: readonly string[] | undefined) {
    return factOf(
        carried === undefined ? 'Machines' : countOf('machine', carried.length),
        carried === undefined || carried.length === 0 ? null : (
            <MachineChips go={props.go} macs={props.macs} names={carried} />
        ),
    );
}

export function AppItem({
    report,
    reports,
    carried,
    declared,
    settings,
    ...props
}: ItemProps & {
    report: AppsReport;
    reports: readonly AppsReport[];
    settings: readonly SettingsRow[];
    carried: readonly string[] | undefined;
    declared: readonly string[];
}) {
    const { App } = report;
    const users = usedBy(App.Name, reports);
    const drift = useRead('machinefile.drift', {});
    const line =
        drift.kind === 'read'
            ? [...(drift.out.Plan.Lines ?? []), ...(drift.out.Plan.Extra ?? [])].find(
                  (one) => one.Step.Package === App.Name,
              )
            : undefined;
    return (
        <>
            <ItemHead
                about={App.Description}
                kind={App.Kind ?? 'app'}
                kindWord={App.Kind === undefined ? '' : kindChipWord(App.Kind)}
                name={App.Name}
                version={report.Version}
            />
            <Two
                main={
                    <>
                        <HowInstalled {...props} report={report} />
                        <Version line={line} report={report} />
                        <ComesWith {...props} report={report} settings={settings} />
                        <OnDisk go={props.go} report={report} />
                    </>
                }
                rail={
                    <Rail
                        facts={[
                            factOf('Kind', kindName(report)),
                            factOf('Category', categoryWord(App.Category)),
                            declaredFact(props.go, declared),
                            carriedFact(props, carried),
                            ...(users.length === 0
                                ? []
                                : [factOf('Used by', <ItemChips {...props} names={users} />)]),
                            originFact(report.Origin),
                        ]}
                        name={App.Name}
                    />
                }
            />
        </>
    );
}
