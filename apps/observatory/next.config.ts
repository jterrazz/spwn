import type { NextConfig } from "next";

const isTauri = process.env.TAURI_ENV_PLATFORM !== undefined;

const nextConfig: NextConfig = {
  // Static export for Tauri (native app bundles static files)
  ...(isTauri ? { output: "export" } : {}),

  // Allow images from any source
  images: {
    unoptimized: true,
  },
};

export default nextConfig;
