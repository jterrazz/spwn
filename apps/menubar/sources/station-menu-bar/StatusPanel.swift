import AppKit
import StationKit
import SwiftUI

// invariant: ADR-036: the app owns its status item and the panel under
// it. The panel's one background is the system glass, an
// NSGlassEffectView tinted to the M1 density, and the SwiftUI content is
// that glass's contentView, so AppKit keeps it legible as the glass
// adapts and the owner's Liquid Glass, Reduce Transparency and Increase
// Contrast settings act on it. Nothing paints over the glass.
@MainActor
final class StatusPanel: NSObject, NSWindowDelegate {
    let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    private let panel = GlassPanel(contentRect: .zero, styleMask: [.borderless, .nonactivatingPanel], backing: .buffered, defer: true)
    private let glass = NSGlassEffectView()
    private let content: () -> AnyView
    private let opened: () -> Void
    private let closed: () -> Void
    private let report: ((String) -> Void)?
    private var outside: Any?
    private var anchor = CGPoint.zero
    private var placing = false

    init(content: @escaping () -> AnyView, opened: @escaping () -> Void, closed: @escaping () -> Void, report: ((String) -> Void)? = nil) {
        self.content = content
        self.opened = opened
        self.closed = closed
        self.report = report
        super.init()
        panel.identifier = PanelWindow.id
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = true
        panel.level = .popUpMenu
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .transient, .ignoresCycle]
        panel.isReleasedWhenClosed = false
        panel.hidesOnDeactivate = false
        glass.style = .regular
        glass.cornerRadius = Token.Glass.window
        panel.contentView = glass
        panel.delegate = self
        panel.cancelled = { [weak self] in self?.hide() }
        item.button?.target = self
        item.button?.action = #selector(clicked(_:))
        item.button?.sendAction(on: [.leftMouseDown, .rightMouseDown])
        if report != nil {
            DispatchQueue.main.asyncAfter(deadline: .now() + 1) { [weak self] in
                guard let self, let spot = self.spot else { return }
                self.report?("item \(Self.frame(spot))")
            }
        }
    }

    var spot: CGRect? {
        guard let button = item.button, let bar = button.window else { return nil }
        return bar.convertToScreen(button.convert(button.bounds, to: nil))
    }

    var shown: Bool { panel.isVisible }

    @objc func clicked(_: Any?) {
        if shown { hide() } else { show() }
    }

    // invariant: each opening holds a new content, so its first frame is
    // the panel's opening (PanelOpening) and no state of the last one shows.
    func show() {
        guard !shown, let button = item.button, let spot, let bar = button.window else { return }
        let view = PanelHosting(rootView: content())
        view.sizingOptions = [.intrinsicContentSize]
        view.resized = { [weak self] size in self?.fit(size) }
        glass.tintColor = Self.tint
        glass.contentView = view
        let size = view.fittingSize
        let room = (bar.screen ?? NSScreen.main)?.visibleFrame ?? spot
        anchor = CGPoint(x: min(max(spot.minX, room.minX + Self.margin), room.maxX - Self.margin - size.width), y: spot.minY - Self.drop)
        report?("item \(Self.frame(spot))")
        fit(size)
        NSApp.activate()
        panel.makeKeyAndOrderFront(nil)
        DispatchQueue.main.async { button.highlight(true) }
        outside = NSEvent.addGlobalMonitorForEvents(matching: [.leftMouseDown, .rightMouseDown, .otherMouseDown]) { [weak self] _ in
            MainActor.assumeIsolated { self?.hide() }
        }
        report?("opened key=\(panel.isKeyWindow)")
        opened()
    }

    func hide() {
        guard shown else { return }
        panel.orderOut(nil)
        glass.contentView = nil
        if let outside { NSEvent.removeMonitor(outside) }
        outside = nil
        item.button?.highlight(false)
        report?("closed")
        closed()
    }

    // invariant: the panel's top edge stays under the item while its
    // height follows the content's spring, so it grows downward.
    func fit(_ size: CGSize) {
        guard !placing, size.width > 0, size.height > 0 else { return }
        placing = true
        let frame = CGRect(x: anchor.x, y: anchor.y - size.height, width: size.width, height: size.height)
        panel.setFrame(frame, display: true)
        placing = false
        report?("window \(Self.frame(frame)) key=\(panel.isKeyWindow)")
    }

    func windowDidResignKey(_: Notification) { hide() }

    static func frame(_ r: CGRect) -> String {
        String(format: "x=%.1f y=%.1f w=%.1f h=%.1f", r.minX, r.minY, r.width, r.height)
    }

    // value: the tint resolves per appearance, so the panel follows the
    // system's dark or light without being rebuilt.
    nonisolated static let tint = NSColor(name: nil) { appearance in
        let dark = appearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
        let color = NSColor(Token.Glass.tint(dark ? .dark : .light))
        return Trial.tint.map { color.withAlphaComponent($0) } ?? color
    }

    // value: the gap under the menu bar and the least room to the screen's
    // edge, as MenuBarExtra's window had them.
    static let drop: CGFloat = 6
    static let margin: CGFloat = 8
}

final class GlassPanel: NSPanel {
    var cancelled: (() -> Void)?

    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { false }
    override func cancelOperation(_: Any?) { cancelled?() }
}

// value: tells the panel each new size the content asks, on the same
// layout pass, so the window follows the spring frame by frame.
final class PanelHosting: NSHostingView<AnyView> {
    var resized: ((CGSize) -> Void)?

    override func invalidateIntrinsicContentSize() {
        super.invalidateIntrinsicContentSize()
        resized?(intrinsicContentSize)
    }
}
