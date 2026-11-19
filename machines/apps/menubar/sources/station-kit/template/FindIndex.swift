import Foundation

// value: one entry of the engine's Find index (SPEC B5), as Find draws
// it: its row, the group it is listed in, its title (a need's or a
// section's list of words after its label), a port's address, its scope,
// its tags, and for a VM's verb the name that offers it.
public struct FindEntry: Equatable, Sendable {
    public let row: Row
    public let group: String
    public let title: String
    public let address: String?
    public let scope: String?
    public let tags: [String]
    public let prefix: String?

    public init(row: Row, group: String, title: String, address: String?, scope: String?, tags: [String], prefix: String?) {
        self.row = row
        self.group = group
        self.title = title
        self.address = address
        self.scope = scope
        self.tags = tags
        self.prefix = prefix
    }

    // value: what a query is scored against: the title, then a port's
    // address.
    public var text: String { [title, address].compactMap { $0 }.joined(separator: Token.Glyph.sep) }
}

// value: a group of results as Find lists it: its title, the entries shown,
// and how many more "Show N more" opens.
public struct FindGroup: Equatable, Sendable {
    public let id: String
    public let title: String
    public var entries: [FindEntry]
    public var more: Int
}

// value: a line Find draws and the keys move through: a result, or a
// group's "Show N more".
public enum FindLine: Equatable, Sendable {
    case entry(FindEntry)
    case more(group: String, count: Int)
}

// invariant: SPEC A6, the one pure function Find draws from. An empty
// query lists the top three needs, the machines opened last, then This
// Mac's first awake action and the two New VM actions; a query scores each
// entry prefix 0, word prefix 1, substring 2, tag prefix 3, lists them
// Machines, Ports, Actions, Sections, the best first, three per group then
// "Show N more" unless the group was opened, and a query of two letters or
// more that begins a VM's name adds that VM's verbs to Actions first.
public struct FindIndex: Equatable, Sendable {
    public static let needs = "needs-you"
    public static let machines = "machines"
    public static let ports = "ports"
    public static let actions = "actions"
    public static let sections = "sections"
    public static let verbs = "verbs"
    public static let recent = "recent"
    public static let order = [machines, ports, actions, sections]
    public static let shown = 3
    public static let verbsFrom = 2
    public static let verbsOf = 2
    static let list = "list"
    static let awake = "awake"
    static let newVM = "new-vm:"

    public let entries: [FindEntry]
    let titles: [String: String]

    public init(_ page: Page) {
        var entries: [FindEntry] = []
        var titles: [String: String] = [:]
        for s in page.sections ?? [] {
            let id = s.id ?? ""
            titles[id] = s.title
            for item in s.items ?? [] {
                if let row = item.row {
                    entries.append(Self.entry(row, group: id, prefix: nil))
                } else if let g = item.group {
                    for row in (g.items ?? []).compactMap(\.row) {
                        entries.append(Self.entry(row, group: id, prefix: g.head?.label))
                    }
                }
            }
        }
        self.entries = entries
        self.titles = titles
    }

    static func entry(_ row: Row, group: String, prefix: String?) -> FindEntry {
        let detail = Self.words(row.detail)
        let tags = Self.words(row.cells?.first).map { $0.lowercased() }
        switch group {
        case ports:
            let scope = detail.dropFirst().joined(separator: Token.Glyph.sep)
            return FindEntry(row: row, group: group, title: row.label ?? "", address: detail.first, scope: scope.isEmpty ? nil : scope, tags: tags, prefix: prefix)
        case needs, sections:
            let rest = row.detail?.kind == Self.list ? detail : []
            return FindEntry(row: row, group: group, title: ([row.label ?? ""] + rest).joined(separator: Token.Glyph.sep), address: nil, scope: nil, tags: tags, prefix: prefix)
        default:
            let scope = detail.joined(separator: Token.Glyph.sep)
            return FindEntry(row: row, group: group, title: row.label ?? "", address: nil, scope: scope.isEmpty ? nil : scope, tags: tags, prefix: prefix)
        }
    }

    static func words(_ value: Value?) -> [String] {
        guard let value, !value.isNone else { return [] }
        if let items = value.items { return items.flatMap { words($0) } }
        let said = Format.show(value)
        return said.isEmpty ? [] : [said]
    }

