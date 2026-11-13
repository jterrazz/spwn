import { useState } from 'react';

import { CopyButton } from '../../design/copy-button';
import { Button, Value } from '../../design/index';
import { QrCode } from '../../design/qr-code';
import type { Openable } from '../../engine/opener';
import { OpenButton } from '../kit/opens';

import styles from './ports.module.css';

type Opens = { openable: Openable | undefined; held: string };

type AddressProps = {
    url: string;
    qr: boolean;
    label?: string;
    opens?: Opens | undefined;
    drawn?: 'links' | 'whole';
};

export function Address({ label = '', url, qr, opens, drawn = 'whole' }: AddressProps) {
    const [open, setOpen] = useState(false);
    if (url === '') {
        return null;
    }
    return (
        <div className={styles.address}>
            {label === '' ? null : <span className={styles.label}>{label}</span>}
            {drawn === 'whole' ? <Value>{url}</Value> : null}
            <span className={styles.links}>
                <CopyButton name={`Copy ${url}`} text={url} />
                {opens === undefined ? null : (
                    <OpenButton
                        held={opens.held}
                        label="Open"
                        openable={opens.openable}
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
