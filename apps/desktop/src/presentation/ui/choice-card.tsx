import { Icon } from './icon';
import type { IconName } from './icon';

import styles from './choice-card.module.css';

export type ChoiceCardProps = {
    icon: IconName;
    title: string;
    count: string;
    description: string;
    foot: string;
    onChoose: () => void;
};

export function ChoiceCard({ icon, title, count, description, foot, onChoose }: ChoiceCardProps) {
    return (
        <button className={styles.card} onClick={onChoose} type="button">
            <span className={styles.icon}>
                <Icon name={icon} />
            </span>
            <span className={styles.title}>{title}</span>
            <span className={styles.count}>{count}</span>
            <span className={styles.description}>{description}</span>
            <span className={styles.foot}>{foot}</span>
        </button>
    );
}
