import type { DiskReading, Target } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { bytes } from '../../../domain/format';
import { SEPARATOR } from '../../../domain/separator';
import { useReread } from '../../dependencies/dependencies';
import { HeldOutcomes } from '../../kit/gestures';
import type { ButtonVariant } from '../../ui/button';
import { Button, Panel, ShareBar } from '../../ui/index';
import type { SharePart } from '../../ui/index';
import { DISK_PENDING_WORD, DISK_TITLE } from '../../words/words';
import { isLow, kindPartsOf, listedOf } from './disk';
import type { Part } from './disk';
import { FreeAllButton, Freed, useFreeAll, useFreeImageCache } from './disk-gestures';
import type { DiskGesture } from './disk-gestures';
import { KeptPanel, UnusedFoot, UnusedPanel, useFreeUnused } from './unused';

import styles from './disk.module.css';

function Used({ reading }: { reading: DiskReading }) {
    const { Total, Used: used } = reading;
    if (Total <= 0) {
        return <span />;
    }
    return (
        <p className={styles.used}>
            <span className={styles.figure}>{bytes(used)}</span>
            {`of ${bytes(Total)} used`}
        </p>
    );
}

function DiskBar({
    parts,
    total,
    label,
    low,
}: {
    parts: readonly Part[];
    total: number;
    label: string;
    low: boolean;
}) {
    const drawn = parts.filter((part) => part.bytes > 0);
    if (total <= 0 || drawn.length === 0) {
        return null;
    }
    const said = parts.map((part) => `${part.label} ${bytes(part.bytes)}`).join(SEPARATOR);
    const held: SharePart[] = drawn.map((part) => ({
        key: part.key,
        label: `${part.label}${SEPARATOR}${bytes(part.bytes)}`,
        rest: part.rest,
        size: part.bytes,
        step: part.step,
    }));
    return <ShareBar label={`${label}: ${said}`} low={low} of={total} parts={held} />;
}

function Legend({ parts, low }: { parts: readonly Part[]; low: boolean }) {
    if (parts.length === 0) {
        return null;
    }
    return (
        <ul aria-label="The parts" className={styles.legend}>
            {parts.map((part) => (
                <li
                    className={part.rest ? `${styles.free} ${low ? styles.low : ''}` : undefined}
                    data-step={part.step}
                    key={part.key}
                >
                    <i className={styles.swatch} />
                    <span className={styles.word}>{part.label}</span>
                    <b className={styles.size}>{bytes(part.bytes)}</b>
                </li>
            ))}
        </ul>
    );
}

type FootProps = {
    reading: DiskReading;
    freeAll: DiskGesture;
    name: string;
    variant: ButtonVariant;
};

function Foot({ reading, freeAll, name, variant }: FootProps) {
    const { CanFree } = reading;
    if (CanFree === undefined) {
        return null;
    }
    return (
        <div className={styles.foot}>
            <p>{CanFree.Sentence}</p>
            {CanFree.Call === undefined ? null : (
                <FreeAllButton
                    gesture={freeAll}
                    label={`Free ${bytes(CanFree.Bytes)}`}
                    name={name}
                    variant={variant}
                />
            )}
        </div>
    );
}

type ImageCacheFootProps = { reading: DiskReading; freeImageCache: DiskGesture; name: string };

function ImageCacheFoot({ reading, freeImageCache, name }: ImageCacheFootProps) {
    const { ImageCache } = reading;
    if (ImageCache?.Call === undefined) {
        return null;
    }
    return (
        <div className={styles.foot}>
            <p>{ImageCache.Sentence}</p>
            <FreeAllButton
                gesture={freeImageCache}
                label={`Free the image cache, ${bytes(ImageCache.Bytes)}`}
                name={name}
            />
        </div>
    );
}

type DiskPanelProps = {
    reading: DiskReading;
    name: string;
    sub: ReactNode;
    gestures: DiskGestures;
    leads: boolean;
};

function DiskPanel({ reading, name, sub, gestures, leads }: DiskPanelProps) {
    const parts = listedOf(kindPartsOf(reading.Kinds));
    const unused = reading.Station?.CanFree;
    const reread = useReread();
    return (
        <Panel note={reading.Pending ? DISK_PENDING_WORD : ''} padded title={DISK_TITLE}>
            <div className={styles.hero}>
                <Used reading={reading} />
                <Button
                    onClick={() => {
                        reread(['disk.read']);
                    }}
                >
                    Rescan
                </Button>
            </div>
            {sub === undefined ? null : <p className={styles.sub}>{sub}</p>}
            <DiskBar
                label={`What holds the disk of ${name}`}
                low={isLow(reading)}
                parts={parts}
                total={reading.Total}
            />
            <Legend low={isLow(reading)} parts={parts} />
            {unused === undefined ? null : (
                <UnusedFoot
                    canFree={unused}
                    gesture={gestures.freeUnused}
                    name={name}
                    variant={leads ? 'primary' : 'plain'}
                />
            )}
            <Foot
                freeAll={gestures.freeAll}
                name={name}
                reading={reading}
                variant={leads && unused?.Call === undefined ? 'primary' : 'plain'}
            />
            <ImageCacheFoot
                freeImageCache={gestures.freeImageCache}
                name={name}
                reading={reading}
            />
        </Panel>
    );
}

type DiskGestures = {
    freeAll: DiskGesture;
    freeImageCache: DiskGesture;
    freeUnused: DiskGesture;
};

type DiskCardProps = {
    reading: DiskReading;
    name: string;
    target: Target | undefined;
    sub?: ReactNode;
    children?: ReactNode;
    // value: whether this card's Free leads the screen: a page of every Mac leads with This Mac's alone, since a
    // value: screen has one primary button.
    leads: boolean;
};

// invariant: every page that shows a machine's disk shows this one card: the same reading, the same parts, the
// invariant: same gestures, so two pages never tell one disk two ways.
export function DiskCard({ reading, name, target, sub, children, leads }: DiskCardProps) {
    const gestures: DiskGestures = {
        freeAll: useFreeAll(target),
        freeImageCache: useFreeImageCache(reading, target),
        freeUnused: useFreeUnused(reading.Station?.CanFree, target),
    };
    const unused = reading.Station?.CanFree;
    return (
        <>
            <DiskPanel
                gestures={gestures}
                leads={leads}
                name={name}
                reading={reading}
                sub={sub ?? reading.Station?.Sentence}
            />
            <HeldOutcomes>
                <Freed gesture={gestures.freeAll} />
                <Freed gesture={gestures.freeImageCache} />
                <Freed gesture={gestures.freeUnused} />
                {unused === undefined ? null : <UnusedPanel canFree={unused} />}
                <KeptPanel kept={reading.Station?.Kept} />
                {children}
            </HeldOutcomes>
        </>
    );
}
