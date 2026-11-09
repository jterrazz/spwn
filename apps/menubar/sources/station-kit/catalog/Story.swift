import Foundation

public let storySchema = "station.story/v1"

public enum StoryGroup: String, Codable, CaseIterable, Sendable {
    case shell, home, needs, mac, workspace, agent, server, ports, accounts, awake, sheet, alert, menu, pop, find, transition

    public static func of(_ id: String) -> StoryGroup? {
        let head = String(id.prefix { $0 != "." })
        return head == "menubar" ? .shell : StoryGroup(rawValue: head)
    }
}

public struct SheetState: Codable, Equatable, Sendable {
    public var id: String
    public var answers: [String: String]?

    public init(id: String, answers: [String: String]? = nil) {
        self.id = id
        self.answers = answers
    }
}

public struct MenuOpen: Codable, Equatable, Sendable {
    public var id: String?
    public var row: String?
    // value: the entry whose submenu is open beside it, by its id.
    public var submenu: String?

    public init(id: String? = nil, row: String? = nil, submenu: String? = nil) {
        self.id = id
        self.row = row
        self.submenu = submenu
    }
}

public struct FindState: Codable, Equatable, Sendable {
    public var query: String
    public var highlight: Int?
    public var open: [String]?

    public init(query: String, highlight: Int? = nil, open: [String]? = nil) {
        self.query = query
        self.highlight = highlight
        self.open = open
    }
}

public struct StoryUI: Codable, Equatable, Sendable {
    public var connection: String?
    public var segment: String?
    public var window: String?
    public var unfolded: [String]?
    public var hover: String?
    public var highlight: String?
    public var focus: String?
    public var sheet: SheetState?
    public var alert: String?
    public var menu: MenuOpen?
    public var popover: String?
    public var find: FindState?
    public var recent: [String]?
    public var scrub: Int?
    public var pending: [String]?
    public var prOpened: [String]?
    public var reply: String?
    public var hud: String?
    public var tooltip: String?
    public var statusItem: Bool?
    public var pressed: Bool?
    // value: the app's own Open at login switch, as the station menu shows it.
    public var loginItem: Bool?

    public init() {}
}

public struct StoryPlay: Codable, Equatable, Sendable {
    public var from: String
    public var gesture: String

    public init(from: String, gesture: String) {
        self.from = from
        self.gesture = gesture
    }
}

public struct StoryFile: Codable, Equatable, Sendable {
    public var schema: String
    public var id: String
    public var group: StoryGroup
    public var note: String
    public var status: String?
    public var pages: [String]
    public var ui: StoryUI?
    public var samples: [String: Double]?
    public var play: StoryPlay?
    public var freeze: Double?
    public var heightNote: String?

    public init(id: String, group: StoryGroup, note: String, pages: [String], status: String? = nil, ui: StoryUI? = nil,
                samples: [String: Double]? = nil, play: StoryPlay? = nil, freeze: Double? = nil, heightNote: String? = nil) {
        schema = storySchema
        self.id = id
        self.group = group
        self.note = note
        self.status = status
        self.pages = pages
        self.ui = ui
        self.samples = samples
        self.play = play
        self.freeze = freeze
        self.heightNote = heightNote
    }

    public var today: Bool { status == "today" }

    public func encoded() throws -> Data {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys, .prettyPrinted, .withoutEscapingSlashes]
        return try encoder.encode(self) + Data("\n".utf8)
    }
}

public struct Story: Identifiable, Sendable {
    public let file: StoryFile
    public let pages: [Page]

    public var id: String { file.id }
    public var group: StoryGroup { file.group }
    public var note: String { file.note }
    public var ui: StoryUI { file.ui ?? StoryUI() }
    public var samples: [String: Double] { file.samples ?? [:] }
    public var play: StoryPlay? { file.play }
    public var freeze: Double? { file.freeze }
}

public enum StoryError: Error, Equatable, CustomStringConvertible {
    case schema(file: String, schema: String)
    case id(file: String, id: String)
    case group(id: String, group: StoryGroup)
    case noPages(id: String)
    case missingPage(id: String, page: String)
    case undecodable(id: String, page: String, reason: String)

    public var description: String {
        switch self {
        case let .schema(file, schema): "\(file): schema \"\(schema)\", want \"\(storySchema)\""
        case let .id(file, id): "\(file): names the story \"\(id)\", not its file's"
        case let .group(id, group): "\(id): in the group \(group.rawValue), not its id's"
        case let .noPages(id): "\(id): names no page"
        case let .missingPage(id, page): "\(id): its page \(page) does not exist"
        case let .undecodable(id, page, reason): "\(id): its page \(page) does not decode: \(reason)"
        }
    }
}

public enum Stories {
    public static let suffix = ".story.json"
    public static let root = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent()
        .deletingLastPathComponent().deletingLastPathComponent().appendingPathComponent("stories")

    public static subscript(id: String) -> Story {
        guard let story = try? load(root: root).first(where: { $0.id == id }) else { preconditionFailure("\(id): no story file loads under \(root.path)") }
        return story
    }

    public static func files(root: URL) throws -> [URL] {
        try FileManager.default.contentsOfDirectory(at: root, includingPropertiesForKeys: nil)
            .filter { $0.lastPathComponent.hasSuffix(suffix) }
            .sorted { $0.lastPathComponent < $1.lastPathComponent }
    }

    public static func id(of file: URL) -> String { String(file.lastPathComponent.dropLast(suffix.count)) }

    public static func read(_ file: URL) throws -> StoryFile {
        let story = try JSONDecoder().decode(StoryFile.self, from: Data(contentsOf: file))
        let name = file.lastPathComponent
        guard story.schema == storySchema else { throw StoryError.schema(file: name, schema: story.schema) }
        guard story.id == id(of: file) else { throw StoryError.id(file: name, id: story.id) }
        guard StoryGroup.of(story.id) == story.group else { throw StoryError.group(id: story.id, group: story.group) }
        guard !story.pages.isEmpty else { throw StoryError.noPages(id: story.id) }
        return story
    }

    public static func page(_ path: String, of story: StoryFile, root: URL) throws -> Page {
        let url = root.appendingPathComponent(path).standardizedFileURL
        guard let data = try? Data(contentsOf: url) else { throw StoryError.missingPage(id: story.id, page: path) }
        do {
            return try Page.decode(data)
        } catch {
            throw StoryError.undecodable(id: story.id, page: path, reason: String(describing: error))
        }
    }

    public static func load(root: URL) throws -> [Story] {
        try files(root: root).map { file in
            let story = try read(file)
            return Story(file: story, pages: try story.pages.map { try page($0, of: story, root: root) })
        }
    }
}
