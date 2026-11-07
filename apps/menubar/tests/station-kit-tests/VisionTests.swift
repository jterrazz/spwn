import StationKit
import SwiftUI
import XCTest

// invariant: L50e: each of the vision's seven menu-bar panels, drawn with
// the vision's own data at 340 pt over its desk, light and dark; its PNG
// sits beside the vision's (station-design/vision-panels) for the eye.
@MainActor
final class VisionTests: XCTestCase {
    static let data = Tree.goldens.appendingPathComponent("vision")

    func page(_ name: String) throws -> Page {
        try Page.decode(Data(contentsOf: Self.data.appendingPathComponent(name + ".json")))
    }

    func drawn(_ name: String, _ state: PanelState) throws {
        let view = MenuPage.view(state, scrolls: false) ?? AnyView(Popover(state, scrolls: false))
        XCTAssertNil(try Golden.check("vision." + name, view))
    }

    func testHomeOnThisMac() throws {
        try drawn("a.home", PanelState(page: try page("a.home"), connection: .live))
    }

    func testTheSwitcherOpen() throws {
        let home = try page("a.home")
        try drawn("b.switcher", PanelState(page: home, connection: .live, pushed: try page("b.switcher"), recent: ["infra"]))
    }

    func testAWorkspace() throws {
        try drawn("c.linux-1", PanelState(page: nil, connection: .live, pushed: try page("c.linux-1")))
    }

    func testAPortItsQROpen() throws {
        try drawn("d.port-qr", PanelState(page: try page("d.port-qr"), connection: .live, qr: "https://jterrazz-macbook-pro.tail77a797.ts.net:40001"))
    }

    func testAwakeEveryMac() throws {
        try drawn("e.awake", PanelState(page: try page("e.awake"), connection: .live))
    }

    func testAServer() throws {
        try drawn("f.infra", PanelState(page: nil, connection: .live, pushed: try page("f.infra")))
    }

    func testThePortsOfAMacTwoClasses() throws {
        try drawn("g.ports-mini", PanelState(page: try page("g.ports-mini"), connection: .live))
    }
}
