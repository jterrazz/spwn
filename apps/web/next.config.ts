import type { NextConfig } from 'next';
import { readFileSync } from 'node:fs';

const isStaticExport = process.env.TAURI_STATIC_BUILD === '1';

const pkg = JSON.parse(readFileSync('./package.json', 'utf8'));

const nextConfig: NextConfig = {
    ...(isStaticExport ? { output: 'export' } : {}),
    distDir: isStaticExport ? '.artifacts/next/export' : '.artifacts/next',
    allowedDevOrigins: ['local://', '*.local', '192.168.*.*', '10.*.*.*'],
    images: {
        unoptimized: true,
    },
    env: {
        NEXT_PUBLIC_APP_VERSION: pkg.version,
    },
};

export default nextConfig;
