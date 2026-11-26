import {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useId,
    useLayoutEffect,
    useRef,
    useState,
} from 'react';
import type { FocusEvent, KeyboardEvent, RefObject } from 'react';
import { createPortal } from 'react-dom';

import { holdFocus } from './focus';
import type { Measured } from './measure';
import { PopoverList } from './popover-list';
import type { PopoverFilter, PopoverSection } from './popover-list';
import { placed } from './popover-place';
import type { Align, Box, Placed } from './popover-place';

import styles from './popover.module.css';

export type { PopoverFilter, PopoverRow, PopoverSection } from './popover-list';

// value: where a popover's layer is drawn: the document's body, or a frame that holds it, as the gallery's do.
export const PopoverHost = createContext<HTMLElement | null>(null);

export type PopoverProps = {
    label: string;
    anchor: RefObject<HTMLElement | null>;
    sections: readonly PopoverSection[];
    empty: string;
    onChoose: (id: string) => void;
    onClose: () => void;
    filter?: PopoverFilter | undefined;
    align?: Align;
    columns?: boolean;
    id?: string | undefined;
};

function boxOf(element: Element): Box {
    const { left, top, width, height } = element.getBoundingClientRect();
    return { height, left, top, width };
}

function same(one: Placed | undefined, other: Placed): boolean {
    return (
        one !== undefined &&
        one.x === other.x &&
        one.y === other.y &&
        one.height === other.height &&
        one.side === other.side
    );
}

// invariant: a popover's natural height is its own less what its list shows plus all its list holds, so a list
// invariant: capped by a short window is placed for what it would be, and a longer window gives it its room back.
function naturalHeight(element: HTMLElement): number {
    const list = element.querySelector<HTMLElement>('[role="listbox"]');
    return list
        ? element.offsetHeight - list.clientHeight + list.scrollHeight
        : element.offsetHeight;
}

type Refs = {
    anchor: RefObject<HTMLElement | null>;
    layer: RefObject<HTMLDivElement | null>;
    pop: RefObject<HTMLDivElement | null>;
};

// invariant: placed again whenever the popover or its layer changes size and whenever the window resizes or
// invariant: anything scrolls, so it follows its opener; it stays unseen (no data-side) until first placed.
function usePlace({ anchor, layer, pop }: Refs, align: Align): Placed | undefined {
    const [place, setPlace] = useState<Placed>();
    useEffect(() => {
        const measure = () => {
            const opener = anchor.current;
            const element = pop.current;
            const ground = layer.current;
            if (opener && element && ground) {
                const size = { height: naturalHeight(element), width: element.offsetWidth };
                const next = placed(boxOf(opener), size, boxOf(ground), align);
                setPlace((was) => (same(was, next) ? was : next));
            }
        };
        const sized = new ResizeObserver(measure);
        for (const watched of [pop.current, layer.current]) {
            if (watched) {
                sized.observe(watched);
            }
        }
        globalThis.addEventListener('resize', measure);
        document.addEventListener('scroll', measure, true);
        return () => {
            sized.disconnect();
            globalThis.removeEventListener('resize', measure);
            document.removeEventListener('scroll', measure, true);
        };
    }, [anchor, layer, pop, align]);
    return place;
}

function useCloseOnce(onClose: () => void): () => void {
    const latest = useRef(onClose);
    const closed = useRef(false);
    useEffect(() => {
        latest.current = onClose;
    });
    return useCallback(() => {
        if (!closed.current) {
            closed.current = true;
            latest.current();
        }
    }, []);
}

function useOutsidePress({ anchor, pop }: Refs, close: () => void): void {
    useEffect(() => {
        const pressed = (event: PointerEvent) => {
            const { target } = event;
            const inside =
                target instanceof Node &&
                ((pop.current?.contains(target) ?? false) ||
                    (anchor.current?.contains(target) ?? false));
            if (!inside) {
                close();
            }
        };
        document.addEventListener('pointerdown', pressed, true);
        return () => {
            document.removeEventListener('pointerdown', pressed, true);
        };
    }, [anchor, pop, close]);
}

