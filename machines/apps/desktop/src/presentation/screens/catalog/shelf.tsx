import type {
    AppsMatrixRow,
    AppsReport,
    CoreVerdictShelf,
    SettingsMatrixRow,
    SettingsRow,
} from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import {
    appsOn,
    carriedApps,
    carriedSettings,
    settingsInCatalog,
} from '../../../domain/catalog/carried';
import { sentence } from '../../../domain/format';
import { useReach, useRead, useWatch } from '../../dependencies/dependencies';
import type { Reading } from '../../dependencies/dependencies';
import { Absent, Unanswered, VerdictHead } from '../../kit/readings';
import { SkeletonHead } from '../../kit/skeleton';
import type { Route } from '../../navigation/route';
import { AppIcon, Row, StatePill, Value } from '../../ui/index';
import { Page } from '../../ui/page';
import { countOf } from '../../words/engine-words';
import { appUpdate, kindChipWord } from '../../words/shown';
import { APP_STATE_TONES } from '../../words/tones';
import { APP_STATE_WORDS } from '../../words/words';
import { Listing } from './listing';
import type { Scope } from './listing';
import { packageOfApp, packageOfSetting } from './model';
import type { Shelf } from './model';
import { SHELVES } from './shelves';

import styles from './catalog.module.css';

function headOf<Out>(
    matrix: Reading<Out>,
    shelfOf: (out: Out) => CoreVerdictShelf,
    what: string,
): (scope: Scope) => ReactNode {
    return (scope) => {
        if (matrix.kind === 'failed') {
            return <Unanswered error={matrix.error} size="page" />;
        }
        if (matrix.kind !== 'read') {
            return <SkeletonHead size="page" what={what} />;
        }
        const shelf = shelfOf(matrix.out);
        return (
            <VerdictHead size="page" verdict={scope === 'carried' ? shelf.Mine : shelf.Catalog} />
        );
    };
}

function kindOf(word: string): { tag?: string; tagLook?: 'type' } {
    return word === '' ? {} : { tag: word, tagLook: 'type' };
}

type Open = (name: string) => void;

function AppRow({
    report,
    carry,
    open,
}: {
    report: AppsReport;
    carry: AppsMatrixRow | undefined;
    open: Open;
}) {
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
            {...(carry === undefined
                ? {}
                : { sentence: `${String(carry.Carry)} of ${countOf('machine', carry.Of)}` })}
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
    return <SkeletonHead size="page" what={what} />;
}

function SettingsShelf({ open }: { open: Open }) {
    const settings = useWatch('settings.list', {});
    const matrix = useRead('settings.matrix', {});
    const size = useCatalogSize();
    if (settings.kind !== 'read') {
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
            <Listing
                carried={
                    matrix.kind === 'read'
                        ? carriedSettings(settings.out ?? [], matrix.out.Rows ?? [])
                        : undefined
                }
                head={headOf(matrix, (out) => out.Verdict, 'Settings')}
                items={settingsInCatalog(settings.out ?? [])}
                packageOf={packageOfSetting}
                renderRow={(row) => (
                    <SettingRow held={held.get(row.Name)} key={row.Name} open={open} row={row} />
                )}
                size={size}
                title={SHELVES.settings.title}
            />
        </Page>
    );
}

// value: the catalog's size is a count of the wire's own rows: every app and skill the engine lists, and every
// value: setting that is a package.
function useCatalogSize(): number | undefined {
    const apps = useWatch('apps.list', {});
    const settings = useWatch('settings.list', {});
    if (apps.kind !== 'read' || settings.kind !== 'read') {
        return undefined;
    }
    return (apps.out ?? []).length + settingsInCatalog(settings.out ?? []).length;
}

function AppsShelf({ shelf, open }: { shelf: 'apps' | 'skills'; open: Open }) {
    const apps = useWatch('apps.list', {});
    const matrix = useRead('apps.matrix', {});
    const size = useCatalogSize();
    const { title } = SHELVES[shelf];
    if (apps.kind !== 'read') {
        return (
            <Page>
                <Waiting reading={apps} what={title} />
            </Page>
        );
    }
    const held = new Map(
        matrix.kind === 'read' ? (matrix.out.Rows ?? []).map((row) => [row.Name, row]) : [],
    );
    return (
        <Page>
            <Listing
                carried={
                    matrix.kind === 'read'
                        ? carriedApps(shelf, apps.out ?? [], matrix.out.Rows ?? [])
                        : undefined
                }
                head={headOf(
                    matrix,
                    (out) => (shelf === 'skills' ? out.Skills : out.Verdict),
                    title,
                )}
                items={appsOn(shelf, apps.out ?? [])}
                packageOf={packageOfApp}
                renderRow={(report) => (
                    <AppRow
                        carry={held.get(report.App.Name)}
                        key={report.App.Name}
                        open={open}
                        report={report}
                    />
                )}
                size={size}
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
