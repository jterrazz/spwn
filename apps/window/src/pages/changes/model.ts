import { countOf } from '../../engine/engine-words';
import type { JournalLine, JournalSubjectKind } from '../../engine/kept-journal/journal';
import { sentence } from '../../model/format';
import { machineRoute } from '../../shell/route';
import type { PlainPage, Route } from '../../shell/route';

function onShelf(page: PlainPage): (name: string) => Route {
    return (item) => ({ item, page });
}

const ROUTE_OF_KIND: Readonly<Record<JournalSubjectKind, (name: string) => Route>> = {
    app: onShelf('catalog-apps'),
    machine: (name) => machineRoute(name),
    setting: onShelf('catalog-settings'),
    skill: onShelf('catalog-skills'),
};

export function routeOfLine({ Subject }: Pick<JournalLine, 'Subject'>): Route | undefined {
    return Subject === null ? undefined : ROUTE_OF_KIND[Subject.Kind](Subject.Name);
}

export function changeOf({ Upgrade, Detail }: Pick<JournalLine, 'Detail' | 'Upgrade'>): string {
    return Upgrade === null ? Detail : `${Upgrade.From} → ${Upgrade.To}`;
}

function placeOf(on: readonly string[]): string {
    const [only] = on;
    return on.length === 1 && only !== undefined ? only : countOf('machine', on.length);
}

export function endOf({ By, Purpose, On }: Pick<JournalLine, 'By' | 'On' | 'Purpose'>): string {
    const on = On ?? [];
    if (By === '') {
        return on.length === 0 ? '' : placeOf(on);
    }
    return sentence([`by ${By}`, Purpose]);
}

export function keyOfLine({ At, Text }: Pick<JournalLine, 'At' | 'Text'>): string {
    return `${At}/${Text}`;
}
