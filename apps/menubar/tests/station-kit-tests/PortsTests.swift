import AppKit
import StationKit
import SwiftUI
import XCTest

@MainActor
final class PortsTests: XCTestCase {
    func texts(_ values: [Value]?) -> [String] { (values ?? []).compactMap(\.text) }

    // invariant: PO-33: a web share's row opens its link in a click; its QR
    // beside unfolds under it the QR of that link, Copy link, which copies
    // it, Open, which opens it, and Stop sharing, a y/n whose yes is the
    // ports.stop-sharing call of that port. Its menu holds the same verbs,
    // and a raw share's holds Copy and Stop sharing.
    func testAWebShareOpensItsLinkAndShowsItsQR() async throws {
        let page = try Tree.page("menubar.ports.linux-1")
        let actions = page.actions ?? []
        let rows = try XCTUnwrap(page.section("private")?.rows)
        let row = try XCTUnwrap(rows.first)
        let url = "https://jterrazz-macbook-pro.tail77a797.ts.net:40001"
        XCTAssertEqual(PortRow.link(row, in: actions)?.run?.open, url)
        let qr = try XCTUnwrap(page.action(row.quick ?? ""))
        XCTAssertEqual(PortsSection.qr(qr, of: row, in: actions), url)
        XCTAssertNil(PortsSection.unfolds(row, shown: nil, in: actions))
        XCTAssertEqual(PortsSection.unfolds(row, shown: url, in: actions)?.absoluteString, url)
        XCTAssertNil(PortsSection.unfolds(rows[1], shown: url, in: actions))
        XCTAssertEqual(PortRow.menu(row, in: actions).map { $0?.label }, ["Open Vite", "Copy link to Vite", "Show QR for Vite", nil, "Stop sharing"])
        XCTAssertEqual(PortRow.menu(rows[1], in: actions).map { $0?.label }, ["Copy link to Port 7000", nil, "Stop sharing"])
        let stop = try XCTUnwrap(PortRow.menu(row, in: actions).last ?? nil)
        XCTAssertEqual(stop.confirm?.question, "Stop sharing linux-1's port 5173?")
        XCTAssertEqual(stop.run?.call, "ports.stop-sharing")
        XCTAssertEqual(Client.input([:], fixed: stop.run?.in) as NSDictionary, ["VM": "linux-1", "Port": "5173", "Mac": "jterrazz-macbook-pro", "Confirm": "yes"] as NSDictionary)
        var copied: [String] = []
        var opened: [URL] = []
        var asked: [String?] = []
        let handlers = PopoverHandlers(confirm: { asked.append($0?.id) }, open: { opened.append($0) }, copy: { copied.append($0) })
        var state = PanelState(page: page, connection: .live, qr: url)
        XCTAssertNil(try Golden.check("menubar.ports.linux-1.qr", try XCTUnwrap(MenuPage.view(state, handlers: handlers, scrolls: false))))
        XCTAssertEqual(copied + opened.map(\.absoluteString) + asked.compactMap { $0 }, [])
        let link = PortsPage.link(try XCTUnwrap(URL(string: url)), handlers: handlers)
        XCTAssertEqual(link.map(\.label), ["Copy link", "Open"])
        link[0].tap()
        XCTAssertEqual(copied, [url])
        XCTAssertEqual(opened, [])
        link[1].tap()
        XCTAssertEqual(opened.map(\.absoluteString), [url])
        state.qr = nil
        state.confirming = stop.id
        XCTAssertNil(try Golden.check("sheet.stop-sharing", try XCTUnwrap(MenuPage.view(state, scrolls: false))))
    }

    // invariant: SPEC A4: a running workspace's Overview has its Ports
    // card, its shares then its first listener with Share, and its menu
    // lists Share a port…, Stop and Delete, each its own call.
    func testAWorkspaceKeepsItsPortsAndItsVerbs() throws {
        let page = try Tree.page("menubar.vm.linux-1")
        let card = try XCTUnwrap(page.section("ports"))
        XCTAssertEqual(card.segment, "overview")
        XCTAssertEqual(PortsSection.label(card), "Ports")
        XCTAssertEqual(card.rows.map(\.label), ["Vite", "Port 7000"])
        XCTAssertEqual(PortRow.menu(card.rows[0], in: page.actions ?? []).last??.run?.call, "ports.stop-sharing")
        let menu = try XCTUnwrap(page.action(page.header?.buttons?.last ?? ""))
        let items = MachineHeader.menu(page, menu)
        XCTAssertEqual(items.map(\.label), ["Share a port", "Stop", "Delete", "Open in Terminal"])
        XCTAssertEqual(items.prefix(3).map { $0.run?.call }, ["ports.share", "machines.stop", "machines.delete"])
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
        let about = try XCTUnwrap(page.section("private")?.about)
        XCTAssertEqual(about.label, "About private links")
        XCTAssertEqual(about.text, "Private links open only on devices signed in to your tailnet; Public routes are a Cloudflare Tunnel or a Funnel you declared.")
        XCTAssertNil(page.section("public")?.about)
        let label = PortsSection(try XCTUnwrap(page.section("private"))).header.frame(width: 360)
        XCTAssertNil(try Golden.check("ports.private-label", AnyView(label), scheme: .dark, desk: false))
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
