import AppKit
import SpwnMenuBar
import SwiftUI

// value: the menu bar on its own, as the rig and `make menubar` build it:
// the panel's library in a process of its own, which opens Spwn's window
// through LaunchServices and ends on Quit.
@main
struct StationMenuBarApp: App {
    @NSApplicationDelegateAdaptor(DevShell.self) private var shell

    // value: an App has a scene; the panel is the status item's, not one.
    var body: some Scene { Settings { EmptyView() } }
}

@MainActor
final class DevShell: NSObject, NSApplicationDelegate {
    func applicationDidFinishLaunching(_: Notification) {
        SpwnMenuBar.start(MenuBarShell(
            openWindow: { _ in
                guard let app = NSWorkspace.shared.urlForApplication(withBundleIdentifier: "com.jterrazz.spwn") else { return }
                NSWorkspace.shared.openApplication(at: app, configuration: NSWorkspace.OpenConfiguration())
            },
            quit: { _ in NSApplication.shared.terminate(nil) }))
    }
}
