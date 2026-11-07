import AppKit
@testable import StationKit
import SwiftUI
import XCTest

final class AtomTests: XCTestCase {
    @MainActor
    func testEveryAtomIsDrawnInBothSchemesAndNonePaintsThePlaceholder() throws {
        let failures = try AtomSheet.Family.allCases.flatMap { family in
            try [ColorScheme.dark, .light].compactMap { scheme in
                try Golden.check("kit/atoms-\(family.rawValue)", AnyView(AtomSheet(family).stationAccessibility()), scheme: scheme, desk: false)
            }
        }
        XCTAssertEqual(failures, [])
    }

    static func part(_ text: String, pri: Int? = nil, short: String? = nil) -> Value {
        var v = Value(kind: "text", text: text)
        v.pri = pri
        v.short = short
        return v
    }

    static let heroFacts = [
        part("infra healthy"), part("jterrazz.com up", short: "site up"), part("2 of 3 Macs"),
        part("3 VMs", pri: 1), part("2 agents working", pri: 2),
    ]

    static func measure(_ text: String) -> CGFloat {
        (text as NSString).size(withAttributes: [.font: NSFont.monospacedDigitSystemFont(ofSize: Token.Role.facts.size, weight: .regular)]).width
    }

    static func heroLine(panel: CGFloat) -> CGFloat {
        let m = Token.Metric.self
        return panel - m.Panel.body.leading - m.Panel.body.trailing - m.Hero.padding.leading - m.Hero.padding.trailing - m.Hero.slot.width - m.Row.gap
    }

    func fitted(_ panel: CGFloat) -> String {
        FactsLine.fit(Self.heroFacts, width: Self.heroLine(panel: panel), measure: Self.measure).replacingOccurrences(of: Token.Glyph.sep, with: " / ")
    }

    func testAWidePanelDropsOnlyTheLeastNeededFact() {
        XCTAssertEqual(fitted(380), "infra healthy / jterrazz.com up / 2 of 3 Macs / 3 VMs")
    }

    func testTheDefaultPanelDropsEveryFactThatMayGoBeforeShorteningOne() {
        XCTAssertEqual(fitted(340), "infra healthy / jterrazz.com up / 2 of 3 Macs")
    }

    func testANarrowPanelRewritesAFactShorterRatherThanCutIt() {
        XCTAssertEqual(fitted(320), "infra healthy / site up / 2 of 3 Macs")
    }

    func testALineThatFitsKeepsEveryFact() {
        XCTAssertEqual(FactsLine.fit(Self.heroFacts, width: .infinity, measure: Self.measure).components(separatedBy: Token.Glyph.sep).count, 5)
    }

    func testAFactWithAnEmptyShortWordingLeavesTheLineWhenNothingElseCanGo() {
        let facts = [Self.part("Mac mini"), Self.part("asleep since 09:12", short: "")]
        XCTAssertEqual(FactsLine.fit(facts, width: 1, measure: Self.measure), "Mac mini")
    }

    func testAMacIsDrawnByItsLidNeverByItsModel() {
        func glyph(_ kind: String, ring: String? = nil) -> Glyph {
            var g = Glyph()
            g.kind = kind
            g.ring = ring
            return g
        }
        XCTAssertEqual(GlyphView.drawn(glyph("laptop")), .symbol(Token.Object.mac(lid: true)))
        XCTAssertEqual(GlyphView.drawn(glyph("desktop")), .symbol(Token.Object.mac(lid: false)))
        XCTAssertEqual(GlyphView.drawn(glyph("desktop", ring: "cpu:mac-mini")), .mac(lid: false))
        XCTAssertNotEqual(Token.Object.mac(lid: true), Token.Object.mac(lid: false))
    }

    func testAServiceWearsItsMarkAndAnUnknownOneTheQuietChip() {
        var g = Glyph()
        g.kind = "port:Vite dev server"
        XCTAssertEqual(GlyphView.drawn(g), .chip(.port, "Vite dev server"))
        XCTAssertEqual(Token.Object.service("Vite dev server", among: Token.Object.ports), "vite")
        XCTAssertNil(Token.Object.service("gitlab", among: Token.Object.accounts))
        let tokens = Token.resolved(scheme: .dark)
        XCTAssertEqual(tokens.chip(Token.Object.service("gitlab", among: Token.Object.accounts)).fill, [tokens.color.q])
    }

    func testALiveValueSaysTheLatestSampleInThePagesUnit() {
        var cpu = Value(kind: "percent")
        cpu.n = 120
        cpu.live = "cpu:linux-1"
        XCTAssertEqual(Format.show(Live.value(cpu, samples: ["cpu:linux-1": 34])), "34\u{00A0}%")
        XCTAssertEqual(Format.show(Live.value(cpu, samples: [:])), Format.show(cpu))
    }

    func testATimeLeftIsSaidInHoursAndMinutes() {
        XCTAssertEqual(Format.left(13920), "3\u{00A0}hr 52\u{00A0}min left")
        XCTAssertEqual(Format.left(1380), "23\u{00A0}min left")
        XCTAssertEqual(Format.left(7200), "2\u{00A0}hr left")
        XCTAssertEqual(Format.left(20), "<1\u{00A0}min left")
    }

    func testAKeyCapSaysItsShortcutAsMacOSDoes() {
        XCTAssertEqual(Token.Key.cap("cmd+k"), "⌘K")
        XCTAssertEqual(Token.Key.cap("cmd+enter"), "⌘↩")
        XCTAssertEqual(Token.Key.cap("opt+enter"), "⌥↩")
        XCTAssertEqual(Token.Key.cap("shift+cmd+p"), "⇧⌘P")
    }
}
