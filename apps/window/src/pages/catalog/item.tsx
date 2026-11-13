import type { AppsReport, MachinesMacView, SettingsRow } from '@jterrazz/station-api';

import { Verdict } from '../../design/index';
import { isSkill } from '../../engine/provisional';
import { useReach, useRead, useWatch } from '../../engine/react';
import type { Reading as Answer } from '../../engine/react';
import { ownState } from '../../engine/shown';
import type { Route } from '../../shell/route';
import { Absent, Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { AppItem } from './item-app';
import type { Go } from './item-parts';
import { SettingItem } from './item-setting';
import type { Shelf } from './model';

function routeOfName(reports: readonly AppsReport[], settings: readonly SettingsRow[]) {
    return (name: string): Route => {
        if (settings.some((row) => row.Name === name)) {
            return { item: name, page: 'catalog-settings' };
        }
        const report = reports.find((one) => one.App.Name === name);
        const skill = report !== undefined && isSkill(report.App.Category);
        return { item: name, page: skill ? 'catalog-skills' : 'catalog-apps' };
    };
}

function NotInTheCatalog({ name }: { name: string }) {
    return (
        <Verdict
            line="The catalog the engine answers holds no package of that name"
            size="page"
            title={`No ${name} in the catalog.`}
            {...ownState('not-given')}
        />
    );
}

function Item({
    shelf,
    name,
    go,
    macs,
    reports,
    settings,
}: {
    shelf: Shelf;
    name: string;
    go: Go;
    macs: readonly MachinesMacView[];
    reports: readonly AppsReport[];
    settings: readonly SettingsRow[];
}) {
    const matrix = useRead('settings.matrix', {});
    const props = { go, macs, routeOf: routeOfName(reports, settings) };
    if (shelf === 'settings') {
        const row = settings.find((one) => one.Name === name);
        return row ? (
            <SettingItem
                {...props}
                matrix={matrix.kind === 'read' ? matrix.out : undefined}
                row={row}
            />
        ) : (
            <NotInTheCatalog name={name} />
        );
    }
    const report = reports.find((one) => one.App.Name === name);
    return report ? (
        <AppItem {...props} report={report} reports={reports} />
    ) : (
        <NotInTheCatalog name={name} />
    );
}

function Waiting({ name, reading }: { name: string; reading: Answer<unknown> }) {
    if (reading.kind === 'failed') {
        return <Unanswered error={reading.error} size="page" />;
    }
    return <Reading size="page" what={name} />;
}

export function ItemPage({
    shelf,
    name,
    go,
    macs,
}: {
    shelf: Shelf;
    name: string;
    go: Go;
    macs: readonly MachinesMacView[];
}) {
    const reach = useReach();
    const apps = useWatch('apps.list', {});
    const settings = useWatch('settings.list', {});
    if (reach.kind === 'unanswered') {
        return (
            <Page>
                <Absent reach={reach} />
            </Page>
        );
    }
    const needed = shelf === 'settings' ? settings : apps;
    if (needed.kind !== 'read') {
        return (
            <Page>
                <Waiting name={name} reading={needed} />
            </Page>
        );
    }
    return (
        <Page>
            <Item
                go={go}
                macs={macs}
                name={name}
                reports={apps.kind === 'read' ? (apps.out ?? []) : []}
                settings={settings.kind === 'read' ? (settings.out ?? []) : []}
                shelf={shelf}
            />
        </Page>
    );
}
