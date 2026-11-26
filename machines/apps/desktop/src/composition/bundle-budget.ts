export const KINDS = ['css', 'fonts', 'js'] as const;

export const MEASURES = ['brotli', 'files', 'raw'] as const;

export type Kind = (typeof KINDS)[number];

export type Measure = (typeof MEASURES)[number];

// value: the most the production build may weigh, in bytes (and font files), a little above what it weighs: js 648,496
// value: raw and 172,030 brotli, css 93,305 raw and 17,865 brotli (Geist Mono's symbols face inlined in it), fonts 62,784
// value: raw in 4 files (variable Geist and Geist Mono 400, each Latin and Latin-extended).
// value: A change that grows one past its line raises the line here, in its own diff, where a reviewer reads it.
export const BUDGET: Record<Kind, Partial<Record<Measure, number>>> = {
    css: { brotli: 18_400, raw: 96_200 },
    fonts: { brotli: 64_700, files: 4, raw: 64_700 },
    js: { brotli: 172_500, raw: 649_000 },
};
