import StationKit
import SwiftUI
import XCTest

@MainActor
final class KeyboardTests: XCTestCase {
    var home: PanelState { Stories["home.needs"].state }
    var mac: PanelState { Stories["mac.this.overview"].state }

    func testHomeWalksItsRowsAndControlsInReadingOrder() {
        let rows: [PanelFocus] = [
            .row("waiting:review-bot"), .row("ending:macbook-pro/linux-1/claude"),
            .control("menubar.ports"), .control("menubar.accounts"), .control("menubar.awake"),
            .row("review-bot"), .row("linux-1"), .row("linux-3"),
            .row("infra"),
            .row("macbook-pro"), .row("mac-mini"), .row("macbook-air"),
        ]
        XCTAssertEqual(PanelOrder.arrows(home), rows)
        XCTAssertEqual(PanelOrder.walk(home), [.control("new-vm"), .menu] + rows + [.find])
    }

    func testAMachinePageWalksTheRowsThatAct() {
        XCTAssertEqual(PanelOrder.walk(mac), [
            .row("ending:macbook-pro/linux-1/claude"), .row("linux-1"), .row("agent-7f"), .row("api"), .row("awake"),
            .row("upkeep:macbook-pro/settings/3"), .row("upkeep:macbook-pro/background/1"), .row("upkeep:macbook-pro/accounts/0"), .row("upkeep:macbook-pro/apps/2"),
        ])
        XCTAssertEqual(PanelOrder.walk(Stories["workspace.agents"].state), [
            .row("linux-1"), .row("docs-bot"), .row("ending:macbook-pro/linux-1/claude"), .row("linux-1:5173"), .row("linux-1:3000"), .row("linux-1:4321"),
        ])
    }

    func testASheetTrapsTheWalkInItsFieldsThenItsButtons() {
        XCTAssertEqual(PanelRegion.of(Stories["sheet.new-vm"].state), .sheet)
        XCTAssertEqual(PanelOrder.walk(Stories["sheet.new-vm"].state), [.field("On"), .field("Options"), .cancel, .confirm])
        XCTAssertEqual(PanelOrder.arrows(Stories["sheet.new-vm"].state), [])
    }

    func testAnAlertTrapsTheWalkInItsNameThenItsButtons() {
        XCTAssertEqual(PanelRegion.of(Stories["alert.stop-vm"].state), .alert)
        XCTAssertEqual(PanelOrder.walk(Stories["alert.stop-vm"].state), [.cancel, .confirm])
        XCTAssertEqual(PanelOrder.walk(Stories["alert.stop-server.typed"].state), [.typed, .cancel, .confirm])
    }

    func testFindWalksItsResultsAndItsShowMoreLines() {
        let find = Stories["find.keyboard"].state
        XCTAssertEqual(PanelRegion.of(find), .find)
        XCTAssertEqual(PanelOrder.walk(find), [
            .row("linux-1"), .row("linux-3"),
            .row("shared:linux-1:5173"), .row("shared:linux-1:3000"), .row("shared:linux-3:8080"), .more("ports"),
            .row("attach:linux-1"), .row("share:linux-1"), .row("share-account:linux-1"), .more("actions"),
        ])
    }

    func testTabWrapsInsideItsRegionAndTheArrowsStopAtTheEnds() {
        let walk = PanelOrder.walk(home)
        XCTAssertEqual(PanelOrder.walked(nil, by: 1, in: walk), .control("new-vm"))
        XCTAssertEqual(PanelOrder.walked(nil, by: -1, in: walk), .find)
        XCTAssertEqual(PanelOrder.walked(.find, by: 1, in: walk), .control("new-vm"))
        XCTAssertEqual(PanelOrder.walked(.control("new-vm"), by: -1, in: walk), .find)
        let arrows = PanelOrder.arrows(home)
        XCTAssertEqual(PanelOrder.moved(nil, by: 1, in: arrows), .row("waiting:review-bot"))
        XCTAssertEqual(PanelOrder.moved(.row("ending:macbook-pro/linux-1/claude"), by: 1, in: arrows), .control("menubar.ports"))
        XCTAssertEqual(PanelOrder.moved(.row("macbook-air"), by: 1, in: arrows), .row("macbook-air"))
        XCTAssertEqual(PanelOrder.moved(.row("waiting:review-bot"), by: -1, in: arrows), .row("waiting:review-bot"))
    }

    func testTheHighlightFollowsItsRowByIdentityAndGoesWithIt() {
        let linux3 = PanelFocus.row("linux-3")
        XCTAssertEqual(PanelOrder.follow(linux3, in: PanelOrder.walk(home)), linux3)
        var reordered = home
        reordered.page?.sections = home.page?.sections?.map { s in
            var s = s
            if s.id == HomeCards.agents { s.items = s.items?.reversed() }
            return s
        }
        XCTAssertEqual(PanelOrder.follow(linux3, in: PanelOrder.walk(reordered)), linux3)
        XCTAssertNil(PanelOrder.follow(linux3, in: PanelOrder.walk(mac)))
        XCTAssertNil(PanelOrder.follow(.panel, in: PanelOrder.walk(home)))
    }

    func testTheStoryDrawsTheAccountsTileLitByTheKeyboard() {
        XCTAssertEqual(Stories["home.keyboard"].ui.highlight.map(PanelFocus.init), .control("menubar.accounts"))
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
        let alert = Stories["alert.stop-vm"].state
        let typed = Stories["alert.stop-server.typed"].state
        let agent = Stories["agent.working"].state
        let workspace = Stories["workspace.agents"].state
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
            ("enter", "an agent row", home, .row("linux-1"), .open("menubar.machine:linux-1")),
            ("enter", "a need row", home, .row("waiting:review-bot"), .run("waiting:review-bot")),
            ("enter", "a tile", home, .control("menubar.accounts"), .open("menubar.accounts")),
            ("enter", "the + button", home, .control("new-vm"), .run("new-vm")),
            ("enter", "nothing lit", home, nil, .none),
            ("space", "a tile", home, .control("menubar.accounts"), .open("menubar.accounts")),
            ("cmd+enter", "an agent row", home, .row("linux-1"), .run("attach:linux-1")),
            ("cmd+enter", "an agent's page", agent, nil, .run("attach")),
            ("cmd+enter", "a Mac's page", mac, nil, .none),
            ("cmd+n", "home", home, nil, .run("new-vm")),
            ("shift+cmd+n", "a workspace's page", workspace, nil, .run("new-agent")),
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
