import type { AppsReport, SettingsMatrixRow, SettingsRow } from '@jterrazz/station-api';

import { AppIcon, Chip, Row } from '../../design/index';
import { useReach, useRead, useWatch } from '../../engine/react';
import type { Reading } from '../../engine/react';
import { appUpdate } from '../../engine/shown';
import { APP_STATE_TONES } from '../../engine/tones';
import { APP_STATE_WORDS, CATALOG_KIND_WORDS } from '../../engine/words';
import { sentence } from '../../model/format';
import { Absent, NoVerdict, Reading as ReadingVerdict, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { Listing } from './listing';
import { appsOn, packageOfApp, packageOfSetting } from './model';
import type { Shelf } from './model';
import { SHELVES } from './shelves';

const ABOUT =
    'Everything station knows how to install or set: apps, agent skills and settings, each one a package';

function kindOf(word: string): null | string {
    return word === '' ? null : word;
}

function AppRow({ report }: { report: AppsReport }) {
    const { App } = report;
    const update = appUpdate(report);
    return (
        <Row
            cells={[sentence([App.Description, report.Detail])]}
            kind={App.Kind === undefined ? null : kindOf(CATALOG_KIND_WORDS[App.Kind])}
            lead={<AppIcon kind={App.Kind} name={App.Name} />}
            size="slim"
            state={
                update ?? {
                    tone: APP_STATE_TONES[report.State],
                    word: APP_STATE_WORDS[report.State],
                }
            }
            template="catalog"
            title={App.Name}
            value={report.Version === '' ? null : <Chip>{report.Version}</Chip>}
        />
    );
}

function SettingRow({ row, held }: { row: SettingsRow; held: SettingsMatrixRow | undefined }) {
    return (
        <Row
            cells={[sentence([row.Package.Description, row.Value, row.Why])]}
            kind={CATALOG_KIND_WORDS.setting}
            lead={<AppIcon kind="setting" name={row.Name} />}
            size="slim"
            template="catalog"
            title={row.Name}
            value={
                held === undefined
                    ? null
                    : `${held.OK} of ${held.Of} ${held.Of === 1 ? 'machine' : 'machines'} OK`
            }
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

function SettingsShelf() {
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
                items={settings.out ?? []}
                packageOf={packageOfSetting}
                renderRow={(row) => (
                    <SettingRow held={held.get(row.Name)} key={row.Name} row={row} />
                )}
                title="Settings"
            />
            {matrix.kind === 'failed' ? <Unanswered error={matrix.error} size="tab" /> : null}
        </Page>
    );
}

function AppsShelf({ shelf }: { shelf: 'apps' | 'skills' }) {
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
                renderRow={(report) => <AppRow key={report.App.Name} report={report} />}
                title={title}
            />
        </Page>
    );
}

export function ShelfPage({ shelf }: { shelf: Shelf }) {
    return shelf === 'settings' ? <SettingsShelf /> : <AppsShelf shelf={shelf} />;
}