function firstOf(element: HTMLElement): HTMLElement {
    return element.querySelector<HTMLElement>('input, [role="listbox"]') ?? element;
}

function dismissalOf(anchor: RefObject<HTMLElement | null>, close: () => void) {
    return {
        onBlur: (event: FocusEvent) => {
            const next = event.relatedTarget;
            const kept =
                !(next instanceof Node) ||
                event.currentTarget.contains(next) ||
                (anchor.current?.contains(next) ?? false);
            if (!kept) {
                close();
            }
        },
        onKeyDown: (event: KeyboardEvent) => {
            if (event.key === 'Escape' || event.key === 'Tab') {
                event.preventDefault();
                event.stopPropagation();
                close();
            }
        },
    };
}

function placedStyle(place: Placed | undefined): Measured {
    if (place === undefined) {
        return {};
    }
    return {
        '--popover-height': `${String(place.height)}px`,
        '--popover-x': `${String(place.x)}px`,
        '--popover-y': `${String(place.y)}px`,
    };
}

// invariant: a popover is no modal: it does not count in aDialogIsOpen, so a window shortcut still acts while it
// invariant: is open, and it closes on Escape, on Tab, on a press outside it and on the focus leaving it.
export function Popover(props: PopoverProps) {
    const {
        label,
        anchor,
        sections,
        empty,
        onChoose,
        onClose,
        filter,
        align = 'start',
        columns = false,
        id,
    } = props;
    const host = useContext(PopoverHost);
    const layer = useRef<HTMLDivElement>(null);
    const pop = useRef<HTMLDivElement>(null);
    const refs = { anchor, layer, pop };
    const close = useCloseOnce(onClose);
    const place = usePlace(refs, align);
    const { onBlur, onKeyDown } = dismissalOf(anchor, close);
    useOutsidePress(refs, close);
    useLayoutEffect(() => holdFocus(pop.current, firstOf), []);
    return createPortal(
        <div className={styles.layer} ref={layer}>
            <div
                aria-label={label}
                className={`${styles.popover} ${columns ? styles.card : ''}`}
                data-side={place?.side}
                id={id}
                onBlur={onBlur}
                onKeyDown={onKeyDown}
                ref={pop}
                role="dialog"
                style={placedStyle(place)}
                tabIndex={-1}
            >
                <PopoverList
                    close={close}
                    columns={columns}
                    empty={empty}
                    filter={filter}
                    label={label}
                    onChoose={onChoose}
                    sections={sections}
                />
            </div>
        </div>,
        host ?? document.body,
    );
}

export type PopoverOpener = {
    ref: RefObject<HTMLButtonElement | null>;
    'aria-controls': string | undefined;
    'aria-expanded': boolean;
    'aria-haspopup': 'dialog';
    onClick: () => void;
    onKeyDown: (event: KeyboardEvent) => void;
};

// value: what opens a popover: a button spread with `opener`, opened by a press, Enter or Space as any button is,
// value: and by ArrowDown, as a menu button is; `anchor` and `id` go to the Popover, `close` is its onClose.
export function usePopover(opened = false) {
    const [open, setOpen] = useState(opened);
    const anchor = useRef<HTMLButtonElement>(null);
    const id = useId();
    const opener: PopoverOpener = {
        'aria-controls': open ? id : undefined,
        'aria-expanded': open,
        'aria-haspopup': 'dialog',
        onClick: () => {
            setOpen((was) => !was);
        },
        onKeyDown: (event) => {
            if (event.key === 'ArrowDown') {
                event.preventDefault();
                setOpen(true);
            }
        },
        ref: anchor,
    };
    return {
        anchor,
        close: () => {
            setOpen(false);
        },
        id,
        open,
        opener,
    };
}
