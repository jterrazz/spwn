import type { ReactNode } from 'react';

import type { Tone } from '../tone';
import { Icon } from './icon';
import type { IconName } from './icon';
import { StateDot } from './state-dot';

import styles from './nav-item.module.css';

export type NavItemProps = {
    label: string;
    current: boolean;
    onSelect: () => void;
    icon?: IconName;
    lead?: ReactNode;
    indent?: boolean;
    count?: undefined | { value: number; urgent: boolean };
    state?: undefined | { tone: Tone; word: string };
};

export function NavItem({
    label,
    current,
    onSelect,
    icon,
    lead,
    indent = false,
    count,
    state,
}: NavItemProps) {
    return (
        <button
            aria-current={current ? 'page' : undefined}
            className={`${styles.item} ${indent ? styles.indent : ''}`}
            onClick={onSelect}
            type="button"
        >
            {icon ? <Icon name={icon} /> : lead}
            <span className={styles.label}>{label}</span>
            {count ? (
                <span className={`${styles.count} ${count.urgent ? styles.urgent : ''}`}>
                    {count.value}
                </span>
            ) : null}
            {state ? <StateDot tone={state.tone} word={state.word} /> : null}
        </button>
    );
}
