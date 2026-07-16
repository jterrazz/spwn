import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from 'yaml';

const currentDir = dirname(fileURLToPath(import.meta.url));
const repo = resolve(currentDir, '../..');
const errors = [];

function fail(message) {
    errors.push(message);
}

function readYaml(name) {
    return parse(readFileSync(join(currentDir, name), 'utf8'));
}

function rel(path) {
    return relative(repo, path);
}

function mustExist(path, context) {
    const abs = join(repo, path);
    if (!existsSync(abs)) {
        fail(`${context}: missing ${path}`);
    }
}

function listDirs(path) {
    const abs = join(repo, path);
    return readdirSync(abs)
        .filter((entry) => statSync(join(abs, entry)).isDirectory())
        .sort();
}

function walkFiles(path) {
    const abs = join(repo, path);
    const out = [];
    for (const entry of readdirSync(abs)) {
        const child = join(abs, entry);
        const stat = statSync(child);
        if (stat.isDirectory()) {
            out.push(...walkFiles(rel(child)));
        } else {
            out.push(rel(child));
        }
    }
    return out.sort();
}

function assertRuntimeContracts() {
    const doc = readYaml('runtimes.yaml');
    for (const [runtime, cfg] of Object.entries(doc.runtimes ?? {})) {
        const runtimeDir = join(repo, 'packages/runtimes', runtime.replaceAll('-', ''));
        if (!existsSync(runtimeDir)) {
            fail(`runtime ${runtime}: expected implementation directory ${rel(runtimeDir)}`);
        }
        for (const path of cfg.docs ?? []) {
            mustExist(path, `runtime ${runtime} docs`);
        }
        for (const path of cfg.tests ?? []) {
            mustExist(path, `runtime ${runtime} tests`);
        }
        const adapterPath = join(runtimeDir, 'adapter.go');
        if (existsSync(adapterPath)) {
            const adapter = readFileSync(adapterPath, 'utf8');
            for (const facet of cfg.facets ?? []) {
                let field = 'Tool:';
                if (facet === 'render') {
                    field = 'Render:';
                } else if (facet === 'spawn') {
                    field = 'Spawn:';
                }
                if (!adapter.includes(field)) {
                    fail(`runtime ${runtime}: adapter.go missing ${field}`);
                }
            }
        }
        if (cfg.goldenOutput) {
            const cases = listDirs('packages/runtimes/testdata');
            const missing = cases.filter(
                (caseName) =>
                    !existsSync(
                        join(repo, 'packages/runtimes/testdata', caseName, cfg.goldenOutput),
                    ) &&
                    !existsSync(
                        join(
                            repo,
                            'packages/runtimes/testdata',
                            caseName,
                            `${cfg.goldenOutput}_error.txt`,
                        ),
                    ),
            );
            if (missing.length > 0) {
                fail(`runtime ${runtime}: missing ${cfg.goldenOutput} in ${missing.join(', ')}`);
            }
        }
    }

    const runtimesReadme = readFileSync(join(repo, 'packages/runtimes/README.md'), 'utf8');
    for (const stale of ['No renderer; codex', 'no renderer yet', 'future codex adapter']) {
        if (runtimesReadme.includes(stale)) {
            fail(`packages/runtimes/README.md contains stale runtime text: ${stale}`);
        }
    }
}

