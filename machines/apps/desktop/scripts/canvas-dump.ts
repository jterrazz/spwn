// outside: a Linux VM only (headless Chromium); a Mac never opens a browser for this.
// value: usage: node scripts/canvas-dump.ts <spwn-wiki-proto-r1.html>: the canvas's fixtures, written in place by
// value: running the prototype's own functions: the wiki sample (src/presentation/ui/canvas/samples/) and the
// value: prototype's states and walks (its __tests__/). The prototype the owner approved lives outside this
// value: repository, at home/spwn/work/uncommitted/os-design/wiki-proto/ of the studio-os estate; its last line exports
// value: its state as window.__proto, and the dump injects the rest of what it reads before that line. The JSON is
// value: written minified: `make fix` formats it.
import { mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { chromium } from 'playwright';

const [protoArgument] = process.argv.slice(2);
if (protoArgument === undefined) {
    throw new Error('usage: node scripts/canvas-dump.ts <spwn-wiki-proto-r1.html>');
}
const canvas = resolve(import.meta.dirname, '../src/presentation/ui/canvas');
const samples = join(canvas, 'samples');
const tests = join(canvas, '__tests__');
const out = mkdtempSync(join(tmpdir(), 'canvas-dump-'));
const MARK = 'window.__proto={S,H,';
const MORE = [
    'globalThis.dumpMore={M,labelOf,tw,PILLF,kidMarks,camTarget,camClamp,pagesOf,layout,stripH,ensureVisible,',
    'needN,nodeHTML,',
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
        marks: more.kidMarks(n.id), needs: more.needN(n.id), parent: n.par,
    }));
    const widths = Object.fromEntries(nodes.map((n) => [n.label, more.tw(n.label, more.PILLF)]));
    return { nodes, roots: more.M.seats.map((s) => 'S:' + s.id), widths };
})()`;

// value: the words the prototype draws on each node, read from its own nodeHTML: a card's top line, title, line and
// value: foot, a head's count, a lead head's title and line, and each floor guide's label; the gallery draws them.
const WORDS = `(() => {
    const more = globalThis.dumpMore;
    const drawn = (id, face) => { const box = document.createElement('div'); box.innerHTML = more.nodeHTML(id, face); return box; };
    const text = (box, selector) => box.querySelector(selector)?.textContent.trim() ?? '';
    const nodes = Object.fromEntries(Object.values(more.proto().NODES).map((n) => {
        const card = drawn(n.id, 'card');
        const head = drawn(n.id, n.k === 1 ? 'region' : 'open');
        return [n.id, {
            count: text(head, '.cnt'),
            foot: (card.querySelector('.cdots')?.lastChild?.textContent ?? '').trim(),
            kicker: text(card, '.ctop > span:not(.you)'),
            lead: text(head, '.pl'),
            line: text(card, '.cl'),
            seat: n.kind === 'seat' || n.kind === 'page' ? n.seat : '',
            title: n.k === 1 ? '' : text(card, 'b'),
        }];
    }));
    const bands = Object.fromEntries(Object.keys(more.M.floors).map((f) => [f, 'E' + f + ' · ' + more.M.floors[f].name]));
    return { bands, icons: more.M.icons, nodes };
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
const words: unknown = await page.evaluate(WORDS);
await browser.close();

writeFileSync(join(samples, 'wiki-graph.json'), JSON.stringify(graph));
writeFileSync(join(samples, 'wiki-words.json'), JSON.stringify(words));
writeFileSync(join(tests, 'prototype-states.json'), JSON.stringify({ narrow, wide }));
writeFileSync(
    join(tests, 'prototype-keys.json'),
    JSON.stringify({ narrow: walksNarrow, wide: walksWide }),
);
process.stdout.write(
    `${String(graph.nodes.length)} nodes, ${String(wide.length + narrow.length)} states\n`,
);
