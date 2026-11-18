import type { CSSProperties } from 'react';

// invariant: the one inline style the window writes: a size computed from data, handed to the element as a CSS
// invariant: custom property its module's CSS draws. A colour, a spacing or any other property stays in the CSS.
export type Measured = CSSProperties & Readonly<Record<`--${string}`, number | string>>;
