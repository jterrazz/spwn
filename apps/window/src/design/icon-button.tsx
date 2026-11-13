import { Icon } from './icon';
import type { IconName } from './icon';

import styles from './icon-button.module.css';

export type IconButtonProps = {
    icon: IconName;
    label: string;
    shortcut?: string;
    onClick: () => void;
    disabled?: boolean;
};

export function IconButton({ icon, label, shortcut, onClick, disabled = false }: IconButtonProps) {
    return (
        <button
            aria-keyshortcuts={shortcut}
            aria-label={label}
            className={styles.button}
            disabled={disabled}
            onClick={onClick}
            title={(shortcut ?? '') === '' ? label : `${label} (${shortcut})`}
            type="button"
        >
            <Icon name={icon} />
        </button>
    );
}
