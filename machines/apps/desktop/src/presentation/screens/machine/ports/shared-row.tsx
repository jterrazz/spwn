import type { CatalogAudience, PortsSharedPort, Target } from '@jterrazz/station-api';

import type { Gesture } from '../../../dependencies/use-gesture';
import { GestureButton, GestureRow } from '../../../kit/gestures';
import { Openable, shareAddressOf } from '../../../kit/openable';
import { Switch, Value } from '../../../ui/index';
import { isCalm } from '../../../ui/state-mark';
import { shareToWord } from '../../../words/shown';
import { SHARE_STATE_TONES } from '../../../words/tones';
import { SHARE_REACH_GESTURE_WORDS, SHARE_STATE_WORDS } from '../../../words/words';
import { Address } from './address';
import { usePortReach, usePortStop } from './port-gestures';

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

const REACHES = ['local', 'private', 'public'] as const satisfies readonly CatalogAudience[];

const REACH_OPTIONS = REACHES.map((id) => ({ id, label: shareToWord(id) }));

function isReach(to: CatalogAudience): to is (typeof REACHES)[number] {
    return REACHES.some((reach) => reach === to);
}

function Reach({
    share,
    disabled,
    onPick,
}: {
    share: PortsSharedPort;
    disabled: boolean;
    onPick: (to: (typeof REACHES)[number]) => void;
}) {
    const { to } = share;
    if (to === undefined || !isReach(to)) {
        return null;
    }
    return (
        <Switch
            disabled={disabled}
            label={`${SHARE_REACH_GESTURE_WORDS.who} ${share.vm}:${share.guest_port}`}
            onChange={(next) => {
                if (next !== to) {
                    onPick(next);
                }
            }}
            options={REACH_OPTIONS}
            surface="bare"
            value={to}
        />
    );
}

function StopButton({ gesture, share }: { gesture: Gesture<unknown>; share: PortsSharedPort }) {
    return (
        <GestureButton
            gesture={gesture}
            label="Stop sharing"
            title={`Stop sharing ${share.vm}:${share.guest_port}?`}
            variant="link"
        />
    );
}

function useReaches(at: { vm: string; port: number; target: Target | undefined }) {
    const reaches = {
        local: usePortReach({ ...at, to: 'local' }),
        private: usePortReach({ ...at, to: 'private' }),
        public: usePortReach({ ...at, to: 'public' }),
    };
    const moving = [reaches.local, reaches.private, reaches.public].find(
        ({ state }) => state.kind !== 'idle' && state.kind !== 'done',
    );
    return { moving, reaches };
}

export function SharedRow({ share, target, onThisMac, compact = false }: SharedRowProps) {
    const { to } = share;
    const label = share.label ?? '';
    const stop = usePortStop({ port: share.guest_port, target, vm: share.vm });
    const { reaches, moving } = useReaches({ port: share.guest_port, target, vm: share.vm });
    const stopButton = <StopButton gesture={stop} share={share} />;
    const row = {
        gesture: moving ?? stop,
        lead: <Value look="raised">{`:${share.guest_port}`}</Value>,
        leadSize: 'value',
        state: {
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
                    {...(to === undefined
                        ? {}
                        : { tag: shareToWord(to), tagLook: 'hint' as const })}
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
                stateAt="start"
                trail={
                    <span className={styles.trail}>
                        <Reach
                            disabled={moving?.state.kind === 'running'}
                            onPick={(next) => {
                                reaches[next].start();
                            }}
                            share={share}
                        />
                        {stopButton}
                    </span>
                }
                trailSize="wide"
            />
            <ShareAddress onThisMac={onThisMac} share={share} />
        </section>
    );
}
