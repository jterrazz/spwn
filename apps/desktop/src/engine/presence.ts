export type Presence = {
    visible: () => boolean;
    // value: `back` when the window is shown again or regains a focus it lost, `hidden` when it is hidden.
    subscribe: (listener: (seen: 'back' | 'hidden') => void) => () => void;
};

const ALWAYS: Presence = {
    subscribe: () => () => undefined,
    visible: () => true,
};

export function documentPresence(): Presence {
    if (typeof document === 'undefined') {
        return ALWAYS;
    }
    return {
        subscribe: (listener) => {
            let away = false;
            const back = () => {
                if (away) {
                    away = false;
                    listener('back');
                }
            };
            const changed = () => {
                if (document.visibilityState === 'hidden') {
                    away = true;
                    listener('hidden');
                } else {
                    back();
                }
            };
            const blurred = () => {
                away = true;
            };
            document.addEventListener('visibilitychange', changed);
            globalThis.addEventListener('blur', blurred);
            globalThis.addEventListener('focus', back);
            return () => {
                document.removeEventListener('visibilitychange', changed);
                globalThis.removeEventListener('blur', blurred);
                globalThis.removeEventListener('focus', back);
            };
        },
        visible: () => document.visibilityState !== 'hidden',
    };
}
