import { useState } from 'react';
import type { ReactNode } from 'react';

import { Caption, SearchField, Switch, Value } from '../../design/index';
import { countOf } from '../../engine/engine-words';
import { categoriesOf, countIn, EVERY_CATEGORY, matches } from './model';
import type { Package } from './model';
import { categoryWord } from './shelves';

import styles from './catalog.module.css';

function Filters({
    title,
    packages,
    category,
    setCategory,
    query,
    setQuery,
    scope,
}: {
    title: string;
    packages: readonly Package[];
    category: string;
    setCategory: (category: string) => void;
    query: string;
    setQuery: (query: string) => void;
    scope: ReactNode;
}) {
    return (
        <>
            <div className={styles.bar}>
                <SearchField
                    label="Search the catalog"
                    onChange={setQuery}
                    placeholder="Search the catalog"
                    value={query}
                />
                {scope}
            </div>
            <Switch
                label="Category"
                onChange={setCategory}
                options={[
                    {
                        count: packages.length,
                        id: EVERY_CATEGORY,
                        label: `All ${title.toLowerCase()}`,
                    },
                    ...categoriesOf(packages).map((name) => ({
                        count: countIn(packages, name),
                        id: name,
                        label: categoryWord(name),
                    })),
                ]}
                value={category}
                wrap
            />
        </>
    );
}

function Group<T>({
    category,
    rows,
    renderRow,
}: {
    category: string;
    rows: readonly T[];
    renderRow: (item: T) => ReactNode;
}) {
    return (
        <div className={styles.group}>
            <Caption>
                <Value look="raised">{categoryWord(category)}</Value>
                <span>{rows.length}</span>
            </Caption>
            {rows.map(renderRow)}
        </div>
    );
}

type Scope = 'carried' | 'every';

// value: the catalog's size and the packages some machine carries are counts of the wire's own rows; a listing given
// value: `carried` offers the choice between them and everything, and says so under the list while it shows them.
function ScopeSwitch({
    carried,
    every,
    scope,
    setScope,
}: {
    carried: number | undefined;
    every: number;
    scope: Scope;
    setScope: (scope: Scope) => void;
}) {
    if (carried === undefined) {
        return null;
    }
    return (
        <Switch
            label="Show"
            onChange={setScope}
            options={[
                { count: carried, id: 'carried', label: 'On your machines' },
                { count: every, id: 'every', label: 'Everything' },
            ]}
            value={scope}
        />
    );
}

function footOf(size: number | undefined): string | undefined {
    return size === undefined
        ? undefined
        : `Only what your machines carry is shown. The catalog holds ${countOf('package', size)}.`;
}

export function Listing<T>({
    title,
    items,
    carried,
    size,
    packageOf,
    renderRow,
}: {
    title: string;
    items: readonly T[];
    carried?: readonly T[] | undefined;
    size?: number | undefined;
    packageOf: (item: T) => Package;
    renderRow: (item: T) => ReactNode;
}) {
    const [category, setCategory] = useState(EVERY_CATEGORY);
    const [query, setQuery] = useState('');
    const [scope, setScope] = useState<Scope>('carried');
    const carrying = carried !== undefined && scope === 'carried';
    const listed = carrying ? carried : items;
    const packages = listed.map(packageOf);
    const shown = listed.filter((item) => matches(packageOf(item), category, query));
    const groups = categoriesOf(shown.map(packageOf));
    const foot = carrying ? footOf(size) : undefined;
    return (
        <>
            <Filters
                category={category}
                packages={packages}
                query={query}
                scope={
                    <ScopeSwitch
                        carried={carried?.length}
                        every={items.length}
                        scope={scope}
                        setScope={setScope}
                    />
                }
                setCategory={setCategory}
                setQuery={setQuery}
                title={title}
            />
            <section aria-label={title} className={styles.list}>
                {groups.length === 0 ? (
                    <p className={styles.none}>Nothing matches.</p>
                ) : (
                    groups.map((name) => (
                        <Group
                            category={name}
                            key={name}
                            renderRow={renderRow}
                            rows={shown.filter((item) => packageOf(item).category === name)}
                        />
                    ))
                )}
                {foot === undefined ? null : <p className={styles.foot}>{foot}</p>}
            </section>
        </>
    );
}
