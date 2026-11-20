import AppKit
import Foundation
import StationKit
import SwiftUI
import XCTest

struct FlatDesk: View {
    let scheme: ColorScheme

    var body: some View {
        scheme == .dark ? Color(red: 0x1B / 255, green: 0x2F / 255, blue: 0x55 / 255) : Color(red: 0x7F / 255, green: 0xB6 / 255, blue: 0xF2 / 255)
    }
}

struct Heights: Decodable {
    let heights: [String: Double]

    static let tolerance = 4.0

    static func load() throws -> Heights {
        try JSONDecoder().decode(Heights.self, from: Data(contentsOf: Tree.storyGoldens.appendingPathComponent("heights.json")))
    }

    // value: under STATION_UPDATE_GOLDENS=1 the file is rewritten from the heights the
    // station-kit render measures, one entry per story in the story list's order.
    static func document(_ stories: [StoryFile], measured: [String: Double]) -> String {
        let rows = stories.map { "  \"\($0.id)\": \(Int((measured[$0.id] ?? 0).rounded()))" }.joined(separator: ",\n")
        return "{\n \"schema\": \"station.story-heights/v1\",\n \"source\": \"the station-kit render at its commit\",\n \"unit\": \"pt\",\n \"heights\": {\n\(rows)\n }\n}\n"
    }

    func audit(_ stories: [StoryFile], measured: [String: Double]) -> [String] {
        var findings: [String] = []
        for story in stories where story.group != .shell && story.heightNote == nil {
            guard let want = heights[story.id] else {
                findings.append("\(story.id): no prototype height in heights.json")
                continue
            }
            guard let have = measured[story.id] else {
                findings.append("\(story.id): no height measured")
                continue
            }
            if abs(have - want) > Self.tolerance {
                findings.append("\(story.id): \(Int(have.rounded())) pt tall, the prototype's \(Int(want)) pt (a heightNote names a difference on purpose)")
            }
        }
        return findings
    }
}

// invariant: a story keeps pixel goldens only for a paint recipe no view
// golden states and no sheet of the kit draws: what each kind of overlay
// composites over its page, and the panel the live capture measures its
// contrast against. Every other story is proved by its view goldens.
enum Witness {
    static let recipes: [String: String] = [
        "pop.info": "a popover's glass and shadow over its page",
        "menu.context.stopped": "a menu's glass, shadow and lit row over its page",
        "alert.delete-vm.persistent": "an alert's card over its dimmed page",
        "sheet.share-port.empty": "a sheet's card over its dimmed page",
        "find.port": "Find's field and results over the panel",
        "pop.hud.copy": "a HUD's glass over its page",
        "home.needs": "the panel specs/machine/menubar/capture/measure.py reads the live capture's contrast against",
    ]

    static func kinds(_ ui: StoryUI) -> Set<String> {
        let opened: [(String, Bool)] = [
            ("a popover", ui.popover != nil), ("a menu", ui.menu != nil), ("an alert", ui.alert != nil),
            ("a sheet", ui.sheet != nil), ("Find", ui.find != nil), ("a HUD", ui.hud != nil),
        ]
        return Set(opened.filter(\.1).map(\.0))
    }

    // value: the overlay kinds the stories open that no witness opens.
    static func unproved(_ stories: [(id: String, ui: StoryUI)], recipes: [String: String] = recipes) -> [String] {
        let opened = stories.reduce(into: Set<String>()) { $0.formUnion(kinds($1.ui)) }
        let proved = stories.filter { recipes[$0.id] != nil }.reduce(into: Set<String>()) { $0.formUnion(kinds($1.ui)) }
        return opened.subtracting(proved).sorted().map { "\($0) over a page: no witness story keeps its pixels" }
    }
}

@MainActor
final class StoryTests: XCTestCase {
    static let schemes: [ColorScheme] = [.dark, .light]

    func stories() throws -> [Story] { try Stories.load(root: Tree.stories) }

    func view(_ story: Story) -> AnyView { AnyView(StoryView(story).stationAccessibility()) }

    func panel(_ story: Story) -> Bool { story.group != .shell && story.ui.statusItem != true }

    func height(_ story: Story) -> Double? {
        let themed = view(story).stationTheme(.dark).environment(\.stationNow, Snapshot.now).environment(\.stationStill, true)
        return Snapshot.drawn(themed.fixedSize(), scale: 1).map { Double($0.height) }
    }

