// outside: a Linux VM only (headless Chromium); a Mac never opens a browser for this.
// value: usage: node scripts/capture.ts [<out-dir> [<world>...]]: .artifacts/captures and every recorded world by
// value: default. It builds the proof pages, serves them, and saves <world>/<page>.png for each page of the
// value: sidebar and <world>/<machine>-<tab>.png for each tab of each machine the world lists, the full height of
// value: the page, and gallery.png, the design components, with gallery/<Component>.png, one reference image per
// value: component the gallery enters, each opened by its start route (`?page=`, `?machine=&tab=`) so no capture clicks.
import { mkdirSync, readdirSync, readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { chromium } from 'playwright';
import type { Browser, Page } from 'playwright';
import { build, preview } from 'vite';

const window = resolve(import.meta.dirname, '..');
const worlds = join(window, 'src/fixtures/worlds');
const PAGES = [
    'home',
    'inbox',
    'storage',
    'machine-files',
    'catalog',
    'catalog-apps',
    'catalog-skills',
    'catalog-settings',
];
const TABS = ['Overview', 'Activity', 'Packages', 'Ports', 'Accounts', 'Storage', 'Logs'];
const PARALLEL = 6;

type Shot = { name: string; query: string; entries?: boolean };

function fieldOf(value: unknown, key: string): unknown {
    if (typeof value !== 'object' || value === null) {
        return undefined;
    }
    const field: unknown = Object.getOwnPropertyDescriptor(value, key)?.value;
    return field;
}

function listOf(value: unknown): unknown[] {
    return Array.isArray(value) ? value : [];
}

function nameOf(value: unknown): string[] {
    const name = fieldOf(value, 'Name');
    return typeof name === 'string' ? [name] : [];
}

function machinesOf(world: string): string[] {
    const recorded: unknown = JSON.parse(
        readFileSync(join(worlds, world, 'machines.list.json'), 'utf8'),
    );
    const [first] = listOf(fieldOf(recorded, 'exchanges'));
    const answers = listOf(fieldOf(first, 'answers'));
    const out = answers
        .map((answer) => fieldOf(answer, 'out'))
        .findLast((found) => found !== undefined);
    return listOf(fieldOf(fieldOf(out, 'All'), 'Macs')).flatMap((mac) => [
        ...nameOf(mac),
        ...listOf(fieldOf(mac, 'VMs')).flatMap((vm) => nameOf(vm)),
    ]);
}

function shotsOf(world: string): Shot[] {
    const pages = PAGES.map((page) => ({
        name: `${world}/${page}`,
        query: `?world=${world}&page=${page}`,
    }));
    const tabs = machinesOf(world).flatMap((machine) =>
        TABS.map((tab) => ({
            name: `${world}/${machine}-${tab.toLowerCase()}`,
            query: `?world=${world}&machine=${machine}&tab=${tab}`,
        })),
    );
    return [...pages, ...tabs];
}

async function capture(browser: Browser, base: string, out: string, shot: Shot): Promise<string[]> {
    const page = await browser.newPage({
        deviceScaleFactor: 1,
        viewport: { height: 820, width: 1280 },
    });
    const errors: string[] = [];
    page.on('pageerror', (error) => errors.push(String(error)));
    page.on('console', (message) => {
        if (message.type() === 'error') {
            errors.push(message.text());
        }
    });
    await page.goto(base + shot.query);
    await page.waitForLoadState('networkidle');
    await page.evaluate('document.fonts.ready');
    await page.waitForTimeout(400);
    const height = Number(
        await page.evaluate("document.querySelector('main > div:last-child')?.scrollHeight ?? 0"),
    );
    if (height > 700) {
        await page.setViewportSize({ height: Math.min(height + 120, 6000), width: 1280 });
        await page.waitForTimeout(200);
    }
    await page.screenshot({ path: join(out, `${shot.name}.png`) });
    if (shot.entries === true) {
        await captureEntries(page, join(out, shot.name));
    }
    await page.close();
    return errors;
}

async function captureEntries(page: Page, folder: string): Promise<void> {
    mkdirSync(folder, { recursive: true });
    for (const entry of await page.locator('[data-entry]').all()) {
        const name = (await entry.getAttribute('data-entry')) ?? 'entry';
        await entry.screenshot({ path: join(folder, `${name}.png`) });
    }
}

const [outArgument, ...named] = process.argv.slice(2);
const out = resolve(outArgument ?? join(window, '.artifacts/captures'));
const chosen = named.length > 0 ? named : readdirSync(worlds);
const proof = join(window, '.artifacts/proof');
await build({
    build: { emptyOutDir: true, outDir: proof },
    logLevel: 'warn',
    mode: 'proof',
    root: window,
});
const server = await preview({ build: { outDir: proof }, preview: { port: 0 }, root: window });
const base = (server.resolvedUrls?.local[0] ?? '').replace(/\/$/u, '');
const shots = [
    ...chosen.flatMap((world) => shotsOf(world)),
    { entries: true, name: 'gallery', query: '?page=gallery' },
];
for (const world of chosen) {
    mkdirSync(join(out, world), { recursive: true });
}
const browser = await chromium.launch();
const failed: string[] = [];
const queue = [...shots];
await Promise.all(
    Array.from({ length: PARALLEL }, async () => {
        for (let shot = queue.shift(); shot; shot = queue.shift()) {
            const errors = await capture(browser, base, out, shot).catch((error: unknown) => [
                String(error),
            ]);
            if (errors.length > 0) {
                failed.push(`${shot.name}: ${errors.join(' | ')}`);
            }
        }
    }),
);
await browser.close();
await server.close();
process.stdout.write(
    `${String(shots.length - failed.length)} of ${String(shots.length)} pages captured in ${out}\n`,
);
for (const line of failed) {
    process.stdout.write(`  ${line}\n`);
}
process.exitCode = failed.length === 0 ? 0 : 1;
