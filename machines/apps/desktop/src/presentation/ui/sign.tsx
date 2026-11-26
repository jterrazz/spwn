import drawing from '../../../src-tauri/icons/sizes/spwn-icon-32.svg';

import styles from './sign.module.css';

// invariant: the sign is the app icon's own drawing, never one of the window's, so its geometry has one owner.
export function Sign() {
    return <img alt="" className={styles.sign} draggable={false} src={drawing} />;
}
