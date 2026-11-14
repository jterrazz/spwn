import { useState } from 'react';

import { CopyButton } from '../../design/copy-button';
import { Button } from '../../design/index';
import { QrCode } from '../../design/qr-code';
import { SEPARATOR } from '../../design/separator';
import type { Openable } from '../../engine/opener';
import { OpenButton } from '../kit/opens';

import styles from './ports.module.css';

type Opens = { openable: Openable | undefined; held: string };

type AddressProps = {
    url: string;
    qr: boolean;
    label?: string;
    opens?: Opens | undefined;
};

export function Address({ label = '', url, qr, opens }: AddressProps) {
    const [open, setOpen] = useState(false);
    if (url === '') {
        return null;
    }
    return (
        <div className={styles.address}>
            <span className={styles.links}>
                {label === '' ? null : `${label}${SEPARATOR}`}
                <CopyButton name={`Copy ${url}`} text={url} variant="inline" />
                {opens === undefined ? null : (
                    <>
                        {SEPARATOR}
                        <OpenButton
                            held={opens.held}
                            label="Open"
                            openable={opens.openable}
                            variant="inline"
                        />
                    </>
                )}
                {qr ? (
                    <>
                        {SEPARATOR}
                        <Button
                            onClick={() => {
                                setOpen(!open);
                            }}
                            variant="inline"
                        >
                            {open ? 'Hide QR code' : 'QR code'}
                        </Button>
                    </>
                ) : null}
            </span>
            {open ? <QrCode text={url} /> : null}
        </div>
    );
}
