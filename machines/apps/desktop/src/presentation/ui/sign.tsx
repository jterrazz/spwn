import drawing from '../../../src-tauri/icons/sizes/spwn-icon-32.svg';

import styles from './sign.module.css';

export type SignProps = { size?: 'large' | 'small' };

// invariant: the sign is the app icon's own drawing, never one of the window's, so its geometry has one owner.
export function Sign({ size = 'small' }: SignProps) {
    return (
        <img
            alt=""
            className={`${styles.sign} ${size === 'large' ? styles.large : ''}`}
            draggable={false}
            src={drawing}
        />
    );
}
