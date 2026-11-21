import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

import { FACES_BLOCK } from './src/presentation/ui/faces-block';

export default defineConfig({
    build: { target: 'es2022' },
    clearScreen: false,
    css: { postcss: { plugins: [FACES_BLOCK] } },
    plugins: [react()],
    server: { port: 1420, strictPort: true },
});
