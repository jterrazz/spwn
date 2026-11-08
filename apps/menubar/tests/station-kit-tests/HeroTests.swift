import StationKit
import SwiftUI
import XCTest

@MainActor
final class HeroTests: XCTestCase {
    static let states = [
        "home.needs", "home.one-need.wait", "home.one-need.lease", "home.many-needs", "home.down", "home.restarting",
        "home.exposed", "home.down-and-exposed", "home.loading", "home.fine", "home.empty",
    ]

    func stories() throws -> [Story] { try Stories.load(root: Tree.stories).filter { Self.states.contains($0.id) } }

    // invariant: every state of the hero reads at Increase Contrast in dark
    // and at Reduce Transparency in light, each proved by its golden.
    func testEveryHeroStateHoldsAtIncreasedContrastAndReducedTransparency() throws {
        let stories = try stories()
        XCTAssertEqual(Set(stories.map(\.id)), Set(Self.states))
        var failures: [String] = []
        for story in stories {
            let variants: [(knob: String, scheme: ColorScheme, view: AnyView)] = [
                ("ic", .dark, AnyView(StoryView(story).stationAccessibility(contrast: .increased))),
                ("rt", .light, AnyView(StoryView(story).stationAccessibility(reduceTransparency: true))),
            ]
            for v in variants {
                let rendered = Snapshot.render(v.view, scheme: v.scheme, over: AnyView(FlatDesk(scheme: v.scheme)))
                let file = "\(story.id).\(v.knob).\(v.scheme == .dark ? "dark" : "light")"
                if let f = try Golden.match(rendered, file: file, in: Tree.storyGoldens) { failures.append(f) }
            }
        }
        XCTAssertEqual(failures, [])
    }

    func hero(_ id: String) throws -> StationKit.Section? {
        let story = try XCTUnwrap(try stories().first { $0.id == id })
        return story.pages.first?.section("hero")
    }

    // invariant: VoiceOver hears the hero once each time it changes, and
    // never on a tick: the words it is told hold only what the hero says,
    // which a sample does not move.
    func testVoiceOverHearsTheHeroOncePerChangeNeverOnATick() throws {
        let needs = try hero("home.needs"), down = try hero("home.down"), restarting = try hero("home.restarting"), fine = try hero("home.fine")
        XCTAssertEqual(Hero.announcement(needs), "2 things need you")
        XCTAssertEqual(Hero.announcement(down), "infra is down\(Token.Glyph.sep)out of memory\(Token.Glyph.sep)since 10:26\(Token.Glyph.sep)on mac-mini")
        XCTAssertEqual(Hero.announcement(restarting), "Restarting infra\(Token.Glyph.ellipsis)\(Token.Glyph.sep)on mac-mini\(Token.Glyph.sep)has an address\(Token.Glyph.sep)2 of 4")

        var ticked = try XCTUnwrap(try stories().first { $0.id == "home.needs" }?.pages.first)
        ticked.strip?.cpu?.n = 999
        ticked.badge?.needs = 2
        let heard = [needs, ticked.section("hero"), needs, down, down, restarting, fine].map(Hero.announcement)
        var said: [String] = []
        var last: String?
        for words in heard where words != last {
            if let words { said.append(words) }
            last = words
        }
        XCTAssertEqual(said.count, 4, "\(said)")
    }
}
