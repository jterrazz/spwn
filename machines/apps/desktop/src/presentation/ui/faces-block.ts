// value: the PostCSS pass `vite.config.ts` runs over every stylesheet: each face is drawn `block`, never `swap`, so its
// value: text waits for its own face, which the launch loads before the window shows; no fallback frame is drawn.
export const FACES_BLOCK = {
    Declaration: {
        'font-display': (declaration: { value: string }) => {
            declaration.value = 'block';
        },
    },
    postcssPlugin: 'faces-block',
};
