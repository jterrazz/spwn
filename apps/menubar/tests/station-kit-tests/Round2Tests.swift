import AppKit
@testable import StationKit
import XCTest

final class Clicks: NSObject {
    var n = 0
    @objc func clicked(_: Any?) { n += 1 }
}

@MainActor
final class Round2Tests: XCTestCase {
    // invariant: the panel closes through the status item's button, so the
    // next click on the icon opens it; its window is never closed behind
    // the extra's back.
    func testThePanelClosesThroughItsStatusButton() {
        let clicks = Clicks()
        let button = NSStatusBarButton(frame: .zero)
        button.target = clicks
        button.action = #selector(Clicks.clicked(_:))
        let holder = NSView()
        let inner = NSView()
        holder.addSubview(inner)
        inner.addSubview(button)
        XCTAssertTrue(PanelWindow.statusButton(in: holder) === button)
        let panel = NSWindow(contentRect: .zero, styleMask: [], backing: .buffered, defer: true)
        panel.isReleasedWhenClosed = false
        PanelWindow.close(panel: panel, button: button)
        XCTAssertEqual(clicks.n, 1)
    }

    // invariant: a status line is about the page it was said on: another
    // page shown clears it, the same page read again keeps it.
    func testAStatusLineStaysOnThePageItWasSaidOn() {
        var state = PanelState(page: nil, connection: .live)
        state.say(Status("no machine named infra", failed: true), on: PageID.of("infra"))
        state.showing(PageID.of("infra"))
        XCTAssertNotNil(state.status)
        state.showing(Navigation.page(Navigation.ports, of: "infra"))
        XCTAssertNil(state.status)
        state.say(Status("Update all: done", failed: false), on: nil)
        state.showing(nil)
        XCTAssertNotNil(state.status)
        state.showing(PageID.find)
        XCTAssertNil(state.status)
    }

    // invariant: the picker draws chosen the term the page says is in
    // force, never Until while a 1 hr term holds, and none while the
    // keep-awake follows a run.
    func testKeepAwakeForDrawsThePickedTerm() {
        var one = Action(), four = Action(), until = Action()
        one.id = "keep:1h"; one.label = "1 hr"; one.picked = true
        four.id = "keep:4h"; four.label = "4 hr"
        until.id = "keep-until"; until.label = "Until 18:00"; until.form = Form()
        let keeps = [one, four, until]
        XCTAssertEqual(keeps.map { AwakePage.picked($0, among: keeps) }, [true, false, false])
        one.picked = nil
        let following = [one, four, until]
        XCTAssertEqual(following.map { AwakePage.picked($0, among: following) }, [false, false, false])
    }

    // invariant: a button whose sheet asks for the name reads as its label
    // and an ellipsis, live, never greyed with the instruction inside it.
    func testATypedStopIsAPlainButton() {
        var stop = Action()
        stop.id = "stop"; stop.label = "Stop"
        var c = Confirm()
        c.type = "infra"
        stop.confirm = c
        XCTAssertEqual(MachineContent.button(stop), "Stop\(Token.Glyph.ellipsis)")
        XCTAssertEqual(MachineContent.role(stop), .plain)
    }

    // invariant: a form's first text field takes the keys as it opens.
    func testAFormsFirstTextFieldTakesTheKeys() {
        var pick = Field(), name = Field()
        pick.key = "Image"; pick.kind = "select"
        name.key = "Name"; name.kind = "text"
        var f = Form()
        f.fields = [pick, name]
        XCTAssertEqual(FormSheet.first(f), "Name")
    }

    // invariant: each update line's Update all spins its own line alone and
    // says it is done once it ran.
    func testUpdateAllSaysDone() {
        var a = Action()
        a.id = "update-all:2-mise"; a.label = "Update all"; a.on = "row"
        XCTAssertTrue(ActionRunner.saysDone(a))
        XCTAssertEqual(ActionRunner.done(a, "apps.update").line, "Update all: done")
    }

    // invariant: the engine-stopped card's Start is the one Start: the
    // footer does not offer it again; the folded Needs you line ends in no
    // stray punctuation.
    func testStoppedOffersStartOnceAndTheFoldedLineIsClean() {
        var start = Action(), open = Action()
        start.id = "start"; open.id = "open-station"
        XCTAssertEqual(HomePage.footer([start, open], stopped: true).map(\.id), ["open-station"])
        XCTAssertEqual(HomePage.footer([start, open], stopped: false).map(\.id), ["start", "open-station"])
        var r1 = Row(), r2 = Row()
        r1.label = "a task is failing,"; r2.label = "x-snapshot is failing,."
        XCTAssertEqual(HomePage.folded([r1, r2]), "a task is failing\(Token.Glyph.sep)x-snapshot is failing")
    }

    // invariant: a page is swapped once the click that asked for it ended,
    // never under it.
    func testAPageSwapWaitsForTheClickToEnd() {
        var later: [() -> Void] = []
        var opened: [String] = []
        var h = PopoverHandlers()
        h.push = { opened.append($0) }
        h.back = { opened.append("back") }
        let d = h.deferred { later.append($0) }
        d.push(Navigation.awake)
        d.back()
        XCTAssertEqual(opened, [])
        later.forEach { $0() }
        XCTAssertEqual(opened, [Navigation.awake, "back"])
    }
}
