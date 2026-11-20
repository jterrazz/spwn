import type { CoreJournalKind, CoreJournalReadLine } from '@jterrazz/station-api';

import { sentence } from '../../../domain/format';
import { machineRoute } from '../../navigation/route';
import type { PlainPage, Route } from '../../navigation/route';
import { countOf } from '../../words/engine-words';

function onShelf(page: PlainPage): (name: string) => Route {
    return (item) => ({ item, page });
}

const ROUTE_OF_KIND: Readonly<Record<CoreJournalKind, (name: string) => Route>> = {
    app: onShelf('catalog-apps'),
    image: () => ({ page: 'storage' }),
    machine: (name) => machineRoute(name),
    setting: onShelf('catalog-settings'),
    skill: onShelf('catalog-skills'),
};

export function routeOfLine({ Subject }: Pick<CoreJournalReadLine, 'Subject'>): Route | undefined {
    if (Subject === null || Subject.Gone) {
        return undefined;
    }
    return ROUTE_OF_KIND[Subject.Kind](Subject.Name);
}

export function changeOf({
    Upgrade,
    Detail,
}: Pick<CoreJournalReadLine, 'Detail' | 'Upgrade'>): string {
    return Upgrade === null ? Detail : `${Upgrade.From} → ${Upgrade.To}`;
}

function placeOf(on: readonly string[]): string {
    const [only] = on;
    return on.length === 1 && only !== undefined ? only : countOf('machine', on.length);
}

export function endOf({
    By,
    Purpose,
    On,
}: Pick<CoreJournalReadLine, 'By' | 'On' | 'Purpose'>): string {
    const on = On ?? [];
    if (By === '') {
        return on.length === 0 ? '' : placeOf(on);
    }
    return sentence([`by ${By}`, Purpose]);
}

export function keyOfLine({
    At,
    Text,
    On,
}: Pick<CoreJournalReadLine, 'At' | 'On' | 'Text'>): string {
    return `${At}/${Text}/${(On ?? []).join(',')}`;
}
