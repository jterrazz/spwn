import { tauriBridge } from './runtime';

/** Checks for a new release and, if one exists, prompts the user to install it; true once it is installed. Called once at app startup from the shell, and false in a browser. The check is best-effort: a failure leaves the running version in place. */
export async function checkForUpdatesOnStartup(): Promise<boolean> {
    const tauri = tauriBridge();
    if (tauri?.updater === undefined || tauri.dialog === undefined || tauri.process === undefined) {
        return false;
    }

    try {
        const update = await tauri.updater.check();
        if (update?.available !== true || update.downloadAndInstall === undefined) {
            return false;
        }

        const ok = await tauri.dialog.ask(
            `spwn web ${update.version ?? ''} is available.\n\n${update.notes ?? ''}\n\nInstall now? The app will restart.`,
            { title: 'Update available', kind: 'info' },
        );
        if (!ok) {
            return false;
        }

        await update.downloadAndInstall();
        await tauri.process.relaunch();
        return true;
    } catch {
        return false;
    }
}
