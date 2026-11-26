import AppKit
import StationKit
import SwiftUI

@MainActor
final class HUDWindow {
    private var panel: NSPanel?
    private var hiding: Task<Void, Never>?

    func show(_ notice: HUDNotice) {
        hiding?.cancel()
        let panel = self.panel ?? Self.make()
        self.panel = panel
        let content = NSHostingView(rootView: HUDFlight(notice).id(UUID()).stationSystemTheme().stationSystemAccessibility())
        let size = content.fittingSize
        panel.contentView = content
        panel.setFrame(Self.frame(size, on: NSScreen.screens.first ?? NSScreen.main), display: true)
        panel.orderFrontRegardless()
        hiding = Task { [weak self] in
            try? await Task.sleep(nanoseconds: UInt64(Token.Motion.freshDuration * 1_000_000_000))
            guard !Task.isCancelled else { return }
            self?.panel?.orderOut(nil)
        }
    }

    static func make() -> NSPanel {
        let panel = NSPanel(contentRect: .zero, styleMask: [.borderless, .nonactivatingPanel], backing: .buffered, defer: true)
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = false
        panel.ignoresMouseEvents = true
        panel.level = NSWindow.Level(rawValue: Int(CGWindowLevelForKey(.mainMenuWindow)) + 1)
        panel.collectionBehavior = [.canJoinAllSpaces, .stationary, .fullScreenAuxiliary, .ignoresCycle]
        panel.isReleasedWhenClosed = false
        return panel
    }

    static func frame(_ size: CGSize, on screen: NSScreen?) -> CGRect {
        guard let screen else { return CGRect(origin: .zero, size: size) }
        let top = screen.frame.maxY - screen.safeAreaInsets.top
        return CGRect(x: screen.frame.midX - size.width / 2, y: top - size.height, width: size.width, height: size.height)
    }
}
