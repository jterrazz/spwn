import AppKit
import QuartzCore
import StationKit
import SwiftUI

// invariant: ADR-036 and ADR-053: the app owns its status item and the
// panel under it. The panel's one background is the system glass, an
// NSGlassEffectView with no tint of its own, over a ground of the system's
// window colour at Token.Glass.ground's alpha; the SwiftUI content lies on
// that ground, inside the glass's contentView, so AppKit keeps it legible
// as the glass adapts and the owner's Liquid Glass, Reduce Transparency and
// Increase Contrast settings act on it.
@MainActor
final class StatusPanel: NSObject, NSWindowDelegate {
    let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    private let panel = KeyPanel(contentRect: .zero, styleMask: [.borderless, .nonactivatingPanel], backing: .buffered, defer: true)
    private let glass = NSGlassEffectView()
    private let ground = Ground()
    private let backdrop = Backdrop()
    private let opening = OpeningLog()
    private let content: () -> AnyView
    private let opened: () -> Void
    private let closed: () -> Void
    private let report: ((String) -> Void)?
    private let generation: (() -> String)?
    private var outside: Any?
    private var press: Any?
    private var anchor = CGPoint.zero
    private var placing = false
    private var pending: CGSize?

    init(content: @escaping () -> AnyView, opened: @escaping () -> Void, closed: @escaping () -> Void, report: ((String) -> Void)? = nil, generation: (() -> String)? = nil) {
        self.content = content
        self.opened = opened
        self.closed = closed
        self.report = report
        self.generation = generation
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
        glass.contentView = ground
        ground.wantsLayer = true
        ground.layer?.cornerRadius = Token.Glass.window
        ground.layer?.cornerCurve = .continuous
        ground.layer?.masksToBounds = true
        backdrop.addSubview(glass)
        backdrop.outside = { [weak self] in self?.hide() }
        panel.contentView = backdrop
        panel.delegate = self
        panel.cancelled = { [weak self] in self?.hide() }
        item.button?.target = self
        item.button?.action = #selector(clicked(_:))
        item.button?.sendAction(on: [.leftMouseDown, .rightMouseDown])
        // hazard: the status button tracks a click it receives and, when the click is
        // released, puts its highlight back as it was before the press, so the pill went
        // out for about a frame after each opening (capture runs mb1b-mut2, mb1b-cap6).
        // The press on the item is taken before the button sees it, and the highlight is
        // driven by the panel alone: lit while it is shown, out when it hides.
        press = NSEvent.addLocalMonitorForEvents(matching: [.leftMouseDown, .rightMouseDown]) { [weak self] event in
            let window = event.windowNumber
            let point = event.locationInWindow
            let taken = MainActor.assumeIsolated { self?.take(window: window, at: point) ?? false }
            return taken ? nil : event
        }
        if report != nil { announce(tries: 60) }
    }

    private func take(window: Int, at point: CGPoint) -> Bool {
        guard let button = item.button, let bar = button.window, bar.windowNumber == window,
              button.bounds.contains(button.convert(point, from: nil)) else { return false }
        clicked(nil)
        return true
    }

