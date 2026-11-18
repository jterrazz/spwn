import {
    siAdguard,
    siAndroidstudio,
    siAnsible,
    siBitwarden,
    siBrave,
    siBun,
    siCocoapods,
    siCursor,
    siDiscord,
    siExpo,
    siGhostty,
    siGit,
    siGo,
    siGooglegemini,
    siHelm,
    siHomebrew,
    siLens,
    siLinear,
    siMessenger,
    siNotion,
    siObsidian,
    siOllama,
    siOpencode,
    siOpenjdk,
    siPnpm,
    siPulumi,
    siPython,
    siRust,
    siShadcnui,
    siSpeedtest,
    siStarship,
    siTerraform,
    siTmux,
    siUv,
    siWhatsapp,
    siXcode,
    siZedindustries,
} from 'simple-icons';
import type { SimpleIcon } from 'simple-icons';

import { MENUBAR_MARKS } from './menubar-marks';

// value: a name the engine sends (`App.Name`, `Step.Item`, `provider`) to the menu bar mark it names otherwise.
export const ALIASES: Readonly<Record<string, string>> = {
    'claude-desktop': 'claude',
    gh: 'github',
    'github-release': 'github',
    kubectl: 'kubernetes',
};

// value: a name the menu bar app has no mark for, to the Simple Icons logo that identifies it.
export const LOGOS: Readonly<Record<string, SimpleIcon>> = {
    adguard: siAdguard,
    'android-studio': siAndroidstudio,
    ansible: siAnsible,
    bitwarden: siBitwarden,
    brave: siBrave,
    bun: siBun,
    cocoapods: siCocoapods,
    cursor: siCursor,
    discord: siDiscord,
    expo: siExpo,
    gemini: siGooglegemini,
    ghostty: siGhostty,
    git: siGit,
    go: siGo,
    helm: siHelm,
    homebrew: siHomebrew,
    java: siOpenjdk,
    lens: siLens,
    linear: siLinear,
    messenger: siMessenger,
    notion: siNotion,
    obsidian: siObsidian,
    ollama: siOllama,
    opencode: siOpencode,
    pnpm: siPnpm,
    pulumi: siPulumi,
    python: siPython,
    rust: siRust,
    shadcn: siShadcnui,
    speedtest: siSpeedtest,
    starship: siStarship,
    terraform: siTerraform,
    tmux: siTmux,
    uv: siUv,
    whatsapp: siWhatsapp,
    xcode: siXcode,
    zed: siZedindustries,
};

export const LETTERED_KINDS: ReadonlySet<string> = new Set(['setting']);

export type Shape = {
    attributes: Readonly<Record<string, string>>;
    children: readonly Shape[];
    key: string;
    tag: string;
};

export type Mark =
    | { kind: 'drawn'; shapes: readonly Shape[] }
    | { kind: 'monogram'; letters: string };

const LOGO_IN_THE_BOX = 'translate(2 2) scale(0.6667)';

function attributesOf(text: string): Record<string, string> {
    return Object.fromEntries(
        [...text.matchAll(/(?<name>[a-z][a-z0-9-]*)="(?<value>[^"]*)"/gu)].map((match) => [
            (match.groups?.name ?? '').replaceAll(/-(?<letter>[a-z])/gu, (_, letter: string) =>
                letter.toUpperCase(),
            ),
            match.groups?.value === '#000' ? 'currentColor' : (match.groups?.value ?? ''),
        ]),
    );
}

export function shapesOf(markup: string): Shape[] {
    const root: Shape[] = [];
    const open: Shape[][] = [root];
    for (const match of markup.matchAll(
        /<(?<close>\/?)(?<tag>[a-z]+)(?<attributes>[^>]*?)(?<empty>\/?)>/gu,
    )) {
        if (match.groups?.close === '/') {
            open.pop();
            continue;
        }
        const children: Shape[] = [];
        const tag = match.groups?.tag ?? '';
        const text = match.groups?.attributes ?? '';
        open.at(-1)?.push({ attributes: attributesOf(text), children, key: `${tag}${text}`, tag });
        if (match.groups?.empty !== '/') {
            open.push(children);
        }
    }
    return root;
}

const DRAWN: Readonly<Record<string, readonly Shape[]>> = Object.fromEntries(
    Object.entries(MENUBAR_MARKS).map(([name, markup]) => [name, shapesOf(markup)]),
);

function monogramOf(name: string): Mark {
    return { kind: 'monogram', letters: name.replaceAll(/[^\p{L}\p{N}]/gu, '').slice(0, 2) };
}

export function markOf(name: string, kind = ''): Mark {
    if (LETTERED_KINDS.has(kind)) {
        return monogramOf(name);
    }
    const lower = name.toLowerCase();
    const drawn: readonly Shape[] | undefined = DRAWN[ALIASES[lower] ?? lower];
    if (drawn !== undefined) {
        return { kind: 'drawn', shapes: drawn };
    }
    const logo: SimpleIcon | undefined = LOGOS[lower];
    if (logo !== undefined) {
        return {
            kind: 'drawn',
            shapes: [
                {
                    attributes: { d: logo.path, transform: LOGO_IN_THE_BOX },
                    children: [],
                    key: logo.slug,
                    tag: 'path',
                },
            ],
        };
    }
    return monogramOf(name);
}
