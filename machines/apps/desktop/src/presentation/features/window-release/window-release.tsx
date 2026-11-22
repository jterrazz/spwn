import type {
    WindowUpdateFlow,
    WindowUpdateState,
} from '../../../application/use-cases/window-update-flow';
import { useWindowUpdate } from '../../dependencies/dependencies';
import { Button, Panel, ProgressLine, Row } from '../../ui/index';
import {
    WINDOW_REFUSAL_WORDS,
    WINDOW_RELEASE_WORDS,
    windowCurrentWord,
    windowDownloadedWord,
    windowDownloadingWord,
    windowOutWord,
    windowRefusalHint,
} from '../../words/words';

import styles from './window-release.module.css';

type Drawn = Exclude<WindowUpdateState, { kind: 'unknown' } | { kind: 'unsupported' }>;

function sentenceOf(state: Drawn): string | undefined {
    switch (state.kind) {
        case 'checking':
        case 'refused': {
            return undefined;
        }
        case 'current': {
            return windowCurrentWord(state.current);
        }
        default: {
            return windowOutWord(state);
        }
    }
}

type FailedState = Extract<Drawn, { kind: 'failed' }>;

function Failed({ flow, state }: { flow: WindowUpdateFlow; state: FailedState }) {
    const hint = windowRefusalHint(state.code);
    return (
        <span className={styles.below}>
            <ProgressLine detail={state.message} failed task={WINDOW_REFUSAL_WORDS[state.code]} />
            {hint === undefined ? null : <span className={styles.fine}>{hint}</span>}
            <span className={styles.acts}>
                <Button
                    onClick={() => {
                        flow.retry();
                    }}
                    variant="primary"
                    yields
                >
                    {WINDOW_RELEASE_WORDS.tryAgain}
                </Button>
            </span>
        </span>
    );
}

function Below({ flow, state }: { flow: WindowUpdateFlow; state: Drawn }) {
    switch (state.kind) {
        case 'checking': {
            return <ProgressLine task={WINDOW_RELEASE_WORDS.checking} />;
        }
        case 'refused': {
            return (
                <ProgressLine
                    detail={state.message}
                    failed
                    task={WINDOW_REFUSAL_WORDS[state.code]}
                />
            );
        }
        case 'out': {
            return (
                <span className={styles.below}>
                    <Button
                        onClick={() => {
                            flow.install();
                        }}
                        variant="primary"
                        yields
                    >
                        {WINDOW_RELEASE_WORDS.install}
                    </Button>
                </span>
            );
        }
        case 'installing': {
            return (
                <ProgressLine
                    detail={windowDownloadedWord(state.downloaded, state.total)}
                    task={windowDownloadingWord(state.version)}
                />
            );
        }
        case 'relaunching': {
            return <ProgressLine task={WINDOW_RELEASE_WORDS.relaunching} />;
        }
        case 'failed': {
            return <Failed flow={flow} state={state} />;
        }
        case 'current': {
            return null;
        }
    }
}

// invariant: drawn whatever the engine answers, since the window's release is its own; nothing before the first answer
// invariant: and nothing where the shell cannot update the window (a debug build, Linux, a page outside Tauri).
export function WindowRelease() {
    const { flow, state } = useWindowUpdate();
    if (
        state.kind === 'unknown' ||
        state.kind === 'unsupported' ||
        (state.kind === 'checking' && state.first)
    ) {
        return null;
    }
    const held = state.kind === 'installing' || state.kind === 'relaunching';
    return (
        <Panel title={WINDOW_RELEASE_WORDS.title}>
            <Row
                below={<Below flow={flow} state={state} />}
                sentence={sentenceOf(state)}
                title={WINDOW_RELEASE_WORDS.latest}
                trail={
                    <Button
                        busy={state.kind === 'checking'}
                        disabled={held}
                        onClick={() => {
                            flow.checkNow();
                        }}
                        title={WINDOW_RELEASE_WORDS.checkTitle}
                    >
                        {WINDOW_RELEASE_WORDS.checkNow}
                    </Button>
                }
            />
        </Panel>
    );
}
