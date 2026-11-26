// outside: a Linux VM only (headless Chromium); a Mac never opens a browser for this.
// value: usage: node scripts/canvas-dump.ts <spwn-wiki-proto-r1.html> <out-dir>: the prototype-*.json fixtures of
// value: src/presentation/ui/canvas/__tests__/, written by running the prototype's own functions. The prototype the owner
// value: approved lives outside this repository, at home/spwn/work/uncommitted/os-design/wiki-proto/ of the studio-os
// value: estate; its last line exports its state as window.__proto, and the dump injects the rest of what it reads
// value: before that line. The JSON is written minified: `make fix` formats it.
import { readFileSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { chromium } from 'playwright';

const [protoArgument, outArgument] = process.argv.slice(2);
if (protoArgument === undefined || outArgument === undefined) {
    throw new Error('usage: node scripts/canvas-dump.ts <spwn-wiki-proto-r1.html> <out-dir>');
}
const out = resolve(outArgument);
const MARK = 'window.__proto={S,H,';
const MORE = [
    'globalThis.dumpMore={M,labelOf,tw,PILLF,kidMarks,camTarget,camClamp,pagesOf,layout,stripH,ensureVisible,',
    'proto:()=>window.__proto,getSP:()=>SP,getDECO:()=>DECO,getBOX:()=>BOX,setCam0:(x,y)=>{cam0={x,y};},',
    'getCam:()=>cam,putCam:(x,y)=>{cam={x,y};}};',
].join('');
const html = readFileSync(protoArgument, 'utf8');
if (html.split(MARK).length !== 2) {
    throw new Error('the prototype no longer ends with the one window.__proto line');
}
const injected = join(out, 'proto-injected.html');
writeFileSync(injected, html.replace(MARK, MORE + MARK));

// value: the graph as the canvas reads it, each seat's band its pages' lowest floor, and each label's width in the
// value: pill's font, measured by the prototype's own canvas context.
const GRAPH = `(() => {
    const more = globalThis.dumpMore;
    const nodes = Object.values(more.proto().NODES).map((n) => ({
        band: n.kind === 'seat' ? Math.min(...more.pagesOf(n.seat).map((p) => p.floor)) : 0,
        depth: n.k, id: n.id, kids: n.kids, kind: n.kind, label: more.labelOf(n.id),
        marks: more.kidMarks(n.id), parent: n.par,
    }));
    const widths = Object.fromEntries(nodes.map((n) => [n.label, more.tw(n.label, more.PILLF)]));
    return { nodes, roots: more.M.seats.map((s) => 'S:' + s.id), widths };
})()`;

// value: four pulls per state, from two starts, for camClamp.
const PROBES = [
    [
        { x: 0, y: 0 },
        { x: -5000, y: -5000 },
    ],
    [
        { x: 0, y: 0 },
        { x: 5000, y: 5000 },
    ],
    [
        { x: -200, y: 100 },
        { x: -3000, y: 3000 },
    ],
    [
        { x: -200, y: 100 },
        { x: 40, y: -30 },
    ],
];

function stateOf(id: null | string): string {
    return `(() => {
    const more = globalThis.dumpMore;
    const proto = more.proto();
    proto.S.sel = [];
    proto.S.leaf = null;
    const id = ${JSON.stringify(id)};
    if (id !== null) proto.activate(id);
    more.layout();
    const space = more.getSP();
    const clamps = ${JSON.stringify(PROBES)}.map(([from, to]) => {
        more.setCam0(from.x, from.y);
        return { from, out: more.camClamp(to.x, to.y), to };
    });
    const rects = Object.fromEntries(Object.entries(proto.LAY).map(([key, r]) =>
        [key, { face: r.f, h: r.h, w: r.w, x: r.x, y: r.y }]));
    const guides = Object.entries(more.getDECO()).map(([key, d]) =>
        ({ band: Number(key.slice(2)), w: d.w, x: d.x, y: d.y }));
    return { id, box: more.getBOX(), clamps, guides, rects, sel: [...proto.S.sel], target: more.camTarget(),
        viewport: { height: space.h - more.stripH(), width: space.w } };
})()`;
}

// value: a walk opened node by node, then for each node drawn: where each arrow (left, right, up, down) moved the
// value: focus through the prototype's own key handler, and where ensureVisible took the camera from three starts.
function walkOf(path: readonly string[]): string {
    return `(() => {
    const more = globalThis.dumpMore;
    const proto = more.proto();
    proto.S.sel = [];
    proto.S.leaf = null;
    for (const id of ${JSON.stringify(path)}) proto.activate(id);
    proto.renderCanvas(false);
    const target = more.camTarget();
    const ids = Object.keys(proto.LAY);
    const moves = Object.fromEntries(ids.map((id) => [id, ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].map((key) => {
        const button = document.querySelector('#world .hitb[data-n="' + CSS.escape(id) + '"]');
        more.putCam(target.x, target.y);
        button.focus();
        button.dispatchEvent(new KeyboardEvent('keydown', { bubbles: true, key }));
        const to = document.activeElement.dataset.n;
        return to === id ? null : to;
    })]));
    const shifts = [{ x: 0, y: 0 }, { x: -900, y: -600 }, { x: 900, y: 600 }];
    const visible = ids.flatMap((id) => shifts.map((shift) => {
        const from = { x: target.x + shift.x, y: target.y + shift.y };
        more.putCam(from.x, from.y);
        more.ensureVisible(id);
        const at = more.getCam();
        return { from, id, out: { x: at.x, y: at.y } };
    }));
    const space = more.getSP();
    return { leaf: proto.S.leaf, moves, sel: [...proto.S.sel],
        viewport: { height: space.h - more.stripH(), width: space.w }, visible };
})()`;
}

const WALKS = [
    [],
    ['S:product'],
    ['S:product', 'P:product/boundaries'],
    ['S:product', 'P:product/boundaries', 'C:product/boundaries:1'],
    [
        'S:product',
        'P:product/boundaries',
        'C:product/boundaries:1',
        'B:product/boundaries:not-a-method',
    ],
];

type Dumped = { nodes: { id: string; depth: number }[] };

function isDumped(value: unknown): value is Dumped {
    return typeof value === 'object' && value !== null && 'nodes' in value;
}

const browser = await chromium.launch();
const context = await browser.newContext({
    reducedMotion: 'reduce',
    viewport: { height: 1000, width: 1600 },
});
const page = await context.newPage();
await page.goto(`file://${injected}`);
await page.waitForFunction('globalThis.dumpMore !== undefined');
await page.evaluate('document.fonts.ready');
await page.waitForTimeout(800);

async function atWidth(width: number): Promise<void> {
    await page.evaluate(
        `(() => { const proto = globalThis.dumpMore.proto(); proto.S.width = ${String(width)}; proto.fit(); })()`,
    );
}

async function each(scripts: readonly string[]): Promise<unknown[]> {
    const found: unknown[] = [];
    for (const script of scripts) {
        found.push(await page.evaluate(script));
    }
    return found;
}

const graph: unknown = await page.evaluate(GRAPH);
if (!isDumped(graph)) {
    throw new Error('the prototype handed back no graph');
}
const upTo = (depth: number) => [
    null,
    ...graph.nodes.filter((node) => node.depth <= depth).map((node) => node.id),
];
await atWidth(1440);
const wide = await each(upTo(3).map((id) => stateOf(id)));
const walksWide = await each(WALKS.map((path) => walkOf(path)));
await atWidth(1100);
const narrow = await each(upTo(2).map((id) => stateOf(id)));
const walksNarrow = await each(WALKS.slice(0, 3).map((path) => walkOf(path)));
await browser.close();

writeFileSync(join(out, 'prototype-graph.json'), JSON.stringify(graph));
writeFileSync(join(out, 'prototype-states.json'), JSON.stringify({ narrow, wide }));
writeFileSync(
    join(out, 'prototype-keys.json'),
    JSON.stringify({ narrow: walksNarrow, wide: walksWide }),
);
process.stdout.write(
    `${String(graph.nodes.length)} nodes, ${String(wide.length + narrow.length)} states\n`,
);
