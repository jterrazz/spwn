import type { PortsSharedPort, Target } from '@jterrazz/station-api';

import { Chip, TypeTag, Value } from '../../design/index';
import { shareToWord } from '../../engine/shown';
import { SHARE_STATE_TONES } from '../../engine/tones';
import { SHARE_STATE_WORDS } from '../../engine/words';
import { sentence } from '../../model/format';
import { GestureButton, GestureRow } from '../kit/gestures';
import { Address } from './address';
import { usePortStop } from './port-gestures';

import styles from './ports.module.css';

export function addressOf(share: PortsSharedPort): string {
    return share.urls.public === '' ? share.urls.local : share.urls.public;
}

export function SharedRow({
    share,
    target,
    compact = false,
}: {
    share: PortsSharedPort;
    target: Target | undefined;
    compact?: boolean;
}) {
    const to = share.to ?? '';
    const label = share.label ?? '';
    const stop = usePortStop({ port: share.guest_port, target, vm: share.vm });
    const stopButton = (
        <GestureButton
            gesture={stop}
            label="Stop sharing"
            title={`Stop sharing ${share.vm}:${share.guest_port}?`}
            variant="link"
        />
    );
    const row = {
        gesture: stop,
        lead: <Chip>{`:${share.guest_port}`}</Chip>,
        leadSize: 'value',
        state: { tone: SHARE_STATE_TONES[share.state], word: SHARE_STATE_WORDS[share.state] },
        title: label === '' ? share.process : label,
    } as const;
    if (compact) {
        return (
            <section aria-label={`Port ${share.guest_port}`} className={styles.shared}>
                <GestureRow
                    {...row}
                    {...(to === '' ? {} : { tag: shareToWord(to) })}
                    sentence={<Value>{addressOf(share)}</Value>}
                    trail={stopButton}
                />
            </section>
        );
    }
    return (
        <section aria-label={`Port ${share.guest_port}`} className={styles.shared}>
            <GestureRow
                {...row}
                sentence={sentence([label === '' ? undefined : share.process, share.detail])}
                trail={
                    <span className={styles.trail}>
                        {to === '' ? null : <TypeTag>{shareToWord(to)}</TypeTag>}
                        {stopButton}
                    </span>
                }
                trailSize="wide"
            />
            <Address opens={share.http} qr={share.urls.public !== ''} url={addressOf(share)} />
        </section>
    );
}
