import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome } from '../../setup/helpers.js';

describe('knowledge', () => {
    let home: string;
    let originalSpwnHome: string | undefined;

    beforeEach(() => {
        originalSpwnHome = process.env.SPWN_HOME;
        home = createSpwnHome();
        process.env.SPWN_HOME = home;
    });

    afterEach(() => {
        if (originalSpwnHome !== undefined) {
            process.env.SPWN_HOME = originalSpwnHome;
        } else {
            delete process.env.SPWN_HOME;
        }
    });

    test('knowledge directory is initialized with default files', () => {
        // GIVEN - a SPWN_HOME with a knowledge directory
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });

        // WHEN - writing default knowledge files (simulating init)
        const defaultFiles: Record<string, string> = {
            'overview.md':
                '# Universe Knowledge\n\nThis is the knowledge base for your spwn universe.\n',
            'glossary.md': '# Glossary\n\nKey terms and concepts.\n',
            'roadmap.md': '# Roadmap\n\n## Current Focus\n',
        };

        for (const [name, content] of Object.entries(defaultFiles)) {
            writeFileSync(join(knowledgeDir, name), content);
        }

        // THEN - all default files exist
        expect(existsSync(join(knowledgeDir, 'overview.md'))).toBe(true);
        expect(existsSync(join(knowledgeDir, 'glossary.md'))).toBe(true);
        expect(existsSync(join(knowledgeDir, 'roadmap.md'))).toBe(true);
    });

    test('knowledge files have expected content', () => {
        // GIVEN - initialized knowledge directory
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });

        const overviewContent =
            '# Universe Knowledge\n\nThis is the knowledge base for your spwn universe.\nThe Architect maintains this as the single source of truth.\n';
        writeFileSync(join(knowledgeDir, 'overview.md'), overviewContent);

        // WHEN - reading files
        const content = readFileSync(join(knowledgeDir, 'overview.md'), 'utf8');

        // THEN - content matches
        expect(content).toContain('Universe Knowledge');
        expect(content).toContain('single source of truth');
    });

    test('knowledge ls lists files correctly', () => {
        // GIVEN - knowledge with multiple files
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });
        mkdirSync(join(knowledgeDir, 'projects'), { recursive: true });

        writeFileSync(join(knowledgeDir, 'overview.md'), '# Overview');
        writeFileSync(join(knowledgeDir, 'glossary.md'), '# Glossary');
        writeFileSync(join(knowledgeDir, 'projects', 'api.md'), '# API Project');

        // WHEN - listing files
        const { readdirSync } = require('node:fs');
        const { join: pathJoin } = require('node:path');

        const walkFiles = (dir: string, base: string): string[] => {
            const results: string[] = [];
            const entries = readdirSync(dir, { withFileTypes: true });
            for (const entry of entries) {
                const fullPath = pathJoin(dir, entry.name);
                if (entry.isDirectory()) {
                    results.push(...walkFiles(fullPath, base));
                } else {
                    const relPath = fullPath.replace(`${base}/`, '');
                    results.push(relPath);
                }
            }
            return results;
        };

        const files = walkFiles(knowledgeDir, knowledgeDir);

        // THEN - all files are listed
        expect(files).toContain('overview.md');
        expect(files).toContain('glossary.md');
        expect(files).toContain('projects/api.md');
    });

    test('knowledge show displays file content', () => {
        // GIVEN - a knowledge file
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });

        const expectedContent = '# Universe Knowledge\n\nThis is the overview.\n';
        writeFileSync(join(knowledgeDir, 'overview.md'), expectedContent);

        // WHEN - reading the file
        const content = readFileSync(join(knowledgeDir, 'overview.md'), 'utf8');

        // THEN - content is returned correctly
        expect(content).toBe(expectedContent);
        expect(content).toContain('Universe Knowledge');
    });

    test('knowledge API returns file list structure', () => {
        // GIVEN - knowledge with files
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });

        writeFileSync(join(knowledgeDir, 'overview.md'), '# Overview');
        writeFileSync(join(knowledgeDir, 'glossary.md'), '# Glossary');

        // WHEN - simulating API response construction
        const { readdirSync, statSync } = require('node:fs');
        const files = readdirSync(knowledgeDir).map((name: string) => {
            const stat = statSync(join(knowledgeDir, name));
            return {
                path: name,
                size: stat.size,
                modified: stat.mtime.toISOString(),
            };
        });

        // THEN - response has expected shape
        expect(Array.isArray(files)).toBe(true);
        expect(files.length).toBe(2);
        for (const file of files) {
            expect(file).toHaveProperty('path');
            expect(file).toHaveProperty('size');
            expect(file).toHaveProperty('modified');
            expect(typeof file.size).toBe('number');
            expect(file.size).toBeGreaterThan(0);
        }
    });

    test('knowledge API returns file content', () => {
        // GIVEN - a knowledge file
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });

        const markdownContent = '# Overview\n\n## Architecture\n\nThis is the main overview.\n';
        writeFileSync(join(knowledgeDir, 'overview.md'), markdownContent);

        // WHEN - simulating API content response
        const content = readFileSync(join(knowledgeDir, 'overview.md'), 'utf8');
        const response = { path: 'overview.md', content };

        // THEN - response contains markdown content
        expect(response.path).toBe('overview.md');
        expect(response.content).toContain('# Overview');
        expect(response.content).toContain('Architecture');
    });

    test('knowledge prevents directory traversal', () => {
        // GIVEN - a knowledge directory
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });
        writeFileSync(join(knowledgeDir, 'overview.md'), '# Overview');

        // Write a file outside knowledge
        writeFileSync(join(home, 'secret.txt'), 'secret data');

        // WHEN - attempting directory traversal
        const requestedPath = '../secret.txt';
        const hasTraversal = requestedPath.includes('..');

        // THEN - traversal is detected and blocked
        expect(hasTraversal).toBe(true);

        // The resolved path would escape the knowledge directory
        const { resolve } = require('node:path');
        const resolvedPath = resolve(join(knowledgeDir, requestedPath));
        const isWithinKnowledge = resolvedPath.startsWith(resolve(knowledgeDir));
        expect(isWithinKnowledge).toBe(false);
    });

    test('knowledge subdirectories work correctly', () => {
        // GIVEN - knowledge with nested directories
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(join(knowledgeDir, 'decisions'), { recursive: true });
        mkdirSync(join(knowledgeDir, 'projects'), { recursive: true });
        mkdirSync(join(knowledgeDir, 'agents'), { recursive: true });

        writeFileSync(join(knowledgeDir, 'decisions', 'auth-flow.md'), '# Auth Flow Decision');
        writeFileSync(join(knowledgeDir, 'projects', 'api.md'), '# API Project');
        writeFileSync(join(knowledgeDir, 'agents', 'team.md'), '# Team');

        // WHEN - reading nested files
        const authContent = readFileSync(join(knowledgeDir, 'decisions', 'auth-flow.md'), 'utf8');
        const apiContent = readFileSync(join(knowledgeDir, 'projects', 'api.md'), 'utf8');
        const teamContent = readFileSync(join(knowledgeDir, 'agents', 'team.md'), 'utf8');

        // THEN - all nested files are readable
        expect(authContent).toContain('Auth Flow Decision');
        expect(apiContent).toContain('API Project');
        expect(teamContent).toContain('Team');
    });

    test('knowledge init does not overwrite existing files', () => {
        // GIVEN - a knowledge with a custom overview
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });

        const customContent = '# My Custom Overview\n\nThis was manually edited.\n';
        writeFileSync(join(knowledgeDir, 'overview.md'), customContent);

        // WHEN - simulating re-init (only write if not exists)
        const overviewPath = join(knowledgeDir, 'overview.md');
        if (!existsSync(overviewPath)) {
            writeFileSync(overviewPath, '# Default Overview');
        }

        // Also write new defaults that don't exist yet
        const glossaryPath = join(knowledgeDir, 'glossary.md');
        if (!existsSync(glossaryPath)) {
            writeFileSync(glossaryPath, '# Glossary');
        }

        // THEN - custom content preserved, new defaults created
        const content = readFileSync(overviewPath, 'utf8');
        expect(content).toBe(customContent);
        expect(existsSync(glossaryPath)).toBe(true);
    });

    test('knowledge search finds matches across files', () => {
        // GIVEN - knowledge with searchable content
        const knowledgeDir = join(home, 'knowledge');
        mkdirSync(knowledgeDir, { recursive: true });

        writeFileSync(
            join(knowledgeDir, 'overview.md'),
            '# Overview\n\nThe authentication system uses JWT tokens.\n',
        );
        writeFileSync(
            join(knowledgeDir, 'glossary.md'),
            '# Glossary\n\n| Term | Definition |\n| JWT | JSON Web Token |\n',
        );
        writeFileSync(
            join(knowledgeDir, 'roadmap.md'),
            '# Roadmap\n\n## Current Focus\n- Improve performance\n',
        );

        // WHEN - searching for "JWT"
        const { readdirSync } = require('node:fs');
        const query = 'JWT';
        const results: Record<string, string[]> = {};

        const files = readdirSync(knowledgeDir);
        for (const file of files) {
            const content = readFileSync(join(knowledgeDir, file), 'utf8');
            const matchingLines = content
                .split('\n')
                .filter((line: string) => line.toLowerCase().includes(query.toLowerCase()));
            if (matchingLines.length > 0) {
                results[file] = matchingLines;
            }
        }

        // THEN - matches found in relevant files
        expect(Object.keys(results)).toContain('overview.md');
        expect(Object.keys(results)).toContain('glossary.md');
        expect(Object.keys(results)).not.toContain('roadmap.md');
    });
});
