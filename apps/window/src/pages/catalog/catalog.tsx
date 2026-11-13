import type { AppsReport, SettingsRow } from '@jterrazz/station-api';

import { ChoiceCard, Verdict } from '../../design/index';
import { useReach, useWatch } from '../../engine/react';
import type { Reading } from '../../engine/react';
import { ownState } from '../../engine/shown';
import type { Route } from '../../shell/route';
import { Absent } from '../kit/readings';
import { Page } from '../page';
import { appsOn, categoriesOf, packageOfApp, packageOfSetting } from './model';
import type { Package, Shelf } from './model';
import { categoryWord, SHELVES } from './shelves';

import styles from './catalog.module.css';

type Go = (to: Route) => void;

function useCatalog() {
    const apps = useWatch('apps.list', {});
    const settings = useWatch('settings.list', {});
    return { apps, settings };
}

function packagesOf(
    shelf: Shelf,
    apps: Reading<AppsReport[] | null>,
    settings: Reading<null | SettingsRow[]>,
): readonly Package[] | undefined {
    if (shelf === 'settings') {
        return settings.kind === 'read' ? (settings.out ?? []).map(packageOfSetting) : undefined;
    }
    return apps.kind === 'read' ? appsOn(shelf, apps.out ?? []).map(packageOfApp) : undefined;
}

function Card({
    shelf,
    packages,
    go,
}: {
    shelf: Shelf;
    packages: readonly Package[] | undefined;
    go: Go;
}) {
    const { description, icon, page, title } = SHELVES[shelf];
    return (
        <ChoiceCard
            count={
                packages === undefined ? 'Not answered yet' : `${packages.length} in the catalog`
            }
            description={description}
            foot={
                packages === undefined ? '' : categoriesOf(packages).map(categoryWord).join(' · ')
            }
            icon={icon}
            onChoose={() => {
                go({ page });
            }}
            title={title}
        />
    );
}

function lineOf(shelves: readonly (readonly Package[] | undefined)[]): string {
    const read = shelves.filter((packages) => packages !== undefined);
    const count = read.reduce((sum, packages) => sum + packages.length, 0);
    return read.length === shelves.length
        ? `Everything station knows how to install or set, ${count} packages · choose what to browse`
        : 'Everything station knows how to install or set · choose what to browse';
}

const SHELF_ORDER = ['apps', 'skills', 'settings'] as const;

export function CatalogPage({ go }: { go: Go }) {
    const reach = useReach();
    const { apps, settings } = useCatalog();
    if (reach.kind === 'unanswered') {
        return (
            <Page>
                <Absent reach={reach} />
            </Page>
        );
    }
    const shelves = SHELF_ORDER.map((shelf) => packagesOf(shelf, apps, settings));
    return (
        <Page>
            <Verdict
                line={lineOf(shelves)}
                size="page"
                title="Catalog."
                {...ownState('not-given')}
            />
            <div className={styles.cards}>
                {SHELF_ORDER.map((shelf, index) => (
                    <Card go={go} key={shelf} packages={shelves[index]} shelf={shelf} />
                ))}
            </div>
        </Page>
    );
}
