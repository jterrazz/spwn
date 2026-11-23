import type { PresencePort } from '../../../application/ports/presence.port';

const ALWAYS: PresencePort = {
    visible: () => true,
};

export function documentPresence(): PresencePort {
    if (typeof document === 'undefined') {
        return ALWAYS;
    }
    return {
        visible: () => document.visibilityState !== 'hidden',
    };
}
