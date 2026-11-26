import { memo, useId, useLayoutEffect, useRef, useSyncExternalStore } from 'react';
import type { KeyboardEvent, RefObject } from 'react';

import { focusablesIn, moveFocus, moveOfKey } from '../ui/focus';
import type { Shown } from '../words/shown';
import type { Route } from './route';
import type { Soon } from './soon';
import { SpacePopover } from './space-popover';
import { opensNow, SOON_WORD, SPACE_WORDS, spaceOfRoute, SPACES } from './spaces';
import type { Space } from './spaces';

import styles from './space-switch.module.css';

export type SpaceSwitchProps = {
    route: Route;
    quiet: boolean;
    soon: Soon;
    standing: Shown | undefined;
    onSpace: (space: Space) => void;
};

type WordProps = Omit<SpaceSwitchProps, 'route' | 'soon'> & {
    space: Space;
    pressed: boolean;
    stop: boolean;
    listed: boolean;
    onClose: () => void;
};

// invariant: the arrows move the focus between the words and never press one, unlike a page's tabs.
function onWordKey(event: KeyboardEvent<HTMLButtonElement>): void {
    const move = moveOfKey(event.key, 'horizontal');
    const words = event.currentTarget.parentElement;
    if (move === undefined || words === null) {
        return;
    }
    event.preventDefault();
    moveFocus(focusablesIn(words, 'button:not(:disabled)'), move, false);
}

// invariant: a Soon list opens from its word whatever pressed it, ⌘2 and ⌘3 included: the word holds the focus before
// invariant: the list takes it, so Escape returns it there.
function FromWord({ anchor }: { anchor: RefObject<HTMLButtonElement | null> }) {
    useLayoutEffect(() => {
        anchor.current?.focus({ preventScroll: true });
    }, [anchor]);
    return null;
}

function WordOf({ space, pressed, stop, listed, quiet, standing, onSpace, onClose }: WordProps) {
    const anchor = useRef<HTMLButtonElement>(null);
    const id = useId();
    const { name } = SPACE_WORDS[space];
    const digit = String(SPACES.indexOf(space) + 1);
    const built = opensNow(space);
    return (
        <>
            <button
                aria-controls={listed ? id : undefined}
                aria-expanded={built ? undefined : listed}
                aria-haspopup={built ? undefined : 'dialog'}
                aria-keyshortcuts={`Meta+${digit}`}
                aria-label={built ? undefined : `${name} ${SOON_WORD.toLowerCase()}`}
                aria-pressed={pressed}
                className={styles.word}
                disabled={quiet}
                onClick={() => {
                    onSpace(space);
                }}
                onKeyDown={onWordKey}
                ref={anchor}
                tabIndex={stop ? 0 : -1}
                title={`${name} (⌘${digit})`}
                type="button"
            >
                <span className={styles.name}>{name}</span>
                <span aria-hidden="true" className={styles.held}>
                    {name}
                </span>
            </button>
            {listed ? <FromWord anchor={anchor} /> : null}
            {listed ? (
                <SpacePopover
                    anchor={anchor}
                    id={id}
                    onClose={onClose}
                    onSpace={onSpace}
                    standing={standing}
                />
            ) : null}
        </>
    );
}

// invariant: a word is drawn again only when its own state changes, so the bar's every render draws the switch alone;
// invariant: the fleet's standing reaches only the word whose list is open.
const Word = memo(WordOf);

// value: the three spaces at the bar's left, after the traffic lights: bare words, the route's own underlined at the
// value: bar's foot, closed by a thin rule; one Tab stop, on the underlined word or else the first.
export function SpaceSwitch({ route, quiet, soon, standing, onSpace }: SpaceSwitchProps) {
    const listed = useSyncExternalStore(soon.subscribe, soon.shown);
    const pressed = quiet ? 'machine' : spaceOfRoute(route);
    const stop = pressed ?? SPACES[0];
    return (
        <>
            <nav aria-label="Spaces" className={styles.spaces}>
                {SPACES.map((space) => (
                    <Word
                        key={space}
                        listed={!quiet && listed === space}
                        onClose={soon.close}
                        onSpace={onSpace}
                        pressed={space === pressed}
                        quiet={quiet}
                        space={space}
                        standing={listed === space ? standing : undefined}
                        stop={space === stop}
                    />
                ))}
            </nav>
            <span aria-hidden="true" className={styles.rule} />
        </>
    );
}
