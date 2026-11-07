import AppKit

// invariant: the panel closes as a click on its icon closes it: through
// the status item's own button, so the menu bar extra knows it is closed
// and the next click on the icon opens it again. A bare close of its
// window leaves the extra thinking it is shown, and spends that click.
// While it is shown it is the key window, so its first click acts.
@MainActor
public enum PanelWindow {
    public static func panel(_ windows: [NSWindow]) -> NSWindow? {
        windows.first { $0.isVisible && $0.className.contains("MenuBarExtra") }
    }

    // value: when the panel opened is made key, in seconds: it appears
    // before macOS shows its window, so the first tries find no window
    // shown yet, and Home, which nothing pushes, would never take a key.
    public static let tries = Token.Motion.keyTries

    public static func key(_ windows: [NSWindow]) {
        guard let panel = panel(windows), !panel.isKeyWindow else { return }
        panel.makeKey()
    }

    public static func close(_ windows: [NSWindow]) {
        guard let panel = panel(windows) else { return }
        let button = windows.lazy.filter { $0.className.contains("StatusBar") }.compactMap { statusButton(in: $0.contentView) }.first
        close(panel: panel, button: button)
    }

    public static func close(panel: NSWindow, button: NSButton?) {
        if let button {
            button.performClick(nil)
        } else {
            panel.close()
        }
    }

    public static func statusButton(in view: NSView?) -> NSStatusBarButton? {
        guard let view else { return nil }
        if let button = view as? NSStatusBarButton { return button }
        for sub in view.subviews {
            if let found = statusButton(in: sub) { return found }
        }
        return nil
    }
}
