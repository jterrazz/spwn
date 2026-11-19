import AppKit
import Foundation
@testable import StationCatalog
import StationKit
import XCTest

final class CatalogExportTests: XCTestCase {
    static let repository = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
        .deletingLastPathComponent().deletingLastPathComponent()
    static let pages = repository.appendingPathComponent("apps/engine/ui/testdata/pages")

    var root: URL!
    var stories: URL { root.appendingPathComponent("stories") }
    var out: URL { root.appendingPathComponent("out") }

    override func setUpWithError() throws {
        root = FileManager.default.temporaryDirectory.appendingPathComponent("station-catalog-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: stories.appendingPathComponent("pages"), withIntermediateDirectories: true)
        for name in ["menubar.home.two-macs", "menubar.mac.mini"] {
            try FileManager.default.copyItem(at: Self.pages.appendingPathComponent(name + ".json"),
                                             to: stories.appendingPathComponent("pages/\(name).json"))
        }
        var item = StoryUI()
        item.statusItem = true
        for story in [
            StoryFile(id: "home.needs", group: .home, note: "Home with a need on top.", pages: ["pages/menubar.home.two-macs.json"]),
            StoryFile(id: "home.fine", group: .home, note: "Home when all is well.", pages: ["pages/menubar.home.two-macs.json"]),
            StoryFile(id: "mac.mini.overview", group: .mac, note: "The mini's page.",
                      pages: ["pages/menubar.home.two-macs.json", "pages/menubar.mac.mini.json"]),
            StoryFile(id: "menubar.count", group: .shell, note: "The status item with a count.",
                      pages: ["pages/menubar.home.two-macs.json"], ui: item),
        ] {
            try story.encoded().write(to: stories.appendingPathComponent(story.id + Stories.suffix))
        }
    }

    override func tearDownWithError() throws {
        try FileManager.default.removeItem(at: root)
    }

    struct Run {
        let code: Int32
        let output: String
        let error: String
    }

    func export(_ arguments: [String], design: URL? = nil) throws -> Run {
        let binary = Bundle(for: Self.self).bundleURL.deletingLastPathComponent().appendingPathComponent("StationCatalog")
        let process = Process()
        process.executableURL = binary
        process.arguments = arguments
        var environment = ProcessInfo.processInfo.environment
        environment["STATION_DESIGN"] = design?.path
        process.environment = environment
        let output = Pipe(), error = Pipe()
        process.standardOutput = output
        process.standardError = error
        let done = DispatchSemaphore(value: 0)
        process.terminationHandler = { _ in done.signal() }
        try process.run()
        if done.wait(timeout: .now() + 120) == .timedOut {
            process.terminate()
            XCTFail("StationCatalog \(arguments.joined(separator: " ")) did not return: it started the app instead of exporting")
        }
        return Run(code: process.terminationStatus,
                   output: String(decoding: output.fileHandleForReading.readDataToEndOfFile(), as: UTF8.self),
                   error: String(decoding: error.fileHandleForReading.readDataToEndOfFile(), as: UTF8.self))
    }

    func written() throws -> [String] {
        try FileManager.default.contentsOfDirectory(atPath: out.path).sorted()
    }

    func size(_ file: String) throws -> (Int, Int) {
        let rep = try XCTUnwrap(NSBitmapImageRep(data: Data(contentsOf: out.appendingPathComponent(file))), "\(file) is not an image")
        return (rep.pixelsWide, rep.pixelsHigh)
    }

    // invariant: SPEC C3's Play replays a transition over its movement's
    // run at 1, 0.25 and 0.1 of its speed; a story at rest has no replay.
    func testPlayReplaysATransitionOverItsRunAtEachSpeed() throws {
        try StoryFile(id: "transition.push", group: .transition, note: "n", pages: ["pages/menubar.home.two-macs.json", "pages/menubar.mac.mini.json"],
                      play: StoryPlay(from: "home.needs", gesture: "click mac-mini"), freeze: 0.35)
            .encoded().write(to: stories.appendingPathComponent("transition.push" + Stories.suffix))
        let loaded = try Stories.load(root: stories)
        let push = try XCTUnwrap(loaded.first { $0.id == "transition.push" })
        let runs = Catalog.speeds.compactMap { Catalog.replay(push, speed: $0) }
        XCTAssertEqual(runs.count, 3)
        for (run, want) in zip(runs, [0.34, 1.36, 3.4]) { XCTAssertEqual(run, want, accuracy: 1e-9) }
        XCTAssertNil(Catalog.replay(try XCTUnwrap(loaded.first { $0.id == "home.needs" }), speed: 1))
    }

    func testTheExportDrawsEveryStoryTheFilterNamesInDarkAndLightThenReturns() throws {
        let run = try export(["--export", out.path, "--filter", "^home", "--stories", stories.path])
        XCTAssertEqual(run.code, 0, run.error)
        XCTAssertEqual(try written(), ["home.fine.dark.png", "home.fine.light.png", "home.needs.dark.png", "home.needs.light.png"])
        XCTAssertEqual(run.output.split(separator: "\n").map { URL(fileURLWithPath: String($0)).lastPathComponent }.sorted(), try written())
        let (width, height) = try size("home.needs.dark.png")
        XCTAssertEqual(width, Int((Token.Space.width + 2 * StoryStage.room) * 2))
        XCTAssertGreaterThan(height, 200)
    }

    func testTheExportWithoutAFilterDrawsEveryStory() throws {
        let run = try export(["--export", out.path, "--stories", stories.path])
        XCTAssertEqual(run.code, 0, run.error)
        XCTAssertEqual(try written().count, 8)
    }

    func testAShellStoryIsTheStatusItemInAStripOfTheMenuBar() throws {
        let run = try export(["--export", out.path, "--filter", "^menubar", "--stories", stories.path])
        XCTAssertEqual(run.code, 0, run.error)
        XCTAssertEqual(try written(), ["menubar.count.dark.png", "menubar.count.light.png"])
        let (width, height) = try size("menubar.count.light.png")
        XCTAssertEqual(width, Int(StoryStage.strip.width * 2))
        XCTAssertEqual(height, Int(StoryStage.strip.height * 2))
    }

    func testAFilterThatMatchesNothingSaysSoAndWritesNothing() throws {
        let run = try export(["--export", out.path, "--filter", "^nothing", "--stories", stories.path])
        XCTAssertEqual(run.code, 1)
        XCTAssertEqual(run.error, "Error: no story matches ^nothing\n")
        XCTAssertFalse(FileManager.default.fileExists(atPath: out.path))
    }

    @MainActor
    func testTheWidthKnobSetsThePanelsWidth() throws {
        let story = try XCTUnwrap(Stories.load(root: stories).first { $0.id == "home.needs" })
        var widths: [Int] = []
        for width in StoryKnobs.widths {
            let png = try XCTUnwrap(StoryStage.png(story, knobs: StoryKnobs(width: width, scale: 1)))
            widths.append(try XCTUnwrap(NSBitmapImageRep(data: png)).pixelsWide)
        }
        XCTAssertEqual(widths, StoryKnobs.widths.map { Int($0 + 2 * StoryStage.room) })
    }

    var goldens: URL { root.appendingPathComponent("goldens") }
    var design: URL { root.appendingPathComponent("design") }

    func gallery(design: URL? = nil) throws -> String {
        try """
        # Stories

        ## Menu bar and panel shell

        ### `menubar.count`

        ## All machines (home)

        ### `home.needs`
        ### `home.fine`

        ## A Mac

        ### `mac.mini.overview`
        """.write(to: stories.appendingPathComponent("STORIES.md"), atomically: true, encoding: .utf8)
        try FileManager.default.createDirectory(at: goldens, withIntermediateDirectories: true)
        let run = try export(["--export", out.path, "--gallery", "--stories", stories.path, "--goldens", goldens.path], design: design)
        XCTAssertEqual(run.code, 0, run.error)
        XCTAssertEqual(run.output.split(separator: "\n").last.map(String.init), out.appendingPathComponent("index.html").path)
        return try String(contentsOf: out.appendingPathComponent("index.html"), encoding: .utf8)
    }

    func cards(_ page: String) -> [String] {
        page.components(separatedBy: "<article ").dropFirst().map { String($0.prefix { $0 != ">" }) }
    }

    func sources(_ page: String) -> [String] {
        page.components(separatedBy: " src=\"").dropFirst().map { String($0.prefix { $0 != "\"" }) }
            + page.components(separatedBy: " href=\"").dropFirst().map { String($0.prefix { $0 != "\"" }) }.filter { !$0.hasPrefix("#") }
    }

    func testTheGalleryHasOneCardPerStoryInTheCataloguesOrderUnderItsGroups() throws {
        let page = try gallery()
        XCTAssertEqual(cards(page), ["id=\"menubar.count\"", "id=\"home.needs\"", "id=\"home.fine\"", "id=\"mac.mini.overview\""])
        XCTAssertTrue(page.contains("<nav><a href=\"#g-menu-bar-and-panel-shell\">Menu bar and panel shell</a><a href=\"#g-all-machines--home-\">All machines (home)</a>"
                                    + "<a href=\"#g-a-mac\">A Mac</a></nav>"), page)
        XCTAssertTrue(page.contains("<p>Home with a need on top.</p>"))
        XCTAssertTrue(page.contains("<img loading=\"lazy\" src=\"home.needs.dark.png\""))
        XCTAssertTrue(page.contains("<img loading=\"lazy\" src=\"home.needs.light.png\""))
        XCTAssertTrue(page.contains("4 stories, 0 failing"))
        XCTAssertFalse(page.contains("Prototype"), "no design folder, no prototype")
    }

    func testAFailingStoryIsRedWithItsActualBesideItsGolden() throws {
        try FileManager.default.createDirectory(at: goldens, withIntermediateDirectories: true)
        try Data("golden".utf8).write(to: goldens.appendingPathComponent("home.fine.light.png"))
        try Data("actual".utf8).write(to: goldens.appendingPathComponent("home.fine.light.actual.png"))
        let page = try gallery()
        XCTAssertEqual(cards(page).filter { $0.contains("failing") }, ["id=\"home.fine\" class=\"failing\""])
        XCTAssertTrue(page.contains("Failing in light"))
        XCTAssertTrue(page.contains("1 failing"))
        XCTAssertEqual(try Data(contentsOf: out.appendingPathComponent("golden/home.fine.light.png")), Data("golden".utf8))
        XCTAssertEqual(try Data(contentsOf: out.appendingPathComponent("actual/home.fine.light.png")), Data("actual".utf8))
        XCTAssertTrue(page.contains("src=\"golden/home.fine.light.png\""))
        XCTAssertTrue(page.contains("src=\"actual/home.fine.light.png\""))
    }

    func testThePrototypesCaptureSitsUnderTheStoryWhenTheDesignFolderIsThere() throws {
        let captures = design.appendingPathComponent("v4/stories")
        try FileManager.default.createDirectory(at: captures, withIntermediateDirectories: true)
        try Data("proto".utf8).write(to: captures.appendingPathComponent("home.needs.png"))
        let page = try gallery(design: design)
        XCTAssertTrue(page.contains("src=\"prototype/home.needs.png\""))
        XCTAssertEqual(page.components(separatedBy: "Prototype, dark").count - 1, 2, "one figure and its alt, for home.needs alone")
    }

    func testTheGalleryOpensOfflineFromItsFolder() throws {
        try FileManager.default.createDirectory(at: goldens, withIntermediateDirectories: true)
        try Data("actual".utf8).write(to: goldens.appendingPathComponent("home.fine.dark.actual.png"))
        let captures = design.appendingPathComponent("v4/stories")
        try FileManager.default.createDirectory(at: captures, withIntermediateDirectories: true)
        try Data("proto".utf8).write(to: captures.appendingPathComponent("home.fine.light.png"))
        let page = try gallery(design: design)
        XCTAssertFalse(page.contains(root.path), "the page names no absolute path")
        XCTAssertFalse(page.contains("http"), "the page loads nothing from the network")
        let files = sources(page)
        XCTAssertGreaterThan(files.count, 16)
        for file in files {
            XCTAssertFalse(file.hasPrefix("/"), file)
            XCTAssertTrue(FileManager.default.fileExists(atPath: out.appendingPathComponent(file).path), "\(file) is not in the gallery's folder")
        }
    }
}
