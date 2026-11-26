const SHIPPED_FILE =
    /url\(\s*(?<quote>["']?)(?<path>[^)"']*-(?:latin|latin-ext|symbols2)-\d+-normal\.woff2)\k<quote>\s*\)\s*format\(\s*["']woff2["']\s*\)/u;

interface Declaration {
    value: string;
}

interface FontFace {
    remove: () => void;
    walkDecls: (property: string, visit: (declaration: Declaration) => void) => void;
}

// value: the PostCSS pass `vite.config.ts` runs over every stylesheet: of each `@fontsource` face the build keeps the
// value: Latin subsets (`latin`, `latin-ext`) and Geist Mono's `symbols2` (box and block drawing), in `woff2` alone,
// value: which every macOS the window runs on reads; the Cyrillic and Vietnamese faces and every `woff` are never shipped.
export const FACES_SHIPPED = {
    AtRule: {
        'font-face': (face: FontFace) => {
            let kept = false;
            face.walkDecls('src', (declaration) => {
                const file = SHIPPED_FILE.exec(declaration.value);
                if (file?.groups?.path !== undefined) {
                    declaration.value = `url(${file.groups.path}) format('woff2')`;
                    kept = true;
                }
            });
            if (!kept) {
                face.remove();
            }
        },
    },
    postcssPlugin: 'faces-shipped',
};
