import AppKit
import Carbon.HIToolbox
import StationKit

// invariant: SPEC A7: ⌘K anywhere opens the panel on Find, a hot key the
// system holds for the app (`RegisterEventHotKey`), registered once; the
// owner turns it off with the `findHotKey` default set to false.
@MainActor
final class FindHotKey {
    static let shared = FindHotKey()
    static let enabledKey = "findHotKey"
    static let signature: OSType = 0x5354_464B
    static let id: UInt32 = 1

    private var ref: EventHotKeyRef?
    private var handler: EventHandlerRef?
    private var pressed: (() -> Void)?

    func register(_ pressed: @escaping () -> Void) {
        guard ref == nil, UserDefaults.standard.object(forKey: Self.enabledKey) as? Bool ?? true else { return }
        self.pressed = pressed
        var spec = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
        InstallEventHandler(GetApplicationEventTarget(), { _, _, _ in
            DispatchQueue.main.async { MainActor.assumeIsolated { FindHotKey.shared.pressed?() } }
            return noErr
        }, 1, &spec, nil, &handler)
        RegisterEventHotKey(UInt32(kVK_ANSI_K), UInt32(cmdKey), EventHotKeyID(signature: Self.signature, id: Self.id), GetApplicationEventTarget(), 0, &ref)
    }
}

extension Model {
    // invariant: ⌘K over an open panel opens Find in it; over a closed one,
    // Find is set first, so the panel's first frame is Find, then the
    // status item is clicked as the owner would (a MenuBarExtra has no API
    // to open its window).
    func findFromAnywhere() {
        find(state.find ?? FindQuery())
        guard PanelWindow.panel(NSApp.windows) == nil else { return }
        let button = NSApp.windows.lazy.filter { $0.className.contains("StatusBar") }.compactMap { PanelWindow.statusButton(in: $0.contentView) }.first
        button?.performClick(nil)
    }
}