    public func title(_ group: String) -> String {
        group == Self.recent ? Token.Find.recent : titles[group] ?? group
    }

    // value: an entry's rank for a query (lowercased): 0 its title begins
    // with it, 1 a word of it does, 2 it holds it, 3 a tag begins with it;
    // nil when nothing matches.
    public static func score(_ entry: FindEntry, query q: String) -> Int? {
        let text = entry.text.lowercased()
        if text.hasPrefix(q) { return 0 }
        if text.components(separatedBy: Token.Find.wordBreaks).contains(where: { $0.hasPrefix(q) }) { return 1 }
        if text.contains(q) { return 2 }
        if entry.tags.contains(where: { $0.hasPrefix(q) }) { return 3 }
        return nil
    }

    public func results(query: String, recent: [String], open: Set<String> = []) -> [FindGroup] {
        let q = query.trimmingCharacters(in: .whitespaces).lowercased()
        return q.isEmpty ? opening(recent: recent) : found(q, open: open)
    }

    func opening(recent: [String]) -> [FindGroup] {
        let needs = entries.filter { $0.group == Self.needs }
        let machines = entries.filter { $0.group == Self.machines }
        let seen = recent.prefix(Self.shown).compactMap { id in machines.first { $0.row.id == id } }
        let actions = entries.filter { $0.group == Self.actions }
        let picked = actions.filter { ($0.row.id ?? "").hasPrefix(Self.awake) }.prefix(1) + actions.filter { ($0.row.id ?? "").hasPrefix(Self.newVM) }.prefix(2)
        return [
            FindGroup(id: Self.needs, title: title(Self.needs), entries: Array(needs.prefix(Self.shown)), more: 0),
            FindGroup(id: Self.recent, title: title(Self.recent), entries: seen, more: 0),
            FindGroup(id: Self.actions, title: title(Self.actions), entries: Array(picked), more: 0),
        ].filter { !$0.entries.isEmpty }
    }

    func found(_ q: String, open: Set<String>) -> [FindGroup] {
        var verbs: [FindEntry] = []
        if q.count >= Self.verbsFrom {
            var named: [String] = []
            for e in entries where e.group == Self.verbs {
                guard let name = e.prefix, name.lowercased().hasPrefix(q) else { continue }
                if !named.contains(name) {
                    guard named.count < Self.verbsOf else { continue }
                    named.append(name)
                }
                verbs.append(e)
            }
        }
        return Self.order.compactMap { id in
            let scored = entries.enumerated().compactMap { i, e -> (rank: Int, at: Int, entry: FindEntry)? in
                guard e.group == id, let s = Self.score(e, query: q) else { return nil }
                return (s, i, e)
            }
            .sorted { ($0.rank, $0.at) < ($1.rank, $1.at) }
            .map(\.entry)
            let all = (id == Self.actions ? verbs : []) + scored
            guard !all.isEmpty else { return nil }
            let opened = open.contains(id) || all.count <= Self.shown
            return FindGroup(id: id, title: title(id), entries: opened ? all : Array(all.prefix(Self.shown)), more: opened ? 0 : all.count - Self.shown)
        }
    }

    // value: the lines the keys move through, in the order they are drawn.
    public static func lines(_ groups: [FindGroup]) -> [FindLine] {
        groups.flatMap { g in g.entries.map(FindLine.entry) + (g.more > 0 ? [.more(group: g.id, count: g.more)] : []) }
    }
}

// value: Find open: what is typed, the line the keys light, and the groups
// opened past their three.
public struct FindQuery: Equatable, Sendable {
    public var query: String
    public var selected: Int
    public var open: [String]

    public init(_ query: String = "", selected: Int = 0, open: [String] = []) {
        self.query = query
        self.selected = selected
        self.open = open
    }

    // invariant: typing starts the results over: the first line lit, no
    // group opened.
    public func typed(_ text: String) -> FindQuery { FindQuery(text) }

    public func moved(_ by: Int, lines: Int) -> FindQuery {
        var q = self
        q.selected = lines == 0 ? 0 : min(max(selected + by, 0), lines - 1)
        return q
    }

    public func opening(_ group: String) -> FindQuery {
        var q = self
        if !q.open.contains(group) { q.open.append(group) }
        return q
    }
}