    static let goldenShards = 4

    static func shard(_ stories: [Story], _ index: Int) -> [Story] {
        stories.enumerated().filter { $0.offset % goldenShards == index }.map(\.element)
    }

    func matchGoldens(shard index: Int) throws {
        let windows = NSApplication.shared.windows.count
        let all = try stories()
        let stories = Self.shard(all, index)
        XCTAssertFalse(stories.isEmpty)
        var failures: [String] = []
        for story in stories {
            guard (height(story) ?? 0) > 0 else {
                failures.append("\(story.id): draws nothing")
                continue
            }
            let under = Self.base(of: story, among: all)
            for scheme in Self.schemes {
                if Witness.recipes[story.id] != nil {
                    let rendered = Snapshot.render(view(story), scheme: scheme, over: AnyView(FlatDesk(scheme: scheme)))
                    if let f = try Golden.match(rendered, file: "\(story.id).\(scheme == .dark ? "dark" : "light")", in: Tree.storyGoldens, desk: 2) { failures.append(f) }
                }
                if let f = try ViewGolden.check(story.id, view(story), scheme: scheme, ground: "FlatDesk", in: Tree.storyGoldens,
                                                over: under.map { ($0.id, view($0)) }) { failures.append(f) }
            }
        }
        XCTAssertEqual(failures, [])
        XCTAssertEqual(NSApplication.shared.windows.count, windows)
    }

    // invariant: MB-12: every story is drawn offscreen, and drawing them
    // all opens no window. The stories are dealt to the shards by index, so
    // each runs in its own process under `swift test --parallel`.
    func testStoryGoldensShard0() throws { try matchGoldens(shard: 0) }
    func testStoryGoldensShard1() throws { try matchGoldens(shard: 1) }
    func testStoryGoldensShard2() throws { try matchGoldens(shard: 2) }
    func testStoryGoldensShard3() throws { try matchGoldens(shard: 3) }

    // value: what an overlay story opens over a page, the keys a base
    // story does not have.
    static func bare(_ ui: StoryUI) -> StoryUI {
        var ui = ui
        ui.popover = nil
        ui.menu = nil
        ui.alert = nil
        ui.sheet = nil
        ui.find = nil
        ui.hud = nil
        return ui
    }

    // invariant: an overlay story is drawn over its base: the first story,
    // outside the transitions, of its pages, its samples and its UI but
    // what it opens over them; nil for a story that opens nothing, or
    // whose page no story draws bare.
    static func base(of story: Story, among stories: [Story]) -> Story? {
        let ui = bare(story.ui)
        guard ui != story.ui else { return nil }
        return stories.first {
            $0.id != story.id && $0.group != .transition && $0.file.pages == story.file.pages && $0.file.samples == story.file.samples && $0.ui == ui
        }
    }

    func testEveryKindOfOverlayKeepsTheWitnessOfItsPixels() throws {
        let all = try stories()
        XCTAssertEqual(Set(Witness.recipes.keys).subtracting(all.map(\.id)), [], "a witness names no story")
        XCTAssertEqual(Witness.unproved(all.map { ($0.id, $0.ui) }), [])
        var recipes = Witness.recipes
        recipes["find.port"] = nil
        XCTAssertEqual(Witness.unproved(all.map { ($0.id, $0.ui) }, recipes: recipes), ["Find over a page: no witness story keeps its pixels"])
    }

    func testAnOverlayStoryIsDrawnOverTheStoryOfItsPageAndNoOther() throws {
        let all = try stories()
        let bases = all.reduce(into: [String: String]()) { $0[$1.id] = Self.base(of: $1, among: all)?.id }
        XCTAssertEqual(bases["pop.info"], "ports.all")
        XCTAssertEqual(bases["menu.context.stopped"], "mac.this.overview")
        XCTAssertEqual(bases["alert.restart-mac"], "mac.mini.overview")
        XCTAssertNil(bases["home.needs"])
        XCTAssertNil(bases["find.port"])
        for (id, base) in bases { XCTAssertNil(bases[base], "\(id) is drawn over \(base), itself an overlay") }
    }

