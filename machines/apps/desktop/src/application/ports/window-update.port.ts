import type {
    WindowChecked,
    WindowInstalled,
    WindowProgress,
    WindowRefused,
} from '../../domain/window-update';

export type HeardProgress = (progress: WindowProgress) => void;

export interface WindowUpdatePort {
    check: () => Promise<WindowChecked | WindowRefused>;
    install: (heard: HeardProgress) => Promise<WindowInstalled | WindowRefused>;
}
