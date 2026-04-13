import type { NextConfig } from "next";
import { readFileSync } from "node:fs";

// Only use static export for explicit Tauri production builds (npm run build:static)
const isStaticExport = process.env.TAURI_STATIC_BUILD === "1";

// Bake the app version into the bundle so the frontend can compare
// it against the latest release without depending on the CLI binary.
const pkg = JSON.parse(readFileSync("./package.json", "utf-8"));

const nextConfig: NextConfig = {
  ...(isStaticExport ? { output: "export" } : {}),
  // Allow LAN devices to access the dev server (HMR, webpack, etc.)
  allowedDevOrigins: ["local://", "*.local", "192.168.*.*", "10.*.*.*"],
  images: {
    unoptimized: true,
  },
  env: {
    NEXT_PUBLIC_APP_VERSION: pkg.version,
  },
};

export default nextConfig;
