import styles from './crumb-dots.module.css';

export type CrumbDot = { id: string; name: string; on: boolean; you: boolean };

// value: the siblings of a crumb as dots, one button each: the current one a bar, one that waits for me a diamond,
// value: each named by its name as handed (its count of needs included by the caller).
export function CrumbDots({
    dots,
    onPick,
}: {
    dots: readonly CrumbDot[];
    onPick: (id: string) => void;
}) {
    return (
        <span className={styles.dots}>
            {dots.map((dot) => (
                <button
                    aria-current={dot.on ? 'true' : undefined}
                    aria-label={dot.name}
                    className={styles.dot}
                    data-on={dot.on ? '' : undefined}
                    data-you={dot.you ? '' : undefined}
                    key={dot.id}
                    onClick={() => {
                        onPick(dot.id);
                    }}
                    title={dot.name}
                    type="button"
                />
            ))}
        </span>
    );
}
