import { useRef, useState } from 'react';
import type { PointerEvent, ReactNode } from 'react';

import type { Identity } from './identity';
import type { MemoryHolder } from './memory-grid';

import styles from './memory-pick.module.css';

export type MemoryTake = {
    min: number;
    max: number;
    onTake: (squares: number) => void;
};

export type MemoryPickProps = {
    name: string;
    line: string;
    holders: readonly MemoryHolder[];
    chosen: number;
    identity?: Identity | undefined;
    free: number;
    take?: MemoryTake | undefined;
    children?: ReactNode;
};

type GridEvent = PointerEvent<HTMLDivElement>;

function countOf(squares: number): number {
    return Number.isFinite(squares) ? Math.max(0, Math.round(squares)) : 0;
}

function keysOf(prefix: string, squares: number): readonly string[] {
    return Array.from({ length: countOf(squares) }, (_, at) => `${prefix}-${String(at)}`);
}

function squareUnder(grid: HTMLElement, under: EventTarget | null): number | undefined {
    const square = under instanceof Element ? under.closest<HTMLElement>('[data-take]') : null;
    return square === null || !grid.contains(square) ? undefined : Number(square.dataset.take);
}

function pointedIn(event: GridEvent): number | undefined {
    const grid = event.currentTarget;
    return squareUnder(grid, grid.ownerDocument.elementFromPoint(event.clientX, event.clientY));
}

function useTaking(take: MemoryTake | undefined, room: number) {
    const held = useRef(false);
    const [dragging, setDragging] = useState(false);
    if (take === undefined) {
        return { dragging: false, handlers: {} };
    }
    const most = Math.min(take.max, room);
    const taken = (square: number | undefined): void => {
        if (square !== undefined && most >= take.min) {
            take.onTake(Math.min(most, Math.max(take.min, square)));
        }
    };
    const end = (): void => {
        held.current = false;
        setDragging(false);
    };
    const down = (event: GridEvent): void => {
        const square = squareUnder(event.currentTarget, event.target);
        if (square === undefined) {
            return;
        }
        event.preventDefault();
        event.currentTarget.setPointerCapture(event.pointerId);
        held.current = true;
        setDragging(true);
        taken(square);
    };
    const move = (event: GridEvent): void => {
        if (held.current) {
            taken(pointedIn(event));
        }
    };
    return {
        dragging,
        handlers: {
            onLostPointerCapture: end,
            onPointerCancel: end,
            onPointerDown: down,
            onPointerMove: move,
            onPointerUp: end,
        },
    };
}

function takeOf(take: MemoryTake | undefined, square: number): string | undefined {
    return take === undefined ? undefined : String(square);
}

export function MemoryPick({
    name,
    line,
    holders,
    chosen,
    identity,
    free,
    take,
    children,
}: MemoryPickProps) {
    const before = countOf(chosen);
    const taking = useTaking(take, before + countOf(free));
    return (
        <div className={styles.pick}>
            <div className={styles.head}>
                {chosen > 0 ? (
                    <b className={styles.size}>
                        {String(chosen)}
                        <small className={styles.unit}>GB</small>
                    </b>
                ) : null}
                {line === '' ? null : <span className={styles.line}>{line}</span>}
            </div>
            <div
                aria-label={`Memory of ${name}, one square per GB`}
                className={`${styles.grid} ${take === undefined ? '' : styles.take}`}
                data-dragging={taking.dragging ? 'true' : undefined}
                role="img"
                {...taking.handlers}
            >
                {holders.map((holder) =>
                    keysOf(holder.key, holder.squares).map((key) => (
                        <i
                            className={`${styles.cell} ${holder.identity === 'system' ? '' : styles.held}`}
                            data-identity={holder.identity}
                            key={key}
                        />
                    )),
                )}
                {keysOf('chosen', chosen).map((key, at) => (
                    <i
                        className={styles.cell}
                        data-identity={identity}
                        data-take={takeOf(take, at + 1)}
                        key={key}
                    />
                ))}
                {keysOf('free', free).map((key, at) => (
                    <i
                        className={`${styles.cell} ${styles.free}`}
                        data-take={takeOf(take, before + at + 1)}
                        key={key}
                    />
                ))}
            </div>
            {children}
        </div>
    );
}
