import type { AppsReport } from '@jterrazz/station-api';

import { Button, Facts, MissingMark, Panel, Row, StatePill, Value } from '../../design/index';
import { isSkill } from '../../engine/provisional';
import { appUpdate } from '../../engine/shown';
import { APP_STATE_TONES } from '../../engine/tones';
import {
    APP_STATE_WORDS,
    CATALOG_KIND_NAMES,
    CATALOG_KIND_WORDS,
    CATALOG_LEVEL_LINES,
    CATALOG_LEVEL_WORDS,
    NEED_WORDS,
    SKILL_KIND_NAME,
} from '../../engine/words';
import { sentence } from '../../model/format';
import { installLinesOf, usedBy } from './item-model';
import { factOf, Installs, ItemChips, ItemHead, Rail, Two } from './item-parts';
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
    const needs = (App.Needs ?? []).map((need) => NEED_WORDS[need]);
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

function Version({ report }: { report: AppsReport }) {
    const update = appUpdate(report);
    const here =
        report.Version === '' ? (
            <StatePill
                quiet
                tone={APP_STATE_TONES[report.State]}
                word={APP_STATE_WORDS[report.State]}
            />
        ) : (
            <Value look="raised">{report.Version}</Value>
        );
    const newest = report.Newest === '' ? <MissingMark /> : <Value>{report.Newest}</Value>;
    return (
        <Panel padded title="Version">
            <Facts
                facts={[
                    { label: 'On your machines', value: here },
                    { label: 'Rule', value: <MissingMark /> },
                    {
                        label: 'Newest',
                        value: update ? (
                            <StatePill tone={update.tone} word={update.word} />
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

function ComesWith({ report, go, routeOf }: ItemProps & { report: AppsReport }) {
    const then = report.App.Then ?? [];
    return then.length === 0 ? null : (
        <Panel note="proposed once it is installed" padded title="Settings it comes with">
            <ItemChips go={go} names={then} routeOf={routeOf} />
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
                        kind={CATALOG_LEVEL_WORDS[stored.Level]}
                        sentence={sentence([CATALOG_LEVEL_LINES[stored.Level], stored.Note])}
                        size="slim"
                        title={stored.Path}
                    />
                ))}
            </div>
        </Panel>
    );
}

function kindName(report: AppsReport): string {
    if (isSkill(report.App.Category)) {
        return SKILL_KIND_NAME;
    }
    return CATALOG_KIND_NAMES[report.App.Kind ?? 'app'];
}

export function AppItem({
    report,
    reports,
    ...props
}: ItemProps & { report: AppsReport; reports: readonly AppsReport[] }) {
    const { App } = report;
    const users = usedBy(App.Name, reports);
    return (
        <>
            <ItemHead
                about={App.Description}
                kind={App.Kind ?? 'app'}
                kindWord={App.Kind === undefined ? '' : CATALOG_KIND_WORDS[App.Kind]}
                name={App.Name}
                version={report.Version}
            />
            <Two
                main={
                    <>
                        <HowInstalled {...props} report={report} />
                        <Version report={report} />
                        <ComesWith {...props} report={report} />
                        <OnDisk go={props.go} report={report} />
                    </>
                }
                rail={
                    <Rail
                        facts={[
                            factOf('Kind', kindName(report)),
                            factOf('Category', categoryWord(App.Category)),
                            factOf('Declared in', null),
                            factOf('Machines', null),
                            ...(users.length === 0
                                ? []
                                : [factOf('Used by', <ItemChips {...props} names={users} />)]),
                            factOf('Comes from', null),
                        ]}
                        name={App.Name}
                    />
                }
            />
        </>
    );
}
