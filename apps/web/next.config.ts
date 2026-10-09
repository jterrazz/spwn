import type { NextConfig } from 'next';
import { readFileSync } from 'node:fs';

const isStaticExport = process.env.TAURI_STATIC_BUILD === '1';

const manifest: unknown = JSON.parse(readFileSync('./package.json', 'utf8'));
const version: unknown =
    typeof manifest === 'object' && manifest !== null
        ? Reflect.get(manifest, 'version')
        : undefined;

const nextConfig: NextConfig = {
    ...(isStaticExport ? { output: 'export' } : {}),
    distDir: isStaticExport ? '.artifacts/next/export' : '.artifacts/next',
    allowedDevOrigins: ['127.0.0.1', 'local://', '*.local', '192.168.*.*', '10.*.*.*'],
    images: {
        unoptimized: true,
    },
    env: {
        NEXT_PUBLIC_APP_VERSION: typeof version === 'string' ? version : '',
    },
};

export default nextConfig;
