import { Icon } from './icon';
import type { IconName } from './icon';

import styles from './icon-button.module.css';

export type IconButtonProps = {
    icon: IconName;
    label: string;
    shortcut?: string;
    onClick: () => void;
    disabled?: boolean;
    pressed?: boolean | undefined;
};

export function IconButton({
    icon,
    label,
    shortcut,
    onClick,
    disabled = false,
    pressed,
}: IconButtonProps) {
    return (
        <button
            aria-keyshortcuts={shortcut}
            aria-label={label}
            aria-pressed={pressed}
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
