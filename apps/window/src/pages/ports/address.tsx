import { useState } from 'react';

import { CopyButton } from '../../design/copy-button';
import { Button, Value } from '../../design/index';
import { QrCode } from '../../design/qr-code';

import styles from './ports.module.css';

const OPEN_NEEDS = 'Opening an address needs a browser outside the window';

type AddressProps = { url: string; qr: boolean; label?: string; opens?: boolean };

export function Address({ label = '', url, qr, opens = false }: AddressProps) {
    const [open, setOpen] = useState(false);
    if (url === '') {
        return null;
    }
    return (
        <div className={styles.address}>
            {label === '' ? null : <span className={styles.label}>{label}</span>}
            <Value>{url}</Value>
            <span className={styles.links}>
                <CopyButton name={`Copy ${url}`} text={url} />
                {opens ? (
                    <Button disabled title={OPEN_NEEDS} variant="link">
                        Open
                    </Button>
                ) : null}
                {qr ? (
                    <Button
                        onClick={() => {
                            setOpen(!open);
                        }}
                        variant="link"
                    >
                        {open ? 'Hide QR code' : 'QR code'}
                    </Button>
                ) : null}
            </span>
            {open ? <QrCode text={url} /> : null}
        </div>
    );
}
