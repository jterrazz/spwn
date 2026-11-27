export const KINDS = ['css', 'fonts', 'js'] as const;

export const MEASURES = ['brotli', 'files', 'raw'] as const;

export type Kind = (typeof KINDS)[number];

export type Measure = (typeof MEASURES)[number];

// value: the most the production build may weigh, in bytes (and font files), the next 500 B above what it weighs: js
// value: 654,507 raw and 173,122 brotli, css 92,586 raw and 17,675 brotli (Geist Mono's symbols face inlined in it),
// value: fonts 62,784 raw and 62,798 brotli in 4 files (variable Geist and Geist Mono 400, each Latin and
// value: Latin-extended).
// value: A change that grows one past its line raises the line here, in its own diff, where a reviewer reads it.
export const BUDGET: Record<Kind, Partial<Record<Measure, number>>> = {
    css: { brotli: 18_000, raw: 93_000 },
    fonts: { brotli: 63_000, files: 4, raw: 63_000 },
    js: { brotli: 173_500, raw: 655_000 },
};
