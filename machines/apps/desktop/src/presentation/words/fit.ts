import type { Unfit } from '../../domain/fit';
import type { Shown } from './shown';

// outside: the engine names only the revision its pages speak, so the verdict's words are the shared vocabulary of
// outside: docs/reference/api.md's table, held equal to it by `fit.test.ts`; they go when ADR-070's `opened.fit`
// outside: arrives worded.
export const FIT_WORDS: Readonly<Record<Unfit, string>> = {
    'app-older': 'this app older than the engine',
    'engine-older': 'engine older than this app',
};

export const FIT_REMEDY_WORDS: Readonly<Record<Unfit, string>> = {
    'app-older': 'Update Spwn',
    'engine-older': 'Update station here',
};

export const FIT_LEAD_WORDS: Readonly<Record<Unfit, string>> = {
    'app-older':
        'The engine on this Mac runs, and speaks a newer revision of its pages than this window draws. Update the window to see your machines.',
    'engine-older':
        'The engine on this Mac runs, and speaks an older revision of its pages than this window draws. Update station to see your machines.',
};

export const FIT_MACHINES_WORDS: Readonly<Record<Unfit, string>> = {
    'app-older': 'Update Spwn to see them',
    'engine-older': 'Update station to see them',
};

// value: the sidebar foot's line, the verdict itself opening a line.
export function fitFootWord(fit: Unfit): string {
    const words = FIT_WORDS[fit];
    return `${words.charAt(0).toUpperCase()}${words.slice(1)}`;
}

export const FIT_UPDATE_TO = 'the latest release';

export const FIT_PANEL_WORD = 'The engine on this Mac';

export const FIT_APP_UPDATE_WORD =
    'This window updates itself from a release build on a Mac; elsewhere, install the latest Spwn_darwin_arm64.zip from its release.';

// invariant: an incompatible engine runs, so its verdict is drawn as attention, never as stopped.
export function fitState(fit: Unfit): Shown {
    return { tone: 'warn', word: FIT_WORDS[fit] };
}
