import AppKit
@testable import StationKit
import SwiftUI
import XCTest

// value: the bodies of the store's views drawn, by name, since the last reset.
@MainActor
final class Evaluations {
    private(set) var counts: [String: Int] = [:]
    func add(_ name: String) { counts[name, default: 0] += 1 }
    func reset() { counts = [:] }
    subscript(_ name: String) -> Int { counts[name] ?? 0 }
}

@MainActor
final class PanelStoreTests: XCTestCase {
    // value: a panel hosted in a window that is never ordered front, laid
    // out until its updates settle.
    var windows: [NSWindow] = []

    func host(_ panel: some View) -> NSView {
        let view = NSHostingView(rootView: AnyView(panel.stationTheme(.dark)))
        let window = NSWindow(contentRect: CGRect(x: 0, y: 0, width: Token.Space.width, height: Token.Metric.Panel.cap), styleMask: [.borderless], backing: .buffered, defer: false)
        window.isReleasedWhenClosed = false
        window.contentView = view
        windows.append(window)
        settle(view)
        return view
    }

    func settle(_ view: NSView) {
        for _ in 0..<5 {
            RunLoop.main.run(until: Date().addingTimeInterval(0.02))
            view.layoutSubtreeIfNeeded()
        }
    }

    // invariant: L4 of the menu bar's audit: each view of the store reads
    // the slices it draws, so a change of one slice draws again the views
    // that read it and no other: Home's change never draws a page pushed
    // over it, and the panel's own state never draws the route.
    func testAChangeOfOneSliceDrawsOnlyTheViewsThatReadIt() throws {
        let story = Stories["mac.mini.overview"].state
        let pushed = try XCTUnwrap(story.pushed)
        var start = story
        start.pushed = nil
        let store = PanelStore(start)
        let counted = Evaluations()
        let view = host(StorePanel(store: store, handlers: PopoverHandlers()).environment(\.stationEvaluated) { counted.add($0) })
        XCTAssertGreaterThan(counted["home"], 0, "Home was never drawn: the count reads nothing")

        func change(_ edit: (inout PanelState) -> Void) -> [String] {
            counted.reset()
            var state = store.state
            edit(&state)
            store.state = state
            settle(view)
            return counted.counts.keys.sorted()
        }

        XCTAssertEqual(change { $0.lit = "mac-mini" }, ["home"], "the panel's own state on Home")
        XCTAssertEqual(change { $0.connection = .connecting }, ["home"], "the link on Home")
        XCTAssertEqual(change { $0.pushed = pushed }, ["pushed", "route"], "a page pushed")
        XCTAssertEqual(change { $0.page = Stories["home.fine"].state.page }, [], "Home's change drew the page pushed over it")
        var reread = pushed
        reread.header?.title = "mac-mini, read again"
        XCTAssertEqual(change { $0.pushed = reread }, ["pushed"], "the page pushed, read again")
        XCTAssertEqual(change { $0.scrub = 3 }, ["pushed"], "the panel's own state on a page pushed")
        XCTAssertEqual(change { _ in }, [], "the same state written again drew a view")
        XCTAssertTrue(Set(change { $0.find = FindQuery() }).isSuperset(of: ["find", "route"]), "Find opened")
    }

    // invariant: ADR-069: Home and every page pushed over it are drawn in
    // one panel frame, the store's, which holds the height and the scroll
    // from one page to the next.
    func testHomeAndAPagePushedDrawInTheStoresOnePanelFrame() throws {
        let start = Stories["home.fine"].state
        let home = try XCTUnwrap(start.page)
        let pushed = try XCTUnwrap(Stories["mac.mini.overview"].state.pushed)
        let store = PanelStore(start)
        let frame = PanelFrame()
        let view = host(StorePanel(store: store, handlers: PopoverHandlers(), frame: frame))
        XCTAssertGreaterThan(frame.shown, 0, "Home drew no height in the store's frame")
        XCTAssertEqual(frame.page, "\(home.id ?? "")/\(home.context?.machine ?? "")", "Home drew in a frame of its own")
        var state = store.state
        state.pushed = pushed
        store.state = state
        settle(view)
        XCTAssertGreaterThan(frame.shown, 0)
        XCTAssertTrue(frame.page.hasPrefix("mac-mini/"), "the page pushed drew in a frame of its own: \(frame.page)")
    }
}
