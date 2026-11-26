export const KINDS = ['css', 'fonts', 'js'] as const;

export const MEASURES = ['brotli', 'files', 'raw'] as const;

export type Kind = (typeof KINDS)[number];

export type Measure = (typeof MEASURES)[number];

// value: the most the production build may weigh, in bytes (and font files), a little above what it weighed when the
// value: Latin faces alone shipped (js 607,430 raw and 161,224 brotli, css 86,015 and 19,281, fonts 97,532 in 10
// value: files), then above it once the window drew the Guide sheet (js 641,855 raw and 170,471 brotli, css 96,494 raw).
// value: A change that grows one past its line raises the line here, in its own diff, where a reviewer reads it.
export const BUDGET: Record<Kind, Partial<Record<Measure, number>>> = {
    css: { brotli: 21_000, raw: 97_000 },
    fonts: { brotli: 105_000, files: 12, raw: 105_000 },
    js: { brotli: 172_000, raw: 645_000 },
};
