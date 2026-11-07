import AppKit
import StationKit
import SwiftUI
import XCTest

@MainActor
final class PortsTests: XCTestCase {
    func texts(_ values: [Value]?) -> [String] { (values ?? []).compactMap(\.text) }

    // invariant: PO-33: a shared port's row unfolds its link, its QR, Copy
    // link, which copies the link, Open, which opens it, and Stop sharing, a
    // y/n whose yes is the ports.stop-sharing call of that port.
    func testPortRowQR() async throws {
        let page = try Tree.page("menubar.ports.linux-1")
        let row = try XCTUnwrap(page.section("private")?.rows.first)
        let url = "https://jterrazz-macbook-pro.tail77a797.ts.net:40001"
        XCTAssertEqual(page.section("private")?.cell(row, "url")?.text, url)
        let stop = try XCTUnwrap(page.actions(for: row.id ?? "").first)
        XCTAssertEqual(stop.label, "Stop sharing")
        XCTAssertEqual(stop.confirm?.question, "Stop sharing linux-1's port 5173?")
        XCTAssertEqual(stop.run?.call, "ports.stop-sharing")
        XCTAssertEqual(Client.input([:], fixed: stop.run?.in) as NSDictionary, ["VM": "linux-1", "Port": "5173", "Mac": "jterrazz-macbook-pro", "Confirm": "yes"] as NSDictionary)
        var copied: [String] = []
        var opened: [URL] = []
        var asked: [String?] = []
        let handlers = PopoverHandlers(confirm: { asked.append($0?.id) }, open: { opened.append($0) }, copy: { copied.append($0) })
        var state = PanelState(page: page, connection: .live, qr: url)
        XCTAssertNil(try Golden.check("menubar.ports.linux-1.qr", try XCTUnwrap(MenuPage.view(state, handlers: handlers, scrolls: false))))
        state.confirming = stop.id
        XCTAssertNil(try Golden.check("sheet.stop-sharing", try XCTUnwrap(MenuPage.view(state, handlers: handlers, scrolls: false))))
        XCTAssertEqual(copied + opened.map(\.absoluteString) + asked.compactMap { $0 }, [])
        let link = PortsPage.link(try XCTUnwrap(URL(string: url)), handlers: handlers)
        XCTAssertEqual(link.map(\.label), ["Copy link", "Open"])
        link[0].tap()
        XCTAssertEqual(copied, [url])
        XCTAssertEqual(opened, [])
        link[1].tap()
        XCTAssertEqual(opened.map(\.absoluteString), [url])
    }

    // invariant: PO-34: mac-mini's Ports: red on top, then Public, then
    // Private, each row's class in words, never in colour alone.
    func testPortsTwoClasses() throws {
        let page = try Tree.page("menubar.ports.mini")
        XCTAssertEqual(page.sections?.compactMap(\.id).filter { ["needs-you", "public", "private"].contains($0) }, ["needs-you", "public", "private"])
        for (id, word) in [("needs-you", "needs you"), ("public", "public"), ("private", "private")] {
            let s = try XCTUnwrap(page.section(id))
            XCTAssertEqual(Set(s.rows.compactMap { s.cell($0, "class")?.text }), [word], id)
        }
        XCTAssertEqual(page.section("needs-you")?.rows.map(\.mark), [.broken, .broken])
        XCTAssertEqual(page.summary()?.label, "mac-mini")
        XCTAssertEqual(page.summary()?.cells?.first.map { Format.show($0) }, "5 ports · 2 need you")
    }

    // invariant: PO-45: Declare public… on mac-mini's Ports asks the port
    // and its route, red ones offered first, and sends ports.declare with
    // the port's machine and number as text.
    func testPortsDeclarePublic() throws {
        let page = try Tree.page("menubar.ports.mini")
        let declare = try XCTUnwrap(page.action("declare-public"))
        XCTAssertEqual(declare.run?.call, "ports.declare")
        let fields = declare.form?.fields ?? []
        XCTAssertEqual(fields.map(\.key), ["Port", "Via"])
        XCTAssertEqual(fields[0].options?.map(\.label), ["mac-mini:5000", "infra:10250", "mac-mini:5900", "infra:6443"])
        XCTAssertEqual(fields[1].options?.map(\.label), ["Cloudflare Tunnel", "Tailscale Funnel", "your LAN"])
        XCTAssertEqual(PortsPage.routes(declare), "To open a port on purpose, declare it public: a Cloudflare Tunnel, Tailscale Funnel or your LAN.")
        let input = Client.input(ActionRunner.filled(declare, answers: [:]), fixed: declare.run?.in)
        XCTAssertEqual(input as NSDictionary, ["Machine": "mac-mini", "Port": "5000", "Via": "cloudflare-tunnel"] as NSDictionary)
        var state = Snapshot.state("menubar.ports.mini", page)
        state.confirming = declare.id
        XCTAssertNil(try Golden.check("sheet.declare-public", try XCTUnwrap(MenuPage.view(state, scrolls: false))))
    }
    // invariant: MB-17: Review on a red row of mac-mini's Ports opens a new
    // terminal on station ports --on mac-mini, and starts nothing else.
    func testPortsReviewOpensTerminal() async throws {
        let page = try Tree.page("menubar.ports.mini")
        let red = try XCTUnwrap(page.section("needs-you")?.rows.first)
        let review = try XCTUnwrap(page.actions(for: red.id ?? "").first)
        XCTAssertEqual(review.label, "Review")
        let calls = Recorded()
        let status = await ActionRunner(executable: "/u/bin/station", socket: "/nonexistent", launcher: RecordingLauncher(calls: calls)).run(review)
        XCTAssertNil(status)
        XCTAssertEqual(calls.all, [
            ["/usr/bin/osascript", "-e", #"tell application "Terminal" to do script "'/u/bin/station' 'ports' '--on' 'mac-mini'""#, "-e", #"tell application "Terminal" to activate"#],
        ])
    }
}
