@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class KeyboardTests: XCTestCase {
    var home: PanelState { Stories["home.needs"].state }
    var mac: PanelState { Stories["mac.this.overview"].state }

    func testHomeWalksItsRowsAndControlsInReadingOrder() {
        let rows: [PanelFocus] = [
            .row("exposed:macbook-pro:5000"), .row("infra"),
            .row("linux-1"), .row("agent-7f"), .row("linux-3"), .row("review-bot"), .row("api"),
            .row("macbook-pro"), .row("mac-mini"), .row("macbook-air"),
            .row("awake"), .row("ports"), .row("accounts"),
        ]
        XCTAssertEqual(PanelOrder.arrows(home), rows)
        XCTAssertEqual(PanelOrder.walk(home), [.control("new-vm"), .menu, .find] + rows)
    }

    func testAMachinePageWalksTheRowsThatAct() {
        XCTAssertEqual(PanelOrder.walk(mac), [
            .row("linux-1"), .row("agent-7f"), .row("api"), .row("awake"),
            .row("upkeep:macbook-pro/settings/3"), .row("upkeep:macbook-pro/background/1"), .row("upkeep:macbook-pro/accounts/0"), .row("upkeep:macbook-pro/apps/2"),
        ])
        XCTAssertEqual(PanelOrder.walk(Stories["workspace.overview"].state), [
            .row("linux-1:5173"), .row("linux-1:3000"), .row("linux-1:4321"),
        ])
    }

    func testASheetTrapsTheWalkInItsFieldsThenItsButtons() {
        XCTAssertEqual(PanelRegion.of(Stories["sheet.new-vm"].state), .sheet)
        XCTAssertEqual(PanelOrder.walk(Stories["sheet.new-vm"].state), [.field("On"), .field("Options"), .cancel, .confirm])
        XCTAssertEqual(PanelOrder.arrows(Stories["sheet.new-vm"].state), [])
    }

    func testAnAlertTrapsTheWalkInItsNameThenItsButtons() {
        XCTAssertEqual(PanelRegion.of(Stories["alert.restart-server"].state), .alert)
        XCTAssertEqual(PanelOrder.walk(Stories["alert.restart-server"].state), [.cancel, .confirm])
        XCTAssertEqual(PanelOrder.walk(Stories["alert.delete-server.typed"].state), [.typed, .cancel, .confirm])
    }

    func testFindWalksItsResultsAndItsShowMoreLines() {
        let find = Stories["find.keyboard"].state
        XCTAssertEqual(PanelRegion.of(find), .find)
        XCTAssertEqual(PanelOrder.walk(find), [
            .row("linux-1"), .row("linux-3"),
            .row("shared:linux-1:5173"), .row("shared:linux-1:3000"), .row("shared:linux-3:8080"), .more("ports"),
            .row("attach:linux-1"), .row("share:linux-1"), .row("grant:linux-1"), .more("actions"),
        ])
    }

    func testTabWrapsInsideItsRegionAndTheArrowsStopAtTheEnds() {
        let walk = PanelOrder.walk(home)
        XCTAssertEqual(PanelOrder.walked(nil, by: 1, in: walk), .control("new-vm"))
        XCTAssertEqual(PanelOrder.walked(nil, by: -1, in: walk), .row("accounts"))
        XCTAssertEqual(PanelOrder.walked(.find, by: 1, in: walk), .row("exposed:macbook-pro:5000"))
        XCTAssertEqual(PanelOrder.walked(.row("accounts"), by: 1, in: walk), .control("new-vm"))
        XCTAssertEqual(PanelOrder.walked(.control("new-vm"), by: -1, in: walk), .row("accounts"))
        let arrows = PanelOrder.arrows(home)
        XCTAssertEqual(PanelOrder.moved(nil, by: 1, in: arrows), .row("exposed:macbook-pro:5000"))
        XCTAssertEqual(PanelOrder.moved(.row("infra"), by: 1, in: arrows), .row("linux-1"))
        XCTAssertEqual(PanelOrder.moved(.row("accounts"), by: 1, in: arrows), .row("accounts"))
        XCTAssertEqual(PanelOrder.moved(.row("exposed:macbook-pro:5000"), by: -1, in: arrows), .row("exposed:macbook-pro:5000"))
    }

    func testTheHighlightFollowsItsRowByIdentityAndGoesWithIt() {
        let mini = PanelFocus.row("mac-mini")
        XCTAssertEqual(PanelOrder.follow(mini, in: PanelOrder.walk(home)), mini)
        var reordered = home
        reordered.page?.sections = home.page?.sections?.map { s in
            var s = s
            if s.id == HomePage.macs { s.items = s.items?.reversed() }
            return s
        }
        XCTAssertEqual(PanelOrder.follow(mini, in: PanelOrder.walk(reordered)), mini)
        XCTAssertNil(PanelOrder.follow(mini, in: PanelOrder.walk(mac)))
        XCTAssertNil(PanelOrder.follow(.panel, in: PanelOrder.walk(home)))
    }

    func testTheStoryDrawsTheAccountsRowLitByTheKeyboard() {
        XCTAssertEqual(Stories["home.keyboard"].ui.highlight.map(PanelFocus.init), .row("accounts"))
        XCTAssertEqual(PanelFocus("linux-1"), .row("linux-1"))
    }

    func testARowUnderAStillPointerWaitsForItToMoveThreePoints() {
        XCTAssertFalse(PointerRule.releases(from: .zero, to: CGPoint(x: 2, y: 2)))
        XCTAssertTrue(PointerRule.releases(from: .zero, to: CGPoint(x: 3, y: 0)))
        XCTAssertTrue(PointerRule.releases(from: CGPoint(x: 10, y: 10), to: CGPoint(x: 10, y: 6)))
    }

    func testEveryKeyOfSpecA7DoesItsOneThing() {
        var menu = home
        menu.menu = true
        let qr = Stories["pop.qr"].state
        let find = Stories["find.keyboard"].state
        let sheet = Stories["sheet.new-vm"].state
        let alert = Stories["alert.restart-server"].state
        let typed = Stories["alert.delete-server.typed"].state
        let workspace = Stories["workspace.overview"].state
        var overHome = home
        overHome.pushed = mac.shown
        let table: [(key: String, on: String, state: PanelState, focus: PanelFocus?, does: PanelEffect)] = [
            ("esc", "a menu", menu, nil, .dismiss(.menu)),
            ("esc", "a popover", qr, nil, .dismiss(.popover)),
            ("esc", "Find", find, nil, .dismiss(.find)),
            ("esc", "a sheet", sheet, nil, .dismiss(.sheet)),
            ("esc", "an alert", alert, nil, .dismiss(.sheet)),
            ("esc", "a page", mac, nil, .pop),
            ("esc", "home", home, nil, .close),
            ("cmd+[", "a page", mac, nil, .pop),
            ("cmd+k", "home", home, nil, .find("")),
            ("l", "home", home, nil, .find("l")),
            ("l", "a page", mac, nil, .none),
            ("down", "home", home, nil, .move(1)),
            ("up", "home", home, nil, .move(-1)),
            ("tab", "home", home, nil, .walk(1)),
            ("backtab", "home", home, nil, .walk(-1)),
            ("left", "a page", mac, nil, .pop),
            ("left", "home", home, nil, .none),
            ("enter", "a server row", home, .row("infra"), .open("menubar.machine:infra")),
            ("enter", "a need row", Stories["home.down-and-exposed"].state, .row("exposed:macbook-pro:5000"), .run("exposed:macbook-pro:5000")),
            ("enter", "a workspace row", mac, .row("linux-1"), .open("menubar.machine:linux-1")),
            ("enter", "a shortcut row", home, .row("accounts"), .open("menubar.accounts")),
            ("enter", "the + button", home, .control("new-vm"), .run("new-vm")),
            ("enter", "nothing lit", home, nil, .none),
            ("space", "a shortcut row", home, .row("accounts"), .open("menubar.accounts")),
            ("cmd+enter", "a workspace row", mac, .row("linux-1"), .run("attach:linux-1")),
            ("cmd+enter", "a workspace's page", workspace, nil, .run("open-station")),
            ("cmd+enter", "a Mac's page", mac, nil, .none),
            ("cmd+n", "home", home, nil, .run("new-vm")),
            ("cmd+t", "home", home, nil, .run("open-station")),
            ("cmd+0", "a page", mac, nil, .home),
            ("cmd+0", "home", home, nil, .none),
            ("cmd+2", "home", home, nil, .open("menubar.machine:mac-mini")),
            ("cmd+2", "a page over home", overHome, nil, .open("menubar.machine:mac-mini")),
            ("cmd+9", "home", home, nil, .none),
            ("cmd+i", "home", home, nil, .open(Navigation.activity)),
            ("cmd+p", "home", home, nil, .open("menubar.ports")),
            ("cmd+u", "home", home, nil, .open("menubar.accounts")),
            ("cmd+e", "home", home, nil, .open("menubar.awake")),
            ("cmd+i", "a Mac's page", mac, nil, .segment("activity")),
            ("cmd+p", "a Mac's page", mac, nil, .segment("ports")),
            ("cmd+u", "a Mac's page", mac, nil, .segment("accounts")),
            ("enter", "a sheet", sheet, nil, .answer(true)),
            ("tab", "a sheet", sheet, nil, .walk(1)),
            ("enter", "an alert", alert, nil, .answer(true)),
            ("enter", "an alert asking a name", typed, nil, .answer(false)),
            ("space", "an alert's Cancel", alert, .cancel, .answer(false)),
            ("down", "Find", find, nil, .move(1)),
            ("enter", "Find", find, nil, .pick(alternate: false)),
            ("cmd+enter", "Find", find, nil, .pick(alternate: true)),
            ("l", "Find", find, nil, .none),
            ("down", "a menu", menu, nil, .none),
        ]
        let wrong = table.compactMap { row -> String? in
            let did = PanelKeys.effect(row.key, row.state, focus: row.focus)
            return did == row.does ? nil : "\(row.key) on \(row.on): \(did), not \(row.does)"
        }
        XCTAssertEqual(wrong, [])
    }
}
