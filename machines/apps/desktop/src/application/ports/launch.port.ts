// value: what the launch reads and asks: how long the page has been loading (ms), the faces the window's text is drawn
// value: in, a run after a delay, and the shell's window shown. `faces` and `show` settle and never reject: a face
// value: that fails is drawn by the fallback, a show refused is the shell's hard cap's (src-tauri/src/window.rs).
export interface LaunchPort {
    since: () => number;
    faces: () => Promise<void>;
    after: (ms: number, run: () => void) => void;
    show: () => Promise<void>;
}
