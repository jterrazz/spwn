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

    func audit(_ stories: [StoryFile], measured: [String: Double]) -> [String] {
        var findings: [String] = []
        for story in stories where !story.today && story.group != .shell && story.heightNote == nil {
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

enum Budget {
    static let stories: Int64 = 120_000_000
    static let rest: Int64 = 5_000_000
    static let slack: Int64 = 1_000_000

    static func bytes(_ folder: URL, skipping skipped: String? = nil) -> Int64 {
        guard let walk = FileManager.default.enumerator(at: folder, includingPropertiesForKeys: [.fileSizeKey, .isRegularFileKey]) else { return 0 }
        var total: Int64 = 0
        for case let file as URL in walk {
            if let skipped, file.lastPathComponent == skipped {
                walk.skipDescendants()
                continue
            }
            guard !file.lastPathComponent.hasSuffix(".actual.png"),
                  let values = try? file.resourceValues(forKeys: [.fileSizeKey, .isRegularFileKey]), values.isRegularFile == true else { continue }
            total += Int64(values.fileSize ?? 0)
        }
        return total
    }

    static func audit(stories: Int64, rest: Int64) -> [String] {
        var findings: [String] = []
        if stories > Self.stories { findings.append("the story goldens weigh \(stories) bytes, past their \(Self.stories)") }
        if rest > Self.rest { findings.append("the other goldens weigh \(rest) bytes, past their ratchet of \(Self.rest)") }
        if rest < Self.rest - slack { findings.append("the other goldens weigh \(rest) bytes: lower their ratchet of \(Self.rest) to what they weigh") }
        return findings
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
        let renderer = ImageRenderer(content: themed.fixedSize())
        renderer.scale = 1
        return renderer.cgImage.map { Double($0.height) }
    }

    // invariant: MB-12: every story is drawn offscreen, and drawing them
    // all opens no window.
    func testEveryStoryMatchesItsGoldensInBothSchemes() throws {
        let windows = NSApplication.shared.windows.count
        let stories = try stories()
        XCTAssertFalse(stories.isEmpty)
        var failures: [String] = []
        for story in stories {
            guard (height(story) ?? 0) > 0 else {
                failures.append("\(story.id): draws nothing")
                continue
            }
            for scheme in Self.schemes {
                let rendered = Snapshot.render(view(story), scheme: scheme, over: AnyView(FlatDesk(scheme: scheme)))
                if let f = try Golden.match(rendered, file: "\(story.id).\(scheme == .dark ? "dark" : "light")", in: Tree.storyGoldens) { failures.append(f) }
            }
        }
        XCTAssertEqual(failures, [])
        XCTAssertEqual(NSApplication.shared.windows.count, windows)
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

    // invariant: what a status item's golden cannot show, its count, is
    // what VoiceOver says of it.
    func testEveryStatusItemStorySaysItsStateInItsLabel() throws {
        let labels = try stories().filter { $0.ui.statusItem == true }.reduce(into: [String: String]()) {
            $0[$1.id] = MenuBarIcon.of(connection: $1.state.connection, page: $1.state.page).label
        }
        XCTAssertEqual(labels, [
            "menubar.count": "station, 2 things need you",
            "menubar.down": "station, infra down",
            "menubar.restarting": "infra restarting, 2 of 4 checks",
            "menubar.fine": "all machines are fine",
            "menubar.open": "station, 2 things need you",
        ])
    }

    // invariant: SPEC C4: one story of each group, tagged a11y, holds at
    // Increase Contrast in dark and at Reduce Transparency in light, each
    // proved by its golden.
    func testEveryGroupHasAStoryProvedAtIncreasedContrastAndReducedTransparency() throws {
        let stories = try stories()
        let tagged = stories.filter(\.a11y)
        XCTAssertEqual(Set(tagged.map(\.group)), Set(stories.map(\.group)), "a group without an a11y story")
        XCTAssertEqual(tagged.count, Set(tagged.map(\.group)).count, "one a11y story per group")
        var failures: [String] = []
        for story in tagged {
            let variants: [(knob: String, scheme: ColorScheme, view: AnyView)] = [
                ("ic", .dark, AnyView(StoryView(story).stationAccessibility(contrast: .increased))),
                ("rt", .light, AnyView(StoryView(story).stationAccessibility(reduceTransparency: true))),
            ]
            for v in variants {
                let rendered = Snapshot.render(v.view, scheme: v.scheme, over: AnyView(FlatDesk(scheme: v.scheme)))
                if let f = try Golden.match(rendered, file: "\(story.id).\(v.knob).\(v.scheme == .dark ? "dark" : "light")", in: Tree.storyGoldens) { failures.append(f) }
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
                if Double(caught.differing(other)) <= Double(caught.width * caught.height) * 0.001 { failures.append("\(story.id): drawn at \(freeze) as \(name)") }
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
        for story in stories where !story.file.today && panel(story) { measured[story.id] = height(story) }
        XCTAssertEqual(try Heights.load().audit(stories.map(\.file), measured: measured), [])
    }

    func testTheSizeCheckHoldsATargetStoryToItsHeightUnlessItSaysWhy() throws {
        let heights = try Heights.load()
        let page = ["pages/fleet.menubar.json"]
        let stories = [
            StoryFile(id: "home.needs", group: .home, note: "n", pages: page),
            StoryFile(id: "home.fine", group: .home, note: "n", pages: page),
            StoryFile(id: "home.empty", group: .home, note: "n", pages: page, heightNote: "the empty state keeps the footer"),
            StoryFile(id: "home.loading", group: .home, note: "n", pages: page, status: "today"),
            StoryFile(id: "menubar.count", group: .shell, note: "n", pages: page),
            StoryFile(id: "home.unknown", group: .home, note: "n", pages: page),
        ]
        let measured = ["home.needs": 692.0, "home.fine": 600, "home.empty": 200, "home.loading": 100]
        XCTAssertEqual(heights.audit(stories, measured: measured), [
            "home.fine: 600 pt tall, the prototype's 637 pt (a heightNote names a difference on purpose)",
            "home.unknown: no prototype height in heights.json",
        ])
    }

    func testTheGoldensStayWithinTheirBudget() {
        let stories = Budget.bytes(Tree.storyGoldens)
        let rest = Budget.bytes(Tree.goldens, skipping: "stories")
        print("goldens: stories \(stories) bytes of \(Budget.stories), the rest \(rest) bytes of \(Budget.rest)")
        XCTAssertEqual(Budget.audit(stories: stories, rest: rest), [])
    }

    func testTheBudgetOnlyLetsTheOldGoldensShrink() {
        XCTAssertEqual(Budget.audit(stories: Budget.stories + 1, rest: Budget.rest + 1), [
            "the story goldens weigh 120000001 bytes, past their 120000000",
            "the other goldens weigh 5000001 bytes, past their ratchet of 5000000",
        ])
        XCTAssertEqual(Budget.audit(stories: 0, rest: 3_000_000), [
            "the other goldens weigh 3000000 bytes: lower their ratchet of 5000000 to what they weigh",
        ])
    }
}
