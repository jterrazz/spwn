import Foundation
import StationKit
import SwiftUI

enum Export {
    struct Request: Equatable {
        var folder: URL
        var filter: String?
        var stories = Paths.stories
        var gallery = false
        var goldens = Paths.goldens
    }

    enum Failure: Error, CustomStringConvertible {
        case usage
        case filter(String)
        case none(String?)
        case render(String)

        var description: String {
            switch self {
            case .usage: "usage: StationCatalog --export <dir> [--filter <regex>] [--stories <dir>] [--gallery [--goldens <dir>]]"
            case let .filter(pattern): "--filter \(pattern) is not a regular expression"
            case let .none(pattern): pattern.map { "no story matches \($0)" } ?? "no story to export"
            case let .render(file): "\(file): nothing rendered"
            }
        }
    }

    static func parse(_ arguments: [String]) throws -> Request {
        func value(_ flag: String) throws -> String? {
            guard let i = arguments.firstIndex(of: flag) else { return nil }
            guard i + 1 < arguments.count, !arguments[i + 1].hasPrefix("--") else { throw Failure.usage }
            return arguments[i + 1]
        }
        guard let folder = try value("--export") else { throw Failure.usage }
        var request = Request(folder: URL(fileURLWithPath: folder), filter: try value("--filter"))
        if let stories = try value("--stories") { request.stories = URL(fileURLWithPath: stories) }
        if let goldens = try value("--goldens") { request.goldens = URL(fileURLWithPath: goldens) }
        request.gallery = arguments.contains("--gallery")
        return request
    }

    static func select(_ stories: [Story], filter: String?) throws -> [Story] {
        guard let filter else { return stories }
        guard let regex = try? Regex(filter) else { throw Failure.filter(filter) }
        return stories.filter { $0.id.contains(regex) }
    }

    @MainActor
    static func run(_ arguments: [String]) -> Int32 {
        do {
            let request = try parse(arguments)
            let stories = try select(try Stories.load(root: request.stories), filter: request.filter)
            guard !stories.isEmpty else { throw Failure.none(request.filter) }
            try FileManager.default.createDirectory(at: request.folder, withIntermediateDirectories: true)
            for story in stories {
                for scheme in [ColorScheme.dark, .light] {
                    let file = "\(story.id).\(scheme == .dark ? "dark" : "light").png"
                    guard let png = StoryStage.png(story, knobs: StoryKnobs(scheme: scheme)) else { throw Failure.render(file) }
                    try png.write(to: request.folder.appendingPathComponent(file))
                    print(request.folder.appendingPathComponent(file).path)
                }
            }
            if request.gallery {
                print(try Gallery.write(stories, to: request.folder, root: request.stories, goldens: request.goldens, design: Paths.design).path)
            }
            return 0
        } catch {
            FileHandle.standardError.write(Data("Error: \(error)\n".utf8))
            return 1
        }
    }
}
