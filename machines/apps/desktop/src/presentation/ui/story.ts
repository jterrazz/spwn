// value: one drawn state of a component, as the gallery renders it: its name and the props that draw it. A
// value: component's stories sit beside it, in `<component>.stories.ts(x)`, a plain array the gallery maps.
export type Story<P> = { name: string; props: P };

export const nothing = (): void => undefined;

export const SECRET_SAMPLE = `x9${'k7'.repeat(20)}`;

export const LONG_NAME = 'a-machine-name-long-enough-to-be-cut-before-it-ends-in-any-column';