    // hazard: a status item the bar has not placed yet has no screen and a frame past the
    // screen's top (y 772 on a 768 pt screen), so the rig is told its place only once it is
    // on a screen, a second at a time, instead of once at launch.
    private func announce(tries: Int) {
        DispatchQueue.main.asyncAfter(deadline: .now() + 1) { [weak self] in
            guard let self else { return }
            guard let spot = self.spot, let screen = self.item.button?.window?.screen else {
                if tries > 0 { self.announce(tries: tries - 1) }
                return
            }
            self.report?("item \(Self.frame(spot))")
            self.report?("screen \(Self.frame(screen.frame))")
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

    // invariant: each opening holds a new content, so no state of the last
    // one shows, and the panel is ordered in whole: glass and content on its
    // first frame, nothing scaled or faded, as the system's own extras open
    // (Passwords on Tahoe, no intermediate frame). It closes as they do, at once.
    func show() {
        guard !shown, let button = item.button, let spot, let bar = button.window else { return }
        let view = PanelHosting(rootView: content())
        view.sizingOptions = [.intrinsicContentSize]
        view.resized = { [weak self] size in self?.fit(size) }
        view.frame = ground.bounds
        ground.addSubview(view)
        let size = view.fittingSize
        let room = (bar.screen ?? NSScreen.main)?.visibleFrame ?? spot
        anchor = CGPoint(x: min(max(spot.maxX - size.width, room.minX + Self.margin), room.maxX - Self.margin - size.width), y: spot.minY - Self.drop)
        report?("item \(Self.frame(spot))")
        fit(size)
        // invariant: the panel is ordered front at the content's own height.
        // The first size is the cap, asked before the page is measured; the
        // content asks its own in the layout that placing the first runs,
        // and since it tells that only on the next turn, the size is read
        // here, out of every pass, or the panel opened at the cap and shrank
        // in view (run mb1-cap6: 630 pt to 455 over 21 frames).
        panel.layoutIfNeeded()
        view.tell()
        button.highlight(true)
        NSApp.activate()
        panel.makeKeyAndOrderFront(nil)
        outside = NSEvent.addGlobalMonitorForEvents(matching: [.leftMouseDown, .rightMouseDown, .otherMouseDown]) { [weak self] _ in
            MainActor.assumeIsolated { self?.hide() }
        }
        report?("opened key=\(panel.isKeyWindow)")
        if let report { opening.begin(panel, report: report, generation: generation) }
        opened()
    }

    // value: a page opened over the one shown, logged frame by frame as an
    // opening is, under the trial's flags.
    func pushed() {
        guard shown, let report else { return }
        opening.begin(panel, what: "push", report: report, generation: generation)
    }

    func hide() {
        guard shown else { return }
        opening.end()
        panel.orderOut(nil)
        for view in ground.subviews { view.removeFromSuperview() }
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
        let area = CGRect(x: anchor.x, y: anchor.y - size.height, width: size.width, height: size.height)
        let frame = area.insetBy(dx: -Self.shade, dy: -Self.shade)
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
        panel.invalidateShadow()
        report?("glass \(Self.frame(area)) window \(Self.frame(frame)) key=\(panel.isKeyWindow)")
        if let next = pending {
            pending = nil
            fit(next)
        }
    }

    func windowDidResignKey(_: Notification) { hide() }

    static func frame(_ r: CGRect) -> String {
        String(format: "x=%.1f y=%.1f w=%.1f h=%.1f", r.minX, r.minY, r.width, r.height)
    }

    // value: the panel's right edge under the item's and its top 31 pt under
    // the screen's, 5 pt under the item, as Passwords' panel hangs on Tahoe
    // (refs/passwords-tahoe); the least room to the screen's edge.
    static let drop: CGFloat = 5
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

// invariant: the ground is the glass's contentView: it paints the system's
// window colour at Token.Glass.ground over the whole glass, under the
// content it holds at its own size, and takes no click.
final class Ground: NSView {
    override func setFrameSize(_ size: NSSize) {
        super.setFrameSize(size)
        needsLayout = true
    }

    override func layout() {
        super.layout()
        for view in subviews where view.frame != bounds { view.frame = bounds }
    }

    override func draw(_: NSRect) {
        Token.Glass.ground.setFill()
        bounds.fill(using: .sourceOver)
    }

    override func viewDidChangeEffectiveAppearance() {
        super.viewDidChangeEffectiveAppearance()
        needsDisplay = true
    }
}

final class KeyPanel: NSPanel {
    var cancelled: (() -> Void)?

    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { false }
    override func cancelOperation(_: Any?) { cancelled?() }
}

// invariant: the window's frame is the panel's alone: the content's size
// never pushes the window through Auto Layout (which would grow it from
// its bottom edge), and every size the content asks, on any constraint or
// layout pass, is told to the panel, which moves to it with its top pinned.
// hazard: the glass sizes its contentView by constraints, so a window frame
// set from inside the content's own constraint or layout pass dirties them
// mid-update, and AppKit kills the app (runs mb1-sweep, mb1-sweep2): every
// size is told on the next turn of the run loop, out of every pass, one
// turn's asks coalesced into the last.
final class PanelHosting: NSHostingView<AnyView> {
    var resized: ((CGSize) -> Void)?
    private var told = CGSize.zero
    private var queued = false

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
        later()
    }

    override func updateConstraints() {
        super.updateConstraints()
        later()
    }

    override func layout() {
        super.layout()
        later()
    }

    private func later() {
        guard !queued else { return }
        queued = true
        DispatchQueue.main.async { [weak self] in
            self?.queued = false
            self?.tell()
        }
    }

    func tell() {
        let size = intrinsicContentSize
        guard size != told else { return }
        told = size
        resized?(size)
    }
}
