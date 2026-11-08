import Foundation

public let pageSchema = "station.page/v1"
public let pageRevision = 4

public struct Page: Codable, Equatable, Sendable {
    public var schema: String?
    public var id: String?
    public var command: String?
    public var context: Context?
    public var state: String?
    public var strip: Strip?
    public var sections: [Section]?
    public var hints: [Hint]?
    public var error: PageError?
    public var actions: [Action]?
    public var badge: Badge?
    public var header: Header?
    public var segments: [Segment]?
    public var footer: Footer?
    public var live: [String]?

    public static func decode(_ data: Data) throws -> Page {
        let page = try JSONDecoder().decode(Page.self, from: data)
        guard page.schema == pageSchema else { throw DecodeError.schema(page.schema ?? "") }
        return page
    }

    public func encoded() throws -> Data {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys]
        return try encoder.encode(self)
    }

    public var loading: Bool { state == "loading" }
    public func section(_ id: String) -> Section? { sections?.first { $0.id == id } }
    public func action(_ id: String) -> Action? { actions?.first { $0.id == id } }
    public var pageActions: [Action] { (actions ?? []).filter { $0.on == "page" && $0.run != nil } }
    public func rowActions(_ row: String) -> [Action] { (actions ?? []).filter { $0.on == "row" && $0.run != nil && $0.rows?.contains(row) == true } }
}

public struct Badge: Codable, Equatable, Sendable {
    public init(needs: Int?, down: Bool? = nil) {
        self.needs = needs
        self.down = down
    }
    public var needs: Int?
    public var down: Bool?
    public var restarting: Bool?
    public var label: String?
}

public struct Header: Codable, Equatable, Sendable {
    public var back: String?
    public var glyph: Glyph?
    public var buttons: [String]?
    public var stale: Value?
}

public struct Segment: Codable, Equatable, Sendable {
    public var id: String?
    public var label: String?
    public var key: String?
    public var picked: Bool?
}

public struct Footer: Codable, Equatable, Sendable {
    public var kind: String?
    public var copy: Value?
    public var note: String?
    public var buttons: [String]?
}

public struct Glyph: Codable, Equatable, Sendable {
    public var kind: String?
    public var signal: String?
    public var dot: String?
    public var ring: String?
    public var spin: Bool?
    public var progress: Int?
}

public enum DecodeError: Error, Equatable {
    case schema(String)
}

public struct Context: Codable, Equatable, Sendable {
    public var machine: String?
    public var kind: String?
    public var registered: Bool?
    public var on: String?
    public init(machine: String? = nil, kind: String? = nil, registered: Bool? = nil, on: String? = nil) {
        self.machine = machine
        self.kind = kind
        self.registered = registered
        self.on = on
    }
}

public struct Section: Codable, Equatable, Sendable {
    public var id: String?
    public var title: String?
    public var role: String?
    public var fold: String?
    public var caption: [Value]?
    public var joined: Bool?
    public var columns: [Column]?
    public var items: [Item]?
    public var counts: [Tally]?
    public var uncounted: Bool?
    public var pipe_counts: Bool?
    public var pipe_groups: Bool?
    public var `break`: Int?
    public var says_dropped: Bool?
    public var top: Int?
    public var rest: Item?
    // value: the section in a few words, for a line that sums it up
    // (Needs you folded).
    public var glance: [Value]?
    public var segment: String?
    public var card: String?
    public var picker: [Action]?
}

public struct Column: Codable, Equatable, Sendable {
    public var key: String?
    public var label: String?
    public var kind: String?
    public var keep: Int?
    public var show: String?
    public var levels: [String]?
    public var whole: Bool?
}

public struct Item: Codable, Equatable, Sendable {
    public init(row: Row? = nil) { self.row = row }
    public var row: Row?
    public var fact: Fact?
    public var group: Group?
    public var empty: Empty?
    public var error: PageError?
    public var note: Note?
    public var link: Link?
    public var tile: Tile?
    public var gauge: PageGauge?
    public var chart: PageChart?
    public var message: Message?
    public var steps: [Step]?
    public var log: Log?
    public var uptime: Uptime?
}

public struct Tile: Codable, Equatable, Sendable {
    public var caption: String?
    public var glyph: Glyph?
    public var number: Value?
    public var word: String?
    public var signal: String?
    public var toggle: String?
    public var nav: String?
    public var plus: String?
}

public struct PageGauge: Codable, Equatable, Sendable {
    public var title: String?
    public var value: Value?
    public var spark: [Int]?
    public var bar: Value?
    public var stale: Bool?
}

public struct PageChart: Codable, Equatable, Sendable {
    public var kind: String?
    public var window: String?
    public var series: [Series]?
    public var peak: Peak?
    public var ends: [String]?
    public var anomaly: Anomaly?
}

public struct Series: Codable, Equatable, Sendable {
    public var key: String?
    public var points: [Double]?
    public var unit: String?
}

public struct Peak: Codable, Equatable, Sendable {
    public var index: Int?
    public var value: Value?
    public var at: String?
}

public struct Anomaly: Codable, Equatable, Sendable {
    public var from: Int?
    public var to: Int?
    public var sentence: String?
}

public struct Message: Codable, Equatable, Sendable {
    public var title: String?
    public var body: String?
    public var since: Value?
    public var pr: PR?
    public var reply: Bool?
}

public struct PR: Codable, Equatable, Sendable {
    public var number: Int?
    public var url: String?
    public var files: Int?
    public var plus: Int?
    public var minus: Int?
    public var opened: Value?
}

public struct Step: Codable, Equatable, Sendable {
    public var at: Value?
    public var text: String?
    public var diff: String?
    public var current: Bool?
}

