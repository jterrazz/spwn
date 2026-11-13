import { useState } from 'react';
import type { ReactNode } from 'react';

import { Chip, SearchField, Switch } from '../../design/index';
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
}: {
    title: string;
    packages: readonly Package[];
    category: string;
    setCategory: (category: string) => void;
    query: string;
    setQuery: (query: string) => void;
}) {
    return (
        <>
            <SearchField
                label="Search the catalog"
                onChange={setQuery}
                placeholder="Search the catalog"
                value={query}
            />
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
            <p className={styles.head}>
                <Chip>{categoryWord(category)}</Chip>
                <span>{rows.length}</span>
            </p>
            {rows.map(renderRow)}
        </div>
    );
}

export function Listing<T>({
    title,
    items,
    packageOf,
    renderRow,
}: {
    title: string;
    items: readonly T[];
    packageOf: (item: T) => Package;
    renderRow: (item: T) => ReactNode;
}) {
    const [category, setCategory] = useState(EVERY_CATEGORY);
    const [query, setQuery] = useState('');
    const packages = items.map(packageOf);
    const shown = items.filter((item) => matches(packageOf(item), category, query));
    const groups = categoriesOf(shown.map(packageOf));
    return (
        <>
            <Filters
                category={category}
                packages={packages}
                query={query}
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
            </section>
        </>
    );
}