    // invariant: the shards are exactly the story list, each story once, and
    // there is one method per shard.
    func testTheShardsAreExactlyTheStoryList() throws {
        let all = try stories()
        let dealt = (0..<Self.goldenShards).flatMap { Self.shard(all, $0).map(\.id) }
        XCTAssertEqual(dealt.count, all.count)
        XCTAssertEqual(Set(dealt), Set(all.map(\.id)))
        let methods = Self.defaultTestSuite.tests.map(\.name).filter { $0.contains("testStoryGoldensShard") }.sorted()
        XCTAssertEqual(methods.count, Self.goldenShards)
        for index in 0..<Self.goldenShards {
            XCTAssertTrue(methods[index].hasSuffix("testStoryGoldensShard\(index)]"), methods[index])
        }
    }

    func testEveryPanelStoryStaysGlass() throws {
        var failures: [String] = []
        for story in try stories() where panel(story) && story.group != .sheet && story.group != .alert && story.ui.sheet == nil
            && StoryMotion.of(story.id)?.motion.slides != true {
            for scheme in Self.schemes {
                let share = GlassTests.opaque(view(story), scheme: scheme) ?? 1
                if share >= 0.4 { failures.append("\(story.id) \(scheme): \(Int(share * 100)) % of it is opaque") }
            }
        }
        XCTAssertEqual(failures, [])
    }

    // invariant: what a status item's golden cannot show, its words, is
    // what VoiceOver says of it.
    func testEveryStatusItemStorySaysItsStateInItsLabel() throws {
        let labels = try stories().filter { $0.ui.statusItem == true }.reduce(into: [String: String]()) {
            $0[$1.id] = MenuBarIcon.of(connection: $1.state.connection, page: $1.state.page).label
        }
        XCTAssertEqual(labels, [
            "menubar.down": "Spwn Menu, infra down",
            "menubar.restarting": "infra restarting, 2 of 4 checks",
            "menubar.fine": "all machines are fine",
            "menubar.open": "all machines are fine",
            "menubar.stopped": "Spwn Menu: engine stopped",
            "menubar.connecting": "Spwn Menu: connecting",
        ])
    }

    // invariant: SPEC C4: one story of each group, tagged a11y, holds at
    // Increase Contrast in dark and at Reduce Transparency in light, each
    // proved by its view golden over the plain story's; the kit's panel
    // sheets keep the pixels of both settings. A status item is drawn in
    // the bar's one ink whatever the settings, so it also draws its plain
    // story's pixels.
    func testEveryGroupHasAStoryProvedAtIncreasedContrastAndReducedTransparency() throws {
        let stories = try stories()
        let tagged = stories.filter(\.a11y)
        XCTAssertEqual(Set(tagged.map(\.group)), Set(stories.map(\.group)), "a group without an a11y story")
        XCTAssertEqual(tagged.count, Set(tagged.map(\.group)).count, "one a11y story per group")
        var failures: [String] = []
        for story in tagged {
            let variants: [(knob: String, scheme: ColorScheme, contrast: ColorSchemeContrast, reduceTransparency: Bool)] = [
                ("ic", .dark, .increased, false), ("rt", .light, .standard, true),
            ]
            for v in variants {
                let drawn = AnyView(StoryView(story).stationAccessibility(contrast: v.contrast, reduceTransparency: v.reduceTransparency))
                if story.ui.statusItem == true {
                    let file = "\(story.id).\(v.knob).\(v.scheme == .dark ? "dark" : "light")"
                    let rendered = Snapshot.render(drawn, scheme: v.scheme, over: AnyView(FlatDesk(scheme: v.scheme)))
                    let plain = Snapshot.render(view(story), scheme: v.scheme, over: AnyView(FlatDesk(scheme: v.scheme)))
                    if rendered?.bitmap.pixels != plain?.bitmap.pixels { failures.append("\(file): a status item draws otherwise than its plain story") }
                }
                if let f = try ViewGolden.check("\(story.id).\(v.knob)", drawn, scheme: v.scheme, ground: "FlatDesk", contrast: v.contrast,
                                                reduceTransparency: v.reduceTransparency, in: Tree.storyGoldens, over: (story.id, view(story))) { failures.append(f) }
            }
        }
        XCTAssertEqual(failures, [])
    }

