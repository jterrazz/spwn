import type { PresencePort } from '../../../application/ports/presence.port';

const ALWAYS: PresencePort = {
    subscribe: () => () => undefined,
    visible: () => true,
};

export function documentPresence(): PresencePort {
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
