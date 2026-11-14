import type { AppsReport, SettingsMatrixRow, SettingsRow } from '@jterrazz/station-api';

import { AppIcon, Row, StatePill, Value } from '../../design/index';
import { countOf } from '../../engine/engine-words';
import { useReach, useRead, useWatch } from '../../engine/react';
import type { Reading } from '../../engine/react';
import { appUpdate, kindChipWord } from '../../engine/shown';
import { APP_STATE_TONES } from '../../engine/tones';
import { APP_STATE_WORDS } from '../../engine/words';
import { sentence } from '../../model/format';
import type { Route } from '../../shell/route';
import { Absent, NoVerdict, Reading as ReadingVerdict, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { Listing } from './listing';
import { appsOn, packageOfApp, packageOfSetting, settingsInCatalog } from './model';
import type { Shelf } from './model';
import { SHELVES } from './shelves';

import styles from './catalog.module.css';

const ABOUT =
    'Everything station knows how to install or set: apps, agent skills and settings, each one a package';

function kindOf(word: string): { tag?: string; tagLook?: 'type' } {
    return word === '' ? {} : { tag: word, tagLook: 'type' };
}

type Open = (name: string) => void;

function AppRow({ report, open }: { report: AppsReport; open: Open }) {
    const { App } = report;
    const update = appUpdate(report);
    const version = report.Version === '' ? null : <Value look="raised">{report.Version}</Value>;
    return (
        <Row
            cells={[App.Description]}
            {...(App.Kind === undefined ? {} : kindOf(kindChipWord(App.Kind)))}
            lead={<AppIcon kind={App.Kind} name={App.Name} />}
            onOpen={() => {
                open(App.Name);
            }}
            size="slim"
            state={
                update === null
                    ? { tone: APP_STATE_TONES[report.State], word: APP_STATE_WORDS[report.State] }
                    : null
            }
            template="catalog"
            title={App.Name}
            value={
                update === null ? (
                    version
                ) : (
                    <span className={styles.update}>
                        <StatePill {...update} />
                        {version}
                    </span>
                )
            }
            valueSize={update === null ? 'short' : 'long'}
        />
    );
}

function settingValue(row: SettingsRow, held: SettingsMatrixRow | undefined): null | string {
    if (held !== undefined) {
        return `${held.OK} of ${countOf('machine', held.Of)} OK`;
    }
    const reading = sentence([row.Value, row.Why]);
    return reading === '' ? null : reading;
}

function SettingRow({
    row,
    held,
    open,
}: {
    row: SettingsRow;
    held: SettingsMatrixRow | undefined;
    open: Open;
}) {
    return (
        <Row
            cells={[row.Package.Description]}
            {...kindOf(kindChipWord('setting'))}
            lead={<AppIcon kind="setting" name={row.Name} />}
            onOpen={() => {
                open(row.Name);
            }}
            size="slim"
            template="catalog"
            title={row.Name}
            value={settingValue(row, held)}
            valueSize="long"
        />
    );
}

function Waiting({ reading, what }: { reading: Reading<unknown>; what: string }) {
    const reach = useReach();
    if (reach.kind === 'unanswered') {
        return <Absent reach={reach} />;
    }
    if (reading.kind === 'failed') {
        return <Unanswered error={reading.error} size="page" />;
    }
    return <ReadingVerdict size="page" what={what} />;
}

function SettingsShelf({ open }: { open: Open }) {
    const reach = useReach();
    const settings = useWatch('settings.list', {});
    const matrix = useRead('settings.matrix', {});
    if (settings.kind !== 'read' || reach.kind === 'unanswered') {
        return (
            <Page>
                <Waiting reading={settings} what="Settings" />
            </Page>
        );
    }
    const held = new Map(
        matrix.kind === 'read' ? (matrix.out.Rows ?? []).map((row) => [row.Name, row]) : [],
    );
    return (
        <Page>
            <NoVerdict line={ABOUT} size="page" />
            <Listing
                items={settingsInCatalog(settings.out ?? [])}
                packageOf={packageOfSetting}
                renderRow={(row) => (
                    <SettingRow held={held.get(row.Name)} key={row.Name} open={open} row={row} />
                )}
                title="Settings"
            />
            {matrix.kind === 'failed' ? <Unanswered error={matrix.error} size="tab" /> : null}
        </Page>
    );
}

function AppsShelf({ shelf, open }: { shelf: 'apps' | 'skills'; open: Open }) {
    const reach = useReach();
    const apps = useWatch('apps.list', {});
    const { title } = SHELVES[shelf];
    if (apps.kind !== 'read' || reach.kind === 'unanswered') {
        return (
            <Page>
                <Waiting reading={apps} what={title} />
            </Page>
        );
    }
    return (
        <Page>
            <NoVerdict line={ABOUT} size="page" />
            <Listing
                items={appsOn(shelf, apps.out ?? [])}
                packageOf={packageOfApp}
                renderRow={(report) => <AppRow key={report.App.Name} open={open} report={report} />}
                title={title}
            />
        </Page>
    );
}

export function ShelfPage({ shelf, go }: { shelf: Shelf; go: (to: Route) => void }) {
    const open = (item: string) => {
        go({ item, page: SHELVES[shelf].page });
    };
    return shelf === 'settings' ? (
        <SettingsShelf open={open} />
    ) : (
        <AppsShelf open={open} shelf={shelf} />
    );
}
