import { useLayoutEffect, useRef } from 'react';
import type { ReactNode } from 'react';

import type { CoreLine } from '../../../application/ports/core.port';
import type { CoreKind } from '../../../application/use-cases/core-flow';
import { INSTALL_LINE } from '../../../domain/core';
import { CopyButton } from '../../ui/copy-button';
import { Icon } from '../../ui/icon';
import { StatePill } from '../../ui/index';
import { shownOrWithheld } from '../../ui/secret';
import { coreState } from '../../words/shown';

import styles from './core.module.css';

export function Mark() {
    return (
        <svg aria-hidden="true" className={styles.mark} fill="currentColor" viewBox="0 0 48 44">
            <circle cx="10" cy="35" r="8" />
            <circle cx="31.86" cy="23.37" r="5.6" />
            <circle cx="42.81" cy="5.85" r="3.9" />
        </svg>
    );
}

export function Pill({ kind }: { kind: CoreKind }) {
    return (
        <span className={styles.pill}>
            <StatePill {...coreState(kind)} />
        </span>
    );
}

export function Head({ title, children }: { title: string; children: ReactNode }) {
    return (
        <div className={styles.head}>
            <h1 className={styles.title}>{title}</h1>
            <p className={styles.lead}>{children}</p>
        </div>
    );
}

export function Mono({ children }: { children: ReactNode }) {
    return <span className={styles.mono}>{children}</span>;
}

export type Row = { key: string; value: ReactNode; place?: string; mono?: boolean };

export function Rows({ label, rows }: { label: string; rows: readonly Row[] }) {
    return (
        <div className={styles.rows}>
            <p className={styles.caption}>{label}</p>
            <ul aria-label={label} className={styles.list}>
                {rows.map((row) => (
                    <li className={styles.row} key={row.key}>
                        <span className={row.mono === true ? styles.keyMono : styles.key}>
                            {row.key}
                        </span>
                        <span className={styles.value}>{row.value}</span>
                        <span className={styles.place}>{row.place ?? ''}</span>
                    </li>
                ))}
            </ul>
        </div>
    );
}

export function Assurances({ said }: { said: readonly string[] }) {
    return (
        <ul aria-label="What the install promises" className={styles.assure}>
            {said.map((one) => (
                <li key={one}>
                    <Icon name="check" />
                    {one}
                </li>
            ))}
        </ul>
    );
}

export function Command() {
    return (
        <div className={styles.command}>
            <code className={styles.code}>
                <span aria-hidden="true" className={styles.prompt}>
                    ${' '}
                </span>
                <span>{INSTALL_LINE}</span>
            </code>
            <CopyButton name="Copy the install command" text={INSTALL_LINE} variant="plain" />
        </div>
    );
}

export function Status({ said, aside }: { said: string; aside?: string }) {
    return (
        <p className={styles.status} role="status">
            <i aria-hidden="true" className={styles.spinner} />
            <span>{said}</span>
            {aside === undefined ? null : <span className={styles.aside}>{aside}</span>}
        </p>
    );
}

export function Progress({ label }: { label: string }) {
    return <div aria-label={label} className={styles.progress} role="progressbar" />;
}

type Numbered = CoreLine & { n: number };

function numbered(lines: readonly CoreLine[]): readonly Numbered[] {
    return lines.map((line, n) => ({ ...line, n }));
}

export function Output({ label, lines }: { label: string; lines: readonly CoreLine[] }) {
    const box = useRef<HTMLDivElement>(null);
    useLayoutEffect(() => {
        const node = box.current;
        if (node && lines.length > 0) {
            node.scrollTop = node.scrollHeight;
        }
    }, [lines]);
    return (
        <div aria-label={label} className={styles.output} ref={box} role="log">
            {numbered(lines).map((line) => (
                <div className={line.stream === 'err' ? styles.err : styles.line} key={line.n}>
                    {shownOrWithheld(line.text)}
                </div>
            ))}
        </div>
    );
}

export function Acts({ children }: { children: ReactNode }) {
    return <div className={styles.acts}>{children}</div>;
}

export function Fine({ children }: { children: ReactNode }) {
    return <span className={styles.fine}>{children}</span>;
}
