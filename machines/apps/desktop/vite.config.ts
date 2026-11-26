import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

import { FACES_BLOCK } from './src/presentation/ui/faces-block';
import { FACES_SHIPPED } from './src/presentation/ui/faces-shipped';

export default defineConfig({
    build: { target: 'es2022' },
    clearScreen: false,
    css: { postcss: { plugins: [FACES_SHIPPED, FACES_BLOCK] } },
    plugins: [react()],
    server: { port: 1420, strictPort: true },
});
