import AppKit
import StationKit
import SwiftUI
import XCTest

@MainActor
final class PortsTests: XCTestCase {
    func texts(_ values: [Value]?) -> [String] { (values ?? []).compactMap(\.text) }

    // invariant: PO-33: a web share's row opens its link in a click, its
    // QR beside shows the QR of that link, and Stop sharing is a y/n whose
    // yes is the ports.stop-sharing call of that port.
    func testAWebShareOpensItsLinkAndShowsItsQR() async throws {
        let page = try Tree.page("menubar.ports.linux-1")
        let actions = page.actions ?? []
        let row = try XCTUnwrap(page.section("private")?.rows.first)
        let url = "https://jterrazz-macbook-pro.tail77a797.ts.net:40001"
        XCTAssertEqual(PortRow.link(row, in: actions)?.run?.open, url)
        let qr = try XCTUnwrap(page.action(row.quick ?? ""))
        XCTAssertEqual(PortsSection.qr(qr, of: row, in: actions), url)
        let stop = try XCTUnwrap(page.actions(for: row.id ?? "").first { $0.label == "Stop sharing" })
        XCTAssertEqual(stop.confirm?.question, "Stop sharing linux-1's port 5173?")
        XCTAssertEqual(stop.run?.call, "ports.stop-sharing")
        XCTAssertEqual(Client.input([:], fixed: stop.run?.in) as NSDictionary, ["VM": "linux-1", "Port": "5173", "Mac": "jterrazz-macbook-pro", "Confirm": "yes"] as NSDictionary)
        var state = PanelState(page: page, connection: .live)
        state.confirming = stop.id
        XCTAssertNil(try Golden.check("sheet.stop-sharing", try XCTUnwrap(MenuPage.view(state, scrolls: false))))
    }

    // invariant: PO-34: mac-mini's Ports: what is exposed first, then
    // Public, then Private, each under its words; its facts count them, the
    // exposed in red.
    func testAMacsPortsSayTheirReachInWords() throws {
        let page = try Tree.page("menubar.ports.mini")
        XCTAssertEqual(page.sections?.compactMap(\.id), ["summary", "exposed", "public", "private"])
        XCTAssertEqual(page.section("exposed")?.rows.map(PortsSection.exposedTitle), ["5000 · AirPlay Receiver", "10250 · kubelet"])
        XCTAssertEqual(page.section("public").map(PortsSection.label), "Public · from outside")
        XCTAssertEqual(page.section("private").map(PortsSection.label), "Private · on your tailnet")
        XCTAssertEqual(texts(page.section("summary")?.caption), ["mac-mini", "1 public", "2 on your tailnet", "2 exposed"])
        XCTAssertEqual(page.section("summary")?.caption?.last?.signal, "failing")
        XCTAssertEqual(PortRow.addressRole.design, .monospaced)
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
    // invariant: MB-17: Show process on a red row of mac-mini's Ports opens
    // a new terminal on station ports --on mac-mini, and starts nothing else.
    func testShowProcessOpensTerminal() async throws {
        let page = try Tree.page("menubar.ports.mini")
        let red = try XCTUnwrap(page.section("exposed")?.rows.first)
        let show = try XCTUnwrap(page.action(red.menu?.last ?? ""))
        XCTAssertEqual(red.menu?.first, "declare-public")
        XCTAssertEqual(show.label, "Show process")
        let calls = Recorded()
        let status = await ActionRunner(executable: "/u/bin/station", socket: "/nonexistent", launcher: RecordingLauncher(calls: calls)).run(show)
        XCTAssertNil(status)
        XCTAssertEqual(calls.all, [
            ["/usr/bin/osascript", "-e", #"tell application "Terminal" to do script "'/u/bin/station' 'ports' '--on' 'mac-mini'""#, "-e", #"tell application "Terminal" to activate"#],
        ])
    }
}
