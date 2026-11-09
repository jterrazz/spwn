import CoreImage
@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class PopoverTests: XCTestCase {
    func page(_ story: String) -> Page { Stories[story].state.shown ?? Page() }

    func row(_ id: String, of page: Page) throws -> Row {
        try XCTUnwrap((page.sections ?? []).flatMap(\.rows).first { $0.id == id }, "\(id) is not a row of the page")
    }

    func testTheQRScansBackToTheLinkItShows() throws {
        for link in ["http://macbook-pro.tail77a797.ts.net:40001", "https://jterrazz.com"] {
            let modules = try XCTUnwrap(QR.modules(link))
            let big = CIImage(cgImage: modules).samplingNearest().transformed(by: CGAffineTransform(scaleX: 12, y: 12))
            let detector = try XCTUnwrap(CIDetector(ofType: CIDetectorTypeQRCode, context: nil, options: [CIDetectorAccuracy: CIDetectorAccuracyHigh]))
            let read = detector.features(in: big).compactMap { ($0 as? CIQRCodeFeature)?.messageString }
            XCTAssertEqual(read, [link])
        }
    }

    func testAPrivateLinkOpensOnTheTailnetAndAPublicOneForAnyoneThroughItsRoute() throws {
        let workspace = page("pop.qr")
        let vite = try row("linux-1:5173", of: workspace)
        XCTAssertEqual(QRPopover.reach(vite).link, "linux-1:5173")
        XCTAssertEqual(QRPopover.reach(vite).line, "Opens on any device of your tailnet")
        let verbs = QRPopover.verbs(of: vite, in: workspace.actions ?? [])
        XCTAssertEqual(verbs.copy?.run?.copy, "http://macbook-pro.tail77a797.ts.net:40001")
        XCTAssertEqual(verbs.open?.run?.open, "http://macbook-pro.tail77a797.ts.net:40001")
        XCTAssertEqual(verbs.stop?.label, "Stop sharing")

        let server = page("pop.qr.public")
        let traefik = try row("infra:443", of: server)
        XCTAssertEqual(QRPopover.reach(traefik).link, "jterrazz.com")
        XCTAssertEqual(QRPopover.reach(traefik).line, "Anyone can open it · Cloudflare Tunnel")
        XCTAssertTrue(QRPopover.reach(traefik).public)
        XCTAssertNil(QRPopover.verbs(of: traefik, in: server.actions ?? []).stop, "a public route is declared, never stopped from its QR")
    }

    func testThePopoverOfARowTheStoryShowsIsTheQROfItsLink() throws {
        let workspace = page("pop.qr")
        let vite = try row("linux-1:5173", of: workspace)
        XCTAssertEqual(PortsSection.unfolds(vite, shown: Stories["pop.qr"].state.qr, in: workspace.actions ?? [])?.absoluteString,
                       "http://macbook-pro.tail77a797.ts.net:40001")
        XCTAssertEqual(Stories["pop.info"].state.about, "private")
        XCTAssertNil(Stories["pop.info"].state.qr)
    }

    func testAPopoverSitsUnderItsButtonWhenItFitsElseOverItAndStaysInsideThePanel() {
        let panel = CGSize(width: 340, height: 640)
        let card = CGSize(width: 304, height: 160)
        let under = PopoverLayout.spot(card, of: CGRect(x: 280, y: 100, width: 20, height: 20), in: panel, side: .below)
        XCTAssertTrue(under.below)
        XCTAssertEqual(under.card.minY, 120 + Token.Metric.Popover.arrowDepth)
        XCTAssertEqual(under.card.maxX, panel.width - Token.Metric.Popover.margin, "kept inside the panel's right edge")
        let over = PopoverLayout.spot(card, of: CGRect(x: 20, y: 560, width: 300, height: 40), in: panel, side: .below)
        XCTAssertFalse(over.below)
        XCTAssertEqual(over.card.maxY, 560 - Token.Metric.Popover.arrowDepth)
        XCTAssertEqual(over.arrow?.maxY, 560)
        let qr = PopoverLayout.spot(card, of: CGRect(x: 20, y: 400, width: 300, height: 40), in: panel, side: .above)
        XCTAssertFalse(qr.below, "a QR sits over its row when it fits there")
        XCTAssertTrue(PopoverLayout.spot(card, of: CGRect(x: 20, y: 60, width: 300, height: 40), in: panel, side: .above).below, "and under it when it does not")
        let tip = PopoverLayout.spot(CGSize(width: 90, height: 20), of: CGRect(x: 20, y: 200, width: 100, height: 56), in: panel, side: .tooltip)
        XCTAssertEqual(tip.card.minY, 256 + Token.Metric.Tooltip.gap)
        XCTAssertNil(tip.arrow)
    }

    func testATooltipSaysThePagesWordsForTheControlItExplains() throws {
        let home = page("pop.tooltip")
        let ports = try XCTUnwrap((home.sections ?? []).flatMap { $0.items ?? [] }.compactMap(\.tile).first { $0.nav == Stories["pop.tooltip"].ui.tooltip })
        XCTAssertEqual(TileView.tooltip(ports), "5 shared · ⌘P")
        let awake = try XCTUnwrap((home.sections ?? []).flatMap { $0.items ?? [] }.compactMap(\.tile).first { $0.caption == "Awake" })
        XCTAssertEqual(TileView.tooltip(awake), "Awake until 18:00 · ⌘E · right-click for a duration")
    }

    func testEveryHUDStoryDrawsTheWordsItsPageGivesTheGesture() throws {
        let want = ["pop.hud.copy": ("Copied ssh linux-1", Token.Symbol.hudCopy), "pop.hud.terminal": ("linux-1 opened in Terminal", Token.Symbol.hudTerminal)]
        for (id, (words, symbol)) in want {
            let story = Stories[id]
            let action = try XCTUnwrap(story.state.shown?.action(story.ui.hud ?? ""), "\(id) names no action of its page")
            let hud = try XCTUnwrap(story.hud)
            XCTAssertEqual(hud.words, action.hud, "\(id): the words are the page's")
            XCTAssertEqual(hud.text, words, id)
            XCTAssertEqual(hud.symbol, symbol, id)
        }
    }

    func testEveryHUDThePagesCarryIsSaidAsThePageSaysIt() throws {
        let stories = try Stories.load(root: Tree.stories)
        let notices = HUDNotice.all(in: stories.flatMap(\.pages))
        XCTAssertGreaterThan(notices.count, 20)
        for page in stories.flatMap(\.pages) {
            for action in page.actions ?? [] {
                guard let hud = action.hud else { continue }
                let notice = try XCTUnwrap(HUDNotice(action))
                XCTAssertEqual(notice.text, Format.show(hud), action.id ?? "")
                XCTAssertFalse(notice.text.isEmpty, "\(action.id ?? ""): a HUD with no words")
            }
        }
    }

    func testACopyOrALinkActsOnTheMacAndSaysSoInTheHUD() {
        var copied: [String] = [], opened: [URL] = [], shown: [String] = [], ran: [String] = []
        var h = PopoverHandlers(run: { a, _ in ran.append(a.id ?? "") }, open: { opened.append($0) }, copy: { copied.append($0) })
        h.hud = { shown.append($0.text) }
        let workspace = page("pop.hud.copy")
        for id in ["copy-ssh", "open-port:linux-1:5173", "open-station"] {
            if let action = workspace.action(id) { StationKit.perform(action, h) }
        }
        XCTAssertEqual(copied, ["ssh linux-1"])
        XCTAssertEqual(opened.map(\.absoluteString), ["http://macbook-pro.tail77a797.ts.net:40001"])
        XCTAssertEqual(ran, ["open-station"], "a terminal runs, its HUD said once it opened")
        XCTAssertEqual(shown, ["Copied ssh linux-1"])
    }
}
