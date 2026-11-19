export interface PresencePort {
    visible: () => boolean;
    // value: `back` when the window is shown again or regains a focus it lost, `hidden` when it is hidden.
    subscribe: (listener: (seen: 'back' | 'hidden') => void) => () => void;
}
