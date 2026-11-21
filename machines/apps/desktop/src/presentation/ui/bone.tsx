import styles from './bone.module.css';

export type BoneWidth = 'long' | 'mid' | 'short' | 'wide';

export type BoneProps = { width?: BoneWidth };

// value: a bar the height of one line of the text it stands in, so the box holding it keeps that text's height.
export function Bone({ width = 'mid' }: BoneProps) {
    return <span aria-hidden="true" className={`${styles.bone} ${styles[width]}`} />;
}
