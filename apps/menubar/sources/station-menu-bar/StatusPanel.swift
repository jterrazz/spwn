import AppKit
import QuartzCore
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
    private let backdrop = Backdrop()
    private let content: () -> AnyView
    private let opened: () -> Void
    private let closed: () -> Void
    private let report: ((String) -> Void)?
    private var outside: Any?
    private var anchor = CGPoint.zero
    private var placing = false
    private var pending: CGSize?

    init(content: @escaping () -> AnyView, opened: @escaping () -> Void, closed: @escaping () -> Void, report: ((String) -> Void)? = nil) {
        self.content = content
        self.opened = opened
        self.closed = closed
        self.report = report
        super.init()
        panel.identifier = PanelWindow.id
        panel.isOpaque = false
        panel.backgroundColor = .clear
        // hazard: the glass draws its own shadow past its edge, and a window
        // the glass's size clips it to the frame's square corners (the
        // owner's capture, 2026-11-11); the window's own shadow then follows
        // that square alpha. So the window is the glass grown by `shade`, a
        // clear margin the glass's shadow fades out in, and casts none itself.
        panel.hasShadow = false
        panel.level = .popUpMenu
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .transient, .ignoresCycle]
        panel.isReleasedWhenClosed = false
        panel.hidesOnDeactivate = false
        glass.style = .regular
        glass.cornerRadius = Token.Glass.window
        backdrop.addSubview(glass)
        backdrop.outside = { [weak self] in self?.hide() }
        panel.contentView = backdrop
        panel.delegate = self
        panel.cancelled = { [weak self] in self?.hide() }
        item.button?.target = self
        item.button?.action = #selector(clicked(_:))
        item.button?.sendAction(on: [.leftMouseDown, .rightMouseDown])
        if report != nil {
            DispatchQueue.main.asyncAfter(deadline: .now() + 1) { [weak self] in
                guard let self, let spot = self.spot else { return }
                self.report?("item \(Self.frame(spot))")
                if let screen = self.item.button?.window?.screen { self.report?("screen \(Self.frame(screen.frame))") }
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
        anchor = CGPoint(x: min(max(spot.maxX - size.width, room.minX + Self.margin), room.maxX - Self.margin - size.width), y: spot.minY - Self.drop)
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
    // height follows the content's, so it grows downward. The content
    // reports only the height its spring ends at (the capture's trial,
    // ADR-036), so the window moves there itself, on the same curve, and
    // snaps under Reduce Motion as the content does.
    // hazard: a size the content asks while the window is being moved
    // (its layout runs inside setFrame) is held and placed right after,
    // never dropped: dropped, the window kept the cap's height around a
    // shorter content (the owner's capture, 2026-11-11).
    func fit(_ size: CGSize) {
        guard size.width > 0, size.height > 0 else { return }
        report?("content w=\(size.width) h=\(size.height)")
        guard !placing else {
            pending = size
            return
        }
        placing = true
        let glassFrame = CGRect(x: anchor.x, y: anchor.y - size.height, width: size.width, height: size.height)
        let frame = glassFrame.insetBy(dx: -Self.shade, dy: -Self.shade)
        if shown, frame != panel.frame, !NSWorkspace.shared.accessibilityDisplayShouldReduceMotion {
            NSAnimationContext.runAnimationGroup { context in
                let p = Token.Motion.smoothPoints.map { Float($0) }
                context.duration = Token.Motion.smoothDuration
                context.timingFunction = CAMediaTimingFunction(controlPoints: p[0], p[1], p[2], p[3])
                panel.animator().setFrame(frame, display: true)
            }
        } else {
            panel.setFrame(frame, display: true)
        }
        placing = false
        report?("glass \(Self.frame(glassFrame)) window \(Self.frame(frame)) key=\(panel.isKeyWindow)")
        if let next = pending {
            pending = nil
            fit(next)
        }
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

    // value: the panel's right edge under the item's, 6 pt under the menu
    // bar, as the capture read MenuBarExtra's window (run old459); the
    // least room to the screen's edge.
    static let drop: CGFloat = 6
    static let margin: CGFloat = 8
    // value: past the glass's shadow, which the capture reads fading out
    // within it on every side.
    nonisolated static let shade: CGFloat = 40
}

// invariant: the window's margin around the glass is clear and holds only
// the glass's shadow; a click there is a click outside the panel.
final class Backdrop: NSView {
    var outside: (() -> Void)?

    // hazard: not an autoresizing mask: the window opens from a zero frame,
    // where a mask records the margins as 40 and -40 and the glass then
    // overhangs the window's top and right (run k1).
    override func resizeSubviews(withOldSize _: NSSize) {
        let inner = bounds.insetBy(dx: StatusPanel.shade, dy: StatusPanel.shade)
        guard !inner.isNull else { return }
        for view in subviews { view.frame = inner }
    }

    override func mouseDown(with _: NSEvent) { outside?() }
    override func rightMouseDown(with _: NSEvent) { outside?() }
}

final class GlassPanel: NSPanel {
    var cancelled: (() -> Void)?

    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { false }
    override func cancelOperation(_: Any?) { cancelled?() }
}

// invariant: the window's frame is the panel's alone: the content's size
// never pushes the window through Auto Layout (which would grow it from
// its bottom edge), and every size the content asks, on any constraint or
// layout pass, is told to the panel, which moves to it with its top pinned.
final class PanelHosting: NSHostingView<AnyView> {
    var resized: ((CGSize) -> Void)?
    private var told = CGSize.zero

    required init(rootView: AnyView) {
        super.init(rootView: rootView)
        for axis in [NSLayoutConstraint.Orientation.horizontal, .vertical] {
            setContentHuggingPriority(.fittingSizeCompression, for: axis)
            setContentCompressionResistancePriority(.fittingSizeCompression, for: axis)
        }
    }

    @available(*, unavailable)
    required init?(coder _: NSCoder) { nil }

    override func invalidateIntrinsicContentSize() {
        super.invalidateIntrinsicContentSize()
        tell()
    }

    override func updateConstraints() {
        super.updateConstraints()
        tell()
    }

    override func layout() {
        super.layout()
        tell()
    }

    func tell() {
        let size = intrinsicContentSize
        guard size != told else { return }
        told = size
        resized?(size)
    }
}