    // invariant: SPEC A8: a transition story is caught mid-flight by its
    // modifiers: drawn at its freeze it is neither the page it leaves nor
    // the page it reaches, so a movement no modifier draws is caught.
    func testEveryTransitionIsCaughtMidFlight() throws {
        var failures: [String] = []
        for story in try stories() where story.group == .transition {
            guard let from = story.from, let freeze = story.freeze else {
                failures.append("\(story.id): no play.from or no freeze")
                continue
            }
            func drawn(_ view: StoryView) -> Bitmap? { Snapshot.render(AnyView(view.stationAccessibility()), scheme: .dark, over: AnyView(FlatDesk(scheme: .dark)))?.bitmap }
            guard let caught = drawn(StoryView(story, time: freeze)), let end = drawn(StoryView(story, rest: true)), let start = drawn(StoryView(from)) else {
                failures.append("\(story.id): draws nothing")
                continue
            }
            for (name, other) in [("its end", end), ("where it starts", start)] where other.width == caught.width && other.height == caught.height {
                if Double(caught.differing(other, tolerance: 16)) <= Double(caught.width * caught.height) * 0.001 { failures.append("\(story.id): drawn at \(freeze) as \(name)") }
            }
        }
        XCTAssertEqual(failures, [])
    }

    // invariant: SPEC C3: the Catalog's scrubber and its Play draw a
    // transition through the moments its golden freezes: at the story's
    // freeze the replay is the story's own drawing.
    func testTheCatalogsReplayAtAStorysFreezeIsTheStoryItself() throws {
        let knobs = StoryKnobs(scale: 1)
        var failures: [String] = []
        for story in try stories() where story.group == .transition {
            let still = try XCTUnwrap(StoryStage.png(story, knobs: knobs).flatMap(Bitmap.init(png:)))
            let renderer = ImageRenderer(content: StoryReplay(story, knobs: knobs, time: story.freeze ?? 1))
            renderer.scale = knobs.scale
            let replayed = try XCTUnwrap(renderer.cgImage.flatMap { NSBitmapImageRep(cgImage: $0).representation(using: .png, properties: [:]) }.flatMap(Bitmap.init(png:)))
            if replayed.width != still.width || replayed.height != still.height || replayed.differing(still) > 0 { failures.append("\(story.id): replayed at its freeze, it is not its still") }
        }
        XCTAssertEqual(failures, [])
    }

    func testEveryTargetStoryIsAsTallAsThePrototypesPanel() throws {
        let stories = try stories()
        var measured: [String: Double] = [:]
        for story in stories where panel(story) { measured[story.id] = height(story) }
        if Golden.update {
            try Heights.document(stories.map(\.file), measured: measured)
                .write(to: Tree.storyGoldens.appendingPathComponent("heights.json"), atomically: true, encoding: .utf8)
        }
        XCTAssertEqual(try Heights.load().audit(stories.map(\.file), measured: measured), [])
    }

    func testTheHeightsFileIsWrittenFromTheMeasuredHeights() throws {
        let page = ["pages/fleet.menubar.json"]
        let stories = [StoryFile(id: "home.needs", group: .home, note: "n", pages: page), StoryFile(id: "menubar.down", group: .shell, note: "n", pages: page)]
        let text = Heights.document(stories, measured: ["home.needs": 641.6])
        XCTAssertEqual(text, "{\n \"schema\": \"station.story-heights/v1\",\n \"source\": \"the station-kit render at its commit\",\n \"unit\": \"pt\",\n \"heights\": {\n  \"home.needs\": 642,\n  \"menubar.down\": 0\n }\n}\n")
        XCTAssertEqual(try JSONDecoder().decode(Heights.self, from: Data(text.utf8)).heights, ["home.needs": 642, "menubar.down": 0])
    }

    func testTheSizeCheckHoldsATargetStoryToItsHeightUnlessItSaysWhy() {
        let heights = Heights(heights: ["home.needs": 455, "home.fine": 455, "home.empty": 360])
        let page = ["pages/fleet.menubar.json"]
        let stories = [
            StoryFile(id: "home.needs", group: .home, note: "n", pages: page),
            StoryFile(id: "home.fine", group: .home, note: "n", pages: page),
            StoryFile(id: "home.empty", group: .home, note: "n", pages: page, heightNote: "the empty state keeps the footer"),
            StoryFile(id: "menubar.down", group: .shell, note: "n", pages: page),
            StoryFile(id: "home.unknown", group: .home, note: "n", pages: page),
        ]
        let measured = ["home.needs": 455.0, "home.fine": 600, "home.empty": 200]
        XCTAssertEqual(heights.audit(stories, measured: measured), [
            "home.fine: 600 pt tall, the prototype's 455 pt (a heightNote names a difference on purpose)",
            "home.unknown: no prototype height in heights.json",
        ])
    }
}
