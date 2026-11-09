import Foundation
@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class StoryLoadTests: XCTestCase {
    var root: URL!

    override func setUpWithError() throws {
        root = FileManager.default.temporaryDirectory.appendingPathComponent("station-stories-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: root.appendingPathComponent("pages"), withIntermediateDirectories: true)
        for name in ["menubar.home.two-macs", "menubar.mac.mini"] {
            try FileManager.default.copyItem(at: Tree.pages.appendingPathComponent(name + ".json"),
                                             to: root.appendingPathComponent("pages/\(name).json"))
        }
    }

    override func tearDownWithError() throws {
        try FileManager.default.removeItem(at: root)
    }

    func write(_ story: StoryFile, as id: String? = nil) throws {
        try story.encoded().write(to: root.appendingPathComponent((id ?? story.id) + Stories.suffix))
    }

    func testAStoryFileRoundTripsWithEveryPartOfItsState() throws {
        var ui = StoryUI()
        ui.segment = "activity"
        ui.unfolded = ["upkeep:mac-mini"]
        ui.hover = "row:linux-1"
        ui.sheet = SheetState(id: "stop", answers: ["name": "infra"])
        ui.alert = "stop"
        ui.menu = MenuOpen(row: "linux-1")
        ui.find = FindState(query: "5173", highlight: 1)
        ui.pending = ["restart:infra"]
        ui.statusItem = false
        let story = StoryFile(id: "transition.push", group: .transition, note: "The machine page slides in over Home.",
                              pages: ["pages/menubar.home.two-macs.json", "pages/menubar.mac.mini.json"], ui: ui,
                              samples: ["cpu:mac-mini": 46], play: StoryPlay(from: "home.needs", gesture: "push(mac:mac-mini)"),
                              freeze: 0.35, heightNote: "the machine page is taller than the prototype's")
        try write(story)
        let loaded = try XCTUnwrap(Stories.load(root: root).first)
        XCTAssertEqual(loaded.file, story)
        XCTAssertEqual(loaded.pages.map(\.id), [PageID.home, PageID.machine])
        XCTAssertEqual(try JSONDecoder().decode(StoryFile.self, from: story.encoded()), story)
        XCTAssertTrue(String(decoding: try story.encoded(), as: UTF8.self).contains("\"schema\" : \"station.story/v1\""))
    }

    func testTheLoaderRefusesAStoryThatDoesNotSayWhatItIs() throws {
        let page = ["pages/menubar.home.two-macs.json"]
        var wrongSchema = StoryFile(id: "home.needs", group: .home, note: "n", pages: page)
        wrongSchema.schema = "station.story/v0"
        let cases: [(StoryFile, String?, StoryError)] = [
            (wrongSchema, nil, .schema(file: "home.needs.story.json", schema: "station.story/v0")),
            (StoryFile(id: "home.fine", group: .home, note: "n", pages: page), "home.needs", .id(file: "home.needs.story.json", id: "home.fine")),
            (StoryFile(id: "menubar.count", group: .home, note: "n", pages: page), nil, .group(id: "menubar.count", group: .home)),
            (StoryFile(id: "home.needs", group: .home, note: "n", pages: []), nil, .noPages(id: "home.needs")),
            (StoryFile(id: "home.needs", group: .home, note: "n", pages: ["pages/fleet.menubar.json"]), nil,
             .missingPage(id: "home.needs", page: "pages/fleet.menubar.json")),
        ]
        for (story, name, want) in cases {
            try write(story, as: name)
            XCTAssertThrowsError(try Stories.load(root: root)) { XCTAssertEqual($0 as? StoryError, want) }
            for file in try Stories.files(root: root) { try FileManager.default.removeItem(at: file) }
        }
        try Data("{\"schema\":\"station.page/v0\"}".utf8).write(to: root.appendingPathComponent("pages/bad.json"))
        try write(StoryFile(id: "home.needs", group: .home, note: "n", pages: ["pages/bad.json"]))
        XCTAssertThrowsError(try Stories.load(root: root)) {
            guard case .undecodable(id: "home.needs", page: "pages/bad.json", reason: _) = $0 as? StoryError else { return XCTFail("\($0)") }
        }
    }

    func testEveryStoryIdBelongsToItsGroup() {
        XCTAssertEqual(StoryGroup.of("menubar.count"), .shell)
        XCTAssertEqual(StoryGroup.of("mac.mini.activity.6m"), .mac)
        XCTAssertEqual(StoryGroup.of("pop.hud.copy"), .pop)
        XCTAssertNil(StoryGroup.of("phone.explainer"))
    }

    func testAStoryDrawsTodaysPanelFromItsState() throws {
        try write(StoryFile(id: "mac.mini.overview", group: .mac, note: "n",
                            pages: ["pages/menubar.home.two-macs.json", "pages/menubar.mac.mini.json"]))
        var find = StoryUI()
        find.find = FindState(query: "5173")
        try write(StoryFile(id: "home.needs", group: .home, note: "n", pages: ["pages/menubar.home.two-macs.json"], ui: find))
        var item = StoryUI()
        item.statusItem = true
        try write(StoryFile(id: "menubar.count", group: .shell, note: "n", pages: ["pages/menubar.home.two-macs.json"], ui: item))
        let stories = Dictionary(uniqueKeysWithValues: try Stories.load(root: root).map { ($0.id, $0) })

        let mac = try XCTUnwrap(stories["mac.mini.overview"]).state
        XCTAssertEqual(mac.page?.id, PageID.home)
        XCTAssertEqual(mac.pushed?.id, PageID.machine)
        XCTAssertEqual(try XCTUnwrap(stories["home.needs"]).state.find, FindQuery("5173"))

        let home = try XCTUnwrap(stories["home.needs"])
        let today = try XCTUnwrap(Snapshot.render(Snapshot.view("home", try Tree.page("menubar.home.two-macs")), scheme: .dark)).bitmap
        let unfiltered = Story(file: StoryFile(id: "home.needs", group: .home, note: "n", pages: home.file.pages), pages: home.pages)
        let drawn = try XCTUnwrap(Snapshot.render(AnyView(StoryView(unfiltered)), scheme: .dark)).bitmap
        XCTAssertEqual(drawn.width, today.width)
        XCTAssertEqual(drawn.height, today.height)
        XCTAssertEqual(drawn.differing(today), 0, "a story draws the panel the app draws from the same state")

        let icon = try XCTUnwrap(Snapshot.render(AnyView(StoryView(try XCTUnwrap(stories["menubar.count"]))), scheme: .dark)).bitmap
        XCTAssertLessThan(icon.height, 200, "a shell story is the status item, not the panel")
        XCTAssertGreaterThan(drawn.height, 400)
    }

    func testStationAppNeverLinksTheLoader() throws {
        let files = FileManager.default.enumerator(at: Tree.app, includingPropertiesForKeys: nil)!.compactMap { $0 as? URL }
            .filter { $0.pathExtension == "swift" }
        XCTAssertGreaterThan(files.count, 0)
        for file in files {
            let text = try String(contentsOf: file, encoding: .utf8)
            for name in ["Stories", "StoryView", "StoryFile", "Story("] {
                XCTAssertFalse(text.contains(name), "\(file.lastPathComponent) reaches the story loader (\(name)), which only the tests, the Catalog and the previews read")
            }
        }
    }
}
