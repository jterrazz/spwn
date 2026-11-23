import type { MenuBarPort } from '../../../application/ports/menu-bar.port';
import type { MenuBarRead, MenuBarRefused } from '../../../domain/menu-bar';

export const MENU_BAR_SCRIPTS = ['approval', 'hidden', 'refusing', 'shown', 'waiting'] as const;

export type MenuBarScriptName = (typeof MENU_BAR_SCRIPTS)[number];

type Held = Exclude<MenuBarRead, { kind: 'unsupported' }>;

type MenuBarScript = { first: Held; shown: Held | MenuBarRefused };

const REFUSED: MenuBarRefused = {
    code: 'menubar.not-switched',
    kind: 'refused',
    message: 'SMAppServiceErrorDomain 1: Operation not permitted',
};

const SCRIPTS: Readonly<Record<MenuBarScriptName, MenuBarScript>> = {
    approval: { first: { kind: 'hidden' }, shown: { kind: 'approval' } },
    hidden: { first: { kind: 'hidden' }, shown: { kind: 'shown' } },
    refusing: { first: { kind: 'hidden' }, shown: REFUSED },
    shown: { first: { kind: 'shown' }, shown: { kind: 'shown' } },
    waiting: { first: { kind: 'approval' }, shown: { kind: 'approval' } },
};

function isMenuBarScript(name: string): name is MenuBarScriptName {
    return MENU_BAR_SCRIPTS.some((known) => known === name);
}

export function menuBarScriptOf(name: string): MenuBarScript | undefined {
    return isMenuBarScript(name) ? SCRIPTS[name] : undefined;
}

// value: the proof build's shell: the login item held in memory, from the script's first state; showing it answers
// value: the script's, hiding it always hides it.
export function scriptedMenuBar(script: MenuBarScript): MenuBarPort {
    let held: Held = script.first;
    return {
        read: async () => await Promise.resolve(held),
        show: async (shown) => {
            const answer: Held | MenuBarRefused = shown ? script.shown : { kind: 'hidden' };
            if (answer.kind !== 'refused') {
                held = answer;
            }
            return await Promise.resolve(answer);
        },
    };
}
