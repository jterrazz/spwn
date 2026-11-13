import styles from './code-block.module.css';

export type CodeLine = { text: string; change?: 'added' | 'removed' };

export function CodeBlock({
    lines,
    numbered = false,
    label,
}: {
    lines: readonly CodeLine[];
    numbered?: boolean;
    label: string;
}) {
    return (
        <pre aria-label={label} className={styles.code}>
            {lines.map((line, index) => (
                <div
                    className={`${styles.line} ${line.change ? styles[line.change] : ''}`}
                    key={index}
                >
                    {numbered ? <span className={styles.number}>{index + 1}</span> : null}
                    <span>{line.text}</span>
                </div>
            ))}
        </pre>
    );
}
