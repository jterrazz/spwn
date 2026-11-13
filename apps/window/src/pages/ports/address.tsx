import { useState } from 'react';

import { CopyButton } from '../../design/copy-button';
import { Button, Value } from '../../design/index';
import { QrCode } from '../../design/qr-code';
import type { Openable } from '../../engine/opener';
import { OpenButton } from '../kit/opens';

import styles from './ports.module.css';

const NO_WEB_ADDRESS = 'The engine gives no web address for this port';

type AddressProps = { url: string; qr: boolean; label?: string; opens?: Openable | undefined };

export function Address({ label = '', url, qr, opens }: AddressProps) {
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
                {opens === undefined ? null : (
                    <OpenButton
                        held={NO_WEB_ADDRESS}
                        label="Open"
                        openable={opens}
                        variant="link"
                    />
                )}
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
