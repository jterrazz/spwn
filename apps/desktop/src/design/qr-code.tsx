import { qrOf } from './qr';

import styles from './qr-code.module.css';

const QUIET = 4;

export function QrCode({ text }: { text: string }) {
    const matrix = qrOf(text);
    if (!matrix) {
        return <p className={styles.long}>Too long for a QR code: copy the address instead.</p>;
    }
    const side = matrix.length + QUIET * 2;
    const path = matrix
        .flatMap((row, y) =>
            row.map((dark, x) => (dark ? `M${x + QUIET} ${y + QUIET}h1v1h-1z` : '')),
        )
        .join('');
    return (
        <svg
            aria-label={`QR code of ${text}`}
            className={styles.code}
            role="img"
            shapeRendering="crispEdges"
            viewBox={`0 0 ${side} ${side}`}
        >
            <rect fill="#fff" height={side} width={side} />
            <path d={path} fill="#000" />
        </svg>
    );
}
