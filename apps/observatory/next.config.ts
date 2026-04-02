import type { NextConfig } from "next";

// Only use static export for explicit Tauri production builds (npm run build:static)
const isStaticExport = process.env.TAURI_STATIC_BUILD === "1";

const nextConfig: NextConfig = {
  ...(isStaticExport ? { output: "export" } : {}),
  images: {
    unoptimized: true,
  },
};

export default nextConfig;