function assertApiRoutes() {
    const doc = readYaml('api-routes.yaml');
    const declared = new Map((doc.routes ?? []).map((entry) => [entry.route, entry]));
    const server = readFileSync(join(repo, 'apps/api/server.go'), 'utf8');
    const actual = [...server.matchAll(/mux\.HandleFunc\("(?<route>[A-Z]+ [^"]+)"/g)]
        .map((match) => match.groups.route)
        .sort();

    for (const route of actual) {
        if (!declared.has(route)) {
            fail(`api route ${route}: missing from tests/_contracts/api-routes.yaml`);
        }
    }
    for (const [route, entry] of declared) {
        if (!actual.includes(route)) {
            fail(`api route ${route}: declared but not registered in apps/api/server.go`);
        }
        if (!entry.tests || entry.tests.length === 0) {
            fail(`api route ${route}: no tests declared`);
        }
        for (const path of entry.tests ?? []) {
            mustExist(path, `api route ${route}`);
        }
    }
}

function assertCliContracts() {
    const doc = readYaml('cli-commands.yaml');
    const docsDir = join(repo, doc.generatedDocsDir);
    if (!existsSync(docsDir)) {
        fail(`cli docs: missing ${doc.generatedDocsDir}`);
        return;
    }
    const docs = readdirSync(docsDir).filter(
        (name) => name.startsWith('spwn') && name.endsWith('.md'),
    );
    if (docs.length === 0) {
        fail(`cli docs: no generated docs found in ${doc.generatedDocsDir}`);
    }
    for (const name of docs) {
        const body = readFileSync(join(docsDir, name), 'utf8');
        if (!body.includes('##') && body.trim().length < 20) {
            fail(`cli docs: ${name} looks empty`);
        }
    }
    for (const entry of doc.requiredBehaviorSpecs ?? []) {
        if (!entry.tests || entry.tests.length === 0) {
            fail(`cli command ${entry.command}: no behavior specs declared`);
        }
        for (const path of entry.tests ?? []) {
            mustExist(path, `cli command ${entry.command}`);
        }
    }
}

function assertWebContracts() {
    const doc = readYaml('web-routes.yaml');
    for (const entry of doc.routes ?? []) {
        if (!entry.tests || entry.tests.length === 0) {
            fail(`web route ${entry.path}: no tests declared`);
        }
        for (const path of entry.tests ?? []) {
            mustExist(path, `web route ${entry.path}`);
        }
    }

    for (const path of walkFiles('tests/web')) {
        if (!/\.(?:ts|tsx)$/.test(path)) {
            continue;
        }
        const body = readFileSync(join(repo, path), 'utf8');
        if (body.includes('waitForTimeout(')) {
            fail(`web test ${path}: fixed waitForTimeout is forbidden; use locator/API assertions`);
        }
    }
}

function assertCatalogContracts() {
    const doc = readYaml('catalog.yaml');
    const declared = new Map((doc.catalog ?? []).map((entry) => [entry.slug, entry]));
    const actual = listDirs('catalog');
    for (const slug of actual) {
        if (!declared.has(slug)) {
            fail(`catalog ${slug}: missing from tests/_contracts/catalog.yaml`);
        }
    }
    for (const [slug, entry] of declared) {
        mustExist(`catalog/${slug}`, `catalog ${slug}`);
        if (!entry.tests || entry.tests.length === 0) {
            fail(`catalog ${slug}: no tests declared`);
        }
        for (const path of entry.tests ?? []) {
            mustExist(path, `catalog ${slug}`);
        }
    }
}

function assertNodePackageContracts() {
    const doc = readYaml('node-packages.yaml');
    for (const entry of doc.packages ?? []) {
        const packagePath = join(repo, entry.path, 'package.json');
        if (!existsSync(packagePath)) {
            fail(`node package ${entry.path}: missing package.json`);
            continue;
        }
        const pkg = JSON.parse(readFileSync(packagePath, 'utf8'));
        for (const script of entry.scripts ?? []) {
            if (!pkg.scripts?.[script]) {
                fail(`node package ${entry.path}: missing script ${script}`);
            }
        }
        for (const path of entry.tests ?? []) {
            mustExist(path, `node package ${entry.path}`);
        }
    }
}

assertRuntimeContracts();
assertApiRoutes();
assertCliContracts();
assertWebContracts();
assertCatalogContracts();
assertNodePackageContracts();

if (errors.length > 0) {
    console.error(`test contract check failed (${errors.length})`);
    for (const error of errors) {
        console.error(`- ${error}`);
    }
    process.exit(1);
}

console.log('test contracts ok');
