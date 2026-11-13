import type { PortsSharedPort, Target } from '@jterrazz/station-api';

import { TypeTag, Value } from '../../design/index';
import { isCalm } from '../../design/state-pill';
import { Openable, shareAddressOf } from '../../engine/opener';
import { shareToWord } from '../../engine/shown';
import { SHARE_STATE_TONES } from '../../engine/tones';
import { SHARE_STATE_WORDS } from '../../engine/words';
import { GestureButton, GestureRow } from '../kit/gestures';
import { Address } from './address';
import { usePortStop } from './port-gestures';

import styles from './ports.module.css';

export const LOCAL_ELSEWHERE =
    'Not opened: its address is local to the Mac it is shared on, and the engine gives none to open from this Mac';

function ShareAddress({ share, onThisMac }: { share: PortsSharedPort; onThisMac: boolean }) {
    return (
        <Address
            label={share.detail}
            opens={
                share.http
                    ? { held: LOCAL_ELSEWHERE, openable: Openable.ofShare(share, onThisMac) }
                    : undefined
            }
            qr={share.urls.public !== ''}
            url={shareAddressOf(share)}
        />
    );
}

type SharedRowProps = {
    share: PortsSharedPort;
    target: Target | undefined;
    onThisMac: boolean;
    compact?: boolean;
};

export function SharedRow({ share, target, onThisMac, compact = false }: SharedRowProps) {
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
        lead: <Value look="raised">{`:${share.guest_port}`}</Value>,
        leadSize: 'value',
        state: {
            calm: 'unsaid',
            tone: SHARE_STATE_TONES[share.state],
            word: SHARE_STATE_WORDS[share.state],
        },
        title: label === '' ? share.process : label,
    } as const;
    if (compact) {
        const { state, ...rest } = row;
        return (
            <section aria-label={`Port ${share.guest_port}`} className={styles.shared}>
                <GestureRow
                    {...rest}
                    {...(isCalm(state.tone) ? {} : { state })}
                    {...(to === '' ? {} : { tag: shareToWord(to), tagLook: 'hint' as const })}
                    sentence={<Value>{shareAddressOf(share)}</Value>}
                    trail={stopButton}
                />
            </section>
        );
    }
    return (
        <section aria-label={`Port ${share.guest_port}`} className={styles.shared}>
            <GestureRow
                {...row}
                {...(label === '' ? {} : { tag: share.process, tagLook: 'hint' as const })}
                sentence={<Value>{shareAddressOf(share)}</Value>}
                trail={
                    <span className={styles.trail}>
                        {to === '' ? null : <TypeTag>{shareToWord(to)}</TypeTag>}
                        {stopButton}
                    </span>
                }
                trailSize="wide"
            />
            <ShareAddress onThisMac={onThisMac} share={share} />
        </section>
    );
}