public struct Log: Codable, Equatable, Sendable {
    public var lines: [String]?
    public var cause: Int?
    public var more: String?
}

public struct Uptime: Codable, Equatable, Sendable {
    public var days: [Day]?
    public var availability: Value?
}

public struct Day: Codable, Equatable, Sendable {
    public var date: String?
    public var mark: String?
    public var tip: String?
}

public struct Row: Codable, Equatable, Sendable {
    public init() {}
    public var id: String?
    public var mark: Mark?
    public var label: String?
    public var short: String?
    public var sentence: Bool?
    public var cells: [Value]?
    public var detail: Value?
    public var reason: String?
    public var hint: Hint?
    public var glyph: Glyph?
    public var mac: String?
    public var trail: String?
    public var verb: String?
    public var quick: String?
    public var nav: String?
    public var menu: [String]?
    public var signal: String?
    public var fresh: Bool?
    public var left: Value?
    public var birth: Value?
}

public struct Fact: Codable, Equatable, Sendable {
    public var key: String?
    public var value: Value?
}

public struct Group: Codable, Equatable, Sendable {
    public var head: Row?
    public var items: [Item]?
    public var fold: String?
}

public struct Empty: Codable, Equatable, Sendable {
    public var noun: String?
    public var text: String?
    public var hint: Hint?
    public var glyph: Glyph?
    public var sub: String?
    public var actions: [String]?
}

public struct PageError: Codable, Equatable, Sendable {
    public var message: String?
    public var hint: String?
}

public struct Note: Codable, Equatable, Sendable {
    public var text: String?
}

public struct Link: Codable, Equatable, Sendable {
    public var label: String?
    public var lead: Value?
    public var value: Value?
    public var detail: Value?
    public var page: String?
    public var row: String?
}

public struct Hint: Codable, Equatable, Sendable {
    public init(command: String? = nil) { self.command = command }
    public var command: String?
    public var external: Bool?
    public var key: String?
    public var text: String?
}

public struct Action: Codable, Equatable, Sendable {
    public var key: String?
    public var label: String?
    public var on: String?
    public var rows: [String]?
    public var confirm: Confirm?
    public var priority: Int?
    public var mutates: Bool?
    public var id: String?
    public var run: Run?
    public var form: Form?
    // value: the choice in force among its row's (Awake's term).
    public var picked: Bool?
    public var icon: String?
    public var `default`: Bool?
    public var busy: String?
    public var sheet: String?
    public var prefill: [String: String]?
    public var submenu: [String]?
    public var disabled: String?
    public var hud: Value?
    public var destructive: Bool?
}

public struct Run: Codable, Equatable, Sendable {
    public var call: String?
    public var `in`: [String: String]?
    public var terminal: Bool?
    public var argv: [String]?
    public var target: Target?
    public var copy: String?
    public var open: String?
}

public struct Target: Codable, Equatable, Sendable {
    public var name: String?
    public var on: String?
}

public struct Confirm: Codable, Equatable, Sendable {
    public var question: String?
    public var lines: [Value]?
    public var danger: Bool?
    public var type: String?
    public var yes: String?
    public var no: String?
    public var icon: String?
}

public struct Form: Codable, Equatable, Sendable {
    public var title: String?
    public var fields: [Field]?
    public var subtitle: String?
    public var primary: [Primary]?
    public var split: [String]?
}

public struct Primary: Codable, Equatable, Sendable {
    public var when: When?
    public var label: String?
}

public struct When: Codable, Equatable, Sendable {
    public var key: String?
    public var value: String?
}

public struct Field: Codable, Equatable, Sendable {
    public var key: String?
    public var label: String?
    public var kind: String?
    public var options: [Option]?
    public var `default`: String?
    public var note: String?
    public var required: Bool?
    public var stops: [Int]?
    public var limit: Int?
    public var limit_note: String?
    public var fields: [Field]?
    public var summary: Value?
    public var when: When?
}

public struct Option: Codable, Equatable, Sendable {
    public var value: String?
    public var label: String?
    public var note: String?
    public var disabled: String?
}

public struct Strip: Codable, Equatable, Sendable {
    public var cpu: Value?
    public var spark: [Int]?
    public var memory: Value?
    public var memory_spark: [Int]?
    public var net_spark: [Int]?
    public var battery: Value?
    public var net: [Value]?
    public var awake: Value?
    public var top: Section?
}

public struct Tally: Codable, Equatable, Sendable {
    public init(mark: Mark?, n: Int?) {
        self.mark = mark
        self.n = n
    }
    public var mark: Mark?
    public var n: Int?
}

public enum Mark: String, Codable, Sendable, CaseIterable {
    case ok, attention, absent, broken, pending
    case offHere = "off-here"
    case notForThisMachine = "not-for-this-machine"
    case notKnown = "unknown"

    public init(from decoder: Decoder) throws {
        let raw = try decoder.singleValueContainer().decode(String.self)
        self = Mark(rawValue: raw) ?? .absent
    }

    public var rank: Int {
        switch self {
        case .broken: 4
        case .attention: 3
        case .pending: 2
        case .absent, .offHere, .notKnown: 1
        case .ok, .notForThisMachine: 0
        }
    }
}

public struct Value: Codable, Equatable, Sendable {
    public var kind: String?
    public var text: String?
    public var n: Int64?
    public var of: Int64?
    public var at: String?
    public var unit: String?
    public var label: String?
    public var items: [Value]?
    public var stale: Bool?
    public var live: String?
    public var pri: Int?
    public var short: String?
    public var signal: String?
    public var nav: String?

    public init(kind: String? = nil, text: String? = nil) {
        self.kind = kind
        self.text = text
    }

    public var isNone: Bool { kind == nil || kind == "none" }
}
