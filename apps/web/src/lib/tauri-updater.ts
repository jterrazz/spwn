/**
 * Tauri auto-updater integration.
 *
 * At app launch we ask the Tauri updater plugin to check the remote manifest
 * (https://github.com/jterrazz/spwn/releases/latest/download/latest.json).
 * If a newer signed release exists, we show a native confirmation dialog
 * and, on user approval, download + install + relaunch the app.
 *
 * Safe to call in the browser - it is a no-op outside the Tauri runtime.
 */

import { isTauri } from "./tauri";

interface UpdateDescriptor {
  available: boolean;
  version?: string;
  notes?: string;
  downloadAndInstall?: () => Promise<void>;
}

interface TauriGlobal {
  updater?: {
    check: () => Promise<UpdateDescriptor | null>;
  };
  dialog?: {
    ask: (message: string, opts?: { title?: string; kind?: "info" | "warning" }) => Promise<boolean>;
  };
  process?: {
    relaunch: () => Promise<void>;
  };
}

function getTauri(): TauriGlobal | null {
  if (!isTauri()) return null;
  // @ts-expect-error Tauri global is injected at runtime only inside the native app.
  return window.__TAURI__ as TauriGlobal;
}

/**
 * Checks for a new release and, if one exists, prompts the user to install it.
 * Called once at app startup from the shell. Silently no-ops in the browser.
 */
export async function checkForUpdatesOnStartup(): Promise<void> {
  const tauri = getTauri();
  if (!tauri?.updater || !tauri.dialog || !tauri.process) return;

  try {
    const update = await tauri.updater.check();
    if (!update?.available || !update.downloadAndInstall) return;

    const ok = await tauri.dialog.ask(
      `spwn web ${update.version ?? ""} is available.\n\n${update.notes ?? ""}\n\nInstall now? The app will restart.`,
      { title: "Update available", kind: "info" },
    );
    if (!ok) return;

    await update.downloadAndInstall();
    await tauri.process.relaunch();
  } catch {
    // Never crash the app on a failed update check.
  }
}
