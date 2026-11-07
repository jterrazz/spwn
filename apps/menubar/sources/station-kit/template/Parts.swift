import SwiftUI

// value: a measure as the vision's tile: its name in 11 pt semibold
// secondary, its figure in 17 pt with what it is out of small after it,
// its sparkline in the measure's hue with a dot where it ends.
struct Metric: View {
    let label: String
    let value: Value?
    let spark: [Int]?
    @Environment(\.stationNow) private var now
    @Environment(\.colorScheme) private var scheme

    var waiting: Bool { value == nil || value?.kind == "pending" }

    var body: some View {
        let (number, unit, of) = Self.parts(value.map { Format.show($0, now: now) } ?? "")
        VStack(alignment: .leading, spacing: Token.Gap.small) {
            Text(label).font(.system(size: Token.Size.label, weight: .semibold)).foregroundStyle(Token.Ink.secondary).lineLimit(1)
                .padding(.vertical, Token.Size.labelLeading)
            Self.figure(Token.Glyph.missing, nil).hidden().padding(.vertical, Token.Size.figureLeading).overlay(alignment: .leading) {
                if waiting || number.isEmpty {
                    Self.figure(Token.Glyph.missing, nil).foregroundStyle(Token.Ink.tertiary)
                } else {
                    Self.figure(of == nil && !unit.isEmpty && unit.count <= 1 ? "\(number) \(unit)" : number, of ?? (unit.count > 1 ? unit : nil))
                        .fixedSize()
                }
            }
            SwiftUI.Group {
                if let spark, spark.count > 1, !waiting {
                    Spark(Self.scaled(spark), height: Token.Tile.spark, tint: Token.Hue.measure(label, scheme))
                } else {
                    Color.clear
                }
            }
            .frame(height: Token.Tile.spark)
        }
        .stationTile()
    }

    // value: a figure: the number in a 17 pt semibold, what follows it
    // (its unit, what it is out of) in 11 pt secondary.
    static func figure(_ number: String, _ small: String?) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: Token.Gap.small) {
            Text(number).font(.system(size: Token.Tile.figure, weight: .semibold)).tracking(Token.Detail.figureTracking * Token.Tile.figure).monospacedDigit().contentTransition(.numericText())
            if let small, !small.isEmpty {
                Text(small).font(.system(size: Token.Size.label, weight: .medium)).foregroundStyle(Token.Ink.secondary)
            }
        }
        .lineLimit(1)
    }

    // value: a shown value cut into its number, its unit, and what it is
    // out of: `14.1/17.2 GB` is 14.1, GB, of 17.2 GB; a whole number loses
    // its `.0` (`16.0 GB` is 16 GB).
    static func parts(_ shown: String) -> (String, String, String?) {
        let (number, unit) = split(shown)
        if let cut = unit.range(of: Token.Glyph.sep) { return (whole(number), String(unit[..<cut.lowerBound]), String(unit[cut.upperBound...])) }
        guard unit.hasPrefix(Token.Glyph.slash) else { return (whole(number), unit, nil) }
        let total = unit.dropFirst().trimmingCharacters(in: .whitespaces)
        let words = total.split(separator: " ")
        return (whole(number), String(words.last ?? ""), "of \((words.dropLast().map { whole(String($0)) } + words.suffix(1).map(String.init)).joined(separator: " "))")
    }

    static func whole(_ number: String) -> String { number.hasSuffix(".0") ? String(number.dropLast(2)) : number }

    static func split(_ shown: String) -> (String, String) {
        let figure = Set("0123456789.,")
        let head = shown.prefix { figure.contains($0) }
        guard !head.isEmpty else { return (shown, "") }
        return (String(head), shown.dropFirst(head.count).trimmingCharacters(in: .whitespaces))
    }

    // value: a series drawn against its own peak, so a rate in KB/s draws
    // like a share in percent.
    static func scaled(_ points: [Int]) -> [Int] {
        let top = max(points.max() ?? 1, 1)
        return top <= 100 ? points : points.map { $0 * 100 / top }
    }
}

// value: the three measures of a machine, three tiles side by side.
struct Metrics: View {
    let measures: [(label: String, value: Value?, spark: [Int]?)]

    var body: some View {
        HStack(alignment: .top, spacing: Token.Tile.pair) {
            ForEach(Array(measures.enumerated()), id: \.offset) { _, m in
                Metric(label: m.label, value: m.value, spark: m.spark)
            }
        }
    }
}

// value: a machine's line as the vision's Machines: its tile (its glyph
// in the accent for This Mac, where the owner is), its name and kind's
// tag, its words, its state dot; the line opens its machine's page.
struct MachineRow: View {
    let row: Row
    let symbol: String
    let kinds: Bool
    var sub: String? = nil
    var picked = false
    var nested = false
    var lit = false
    let open: (String) -> Void
    @Environment(\.stationPalette) private var palette

    var kind: String { row.cells?.first?.text ?? "" }
    var words: String {
        if let sub { return sub }
        let rest = (row.cells ?? []).dropFirst().filter { !$0.isNone }.map { Self.said($0) }
        return ([kind == "Mac" ? kind : nil] + rest.map { Optional($0) }).compactMap { $0 }.joined(separator: Token.Glyph.sep)
    }

    // value: a machine's words, its restarts as the vision's `↺ 0`.
    static func said(_ v: Value) -> String {
        guard v.kind == "list" else { return Format.show(v) }
        return (v.items ?? []).map { i in
            i.kind == "count" && i.unit == "restarts" ? "\(Token.Glyph.restart) \(i.n ?? 0)" : said(i)
        }.filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
    }

    var badge: String? { (kinds || kind == "Server") && kind != "Mac" && !kind.isEmpty ? kind : nil }

    var body: some View {
        Button { open(row.id ?? "") } label: {
            HStack(spacing: Token.Tile.gap) {
                IconTile(symbol, active: row.label == Page.ThisMacWord && !picked, tint: picked ? palette.onTint : nil, pulse: row.mark == .pending)
                RowText(title: row.label ?? "", subtitle: words.isEmpty ? nil : words, lines: 1) {
                    if let badge { KindTag(badge) }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                if picked {
                    Image(systemName: Token.Symbol.check).font(.system(size: Token.Detail.check, weight: .semibold)).foregroundStyle(palette.onTint)
                } else {
                    StateDot(mark: row.mark)
                }
            }
            .padding(EdgeInsets(top: Token.Gap.row, leading: nested ? Token.Tile.indent : (picked ? Token.Gap.pair : Token.Gap.edge), bottom: Token.Gap.row, trailing: picked ? Token.Gap.pair : Token.Gap.edge))
            .frame(minHeight: Token.Tile.row)
            .foregroundStyle(picked ? AnyShapeStyle(palette.onTint) : AnyShapeStyle(Token.Ink.primary), picked ? AnyShapeStyle(palette.onTint.opacity(Token.Fill.onTintSub)) : AnyShapeStyle(Token.Ink.secondary))
            .environment(\.colorScheme, picked ? .dark : envScheme)
            .background {
                if picked {
                    RoundedRectangle(cornerRadius: Token.Tile.fieldCorner, style: .continuous).fill(palette.accent.opacity(Token.Fill.selected))
                }
                if lit {
                    RoundedRectangle(cornerRadius: Token.Tile.fieldCorner, style: .continuous).strokeBorder(palette.accent, lineWidth: Token.Stroke.focus)
                }
            }
            .padding(.horizontal, picked || lit ? Token.Gap.small : 0)
            .stationHover()
        }
        .buttonStyle(.plain)
    }

    @Environment(\.colorScheme) private var envScheme
}

// invariant: SPEC §10.2's Machines as the vision draws them: one list,
// each Mac, then its groups (Workspaces, Servers) captioned and their VMs
// nested under it, a hairline between two lines; a line opens its
// machine's page.
struct MachineTree: View {
    let section: Section
    let kinds: Bool
    let only: Set<String>?
    var lit: String? = nil
    var stopping: Set<String> = []
    let open: (String) -> Void

    enum Line {
        case caption(String)
        case row(Row, nested: Bool, symbol: String)

        // value: a line's own identity, its machine's name or its caption's
        // words, never its place: a VM inserted above keeps every other line
        // drawn as itself, its icon included.
        var key: String {
            switch self {
            case let .caption(text): "caption:\(text)"
            case let .row(row, _, _): "row:\(row.id ?? row.label ?? "")"
            }
        }
    }

    var body: some View {
        let all = lines()
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Self.keyed(all), id: \.key) { i, _, l in
                switch l {
                case let .caption(text):
                    GroupCaption(text)
                case let .row(row, nested, symbol):
                    if i > 0, case .row = all[i - 1] { RowSeparator() }
                    MachineRow(row: row, symbol: symbol, kinds: kinds, nested: nested, lit: lit != nil && lit == row.id, open: open)
                }
            }
            ForEach(Array((section.items ?? []).compactMap(\.empty).enumerated()), id: \.offset) { _, empty in
                EmptyLine(empty).padding(Token.Tile.rowPadding)
            }
        }
        .stationRows()
    }

    // value: the lines with their identities, a key seen again (a second
    // Mac's Workspaces) told apart by how many times it came before.
    static func keyed(_ lines: [Line]) -> [(i: Int, key: String, line: Line)] {
        var seen: [String: Int] = [:]
        return lines.enumerated().map { i, l in
            let n = seen[l.key, default: 0]
            seen[l.key] = n + 1
            return (i, n == 0 ? l.key : "\(l.key)#\(n)", l)
        }
    }

    func said(_ r: Row) -> Row { stopping.contains(r.id ?? "") ? StopAll.shown(r) : r }

    func shown(_ r: Row) -> Bool { only == nil || only!.contains(r.id ?? "") }

    func lines() -> [Line] {
        var out: [Line] = []
        for item in section.items ?? [] {
            if let g = item.group, g.head?.cells?.isEmpty ?? true {
                out += nested(g.items ?? [], caption: g.head?.label, indent: false)
            } else if let mac = item.group {
                if let head = mac.head, shown(head) { out.append(.row(head, nested: false, symbol: Token.Symbol.mac)) }
                out += nested(mac.items ?? [], caption: nil, indent: only == nil)
            } else if let row = item.row, shown(row) {
                out.append(.row(row, nested: false, symbol: Token.Symbol.machine(row.cells?.first?.text)))
            }
        }
        return out
    }

    func nested(_ items: [Item], caption: String?, indent: Bool) -> [Line] {
        var out: [Line] = []
        for item in items {
            if let g = item.group {
                let rows = (g.items ?? []).compactMap(\.row).filter(shown)
                if !rows.isEmpty {
                    if only == nil, let label = g.head?.label { out.append(.caption(label)) }
                    out += rows.map { .row(said($0), nested: indent, symbol: Token.Symbol.machine($0.cells?.first?.text)) }
                }
            } else if let row = item.row, shown(row) {
                out.append(.row(said(row), nested: indent, symbol: Token.Symbol.machine(row.cells?.first?.text)))
            }
        }
        return out
    }
}

// value: a menu line of the panel, as a macOS menu's item: its symbol,
// its words, its key on the right.
struct MenuLine: View {
    let action: Action
    let symbol: String
    let danger: Bool
    let tap: () -> Void

    var title: String { (action.form != nil || action.confirm?.type != nil) ? "\(action.label ?? "")\(Token.Glyph.ellipsis)" : action.label ?? "" }

    var body: some View {
        Button(action: tap) {
            MenuText(symbol: symbol, title: title, key: action.key.map(Token.Key.shown), danger: danger)
        }
        .buttonStyle(.plain)
    }
}

struct MenuText: View {
    let symbol: String
    let title: String
    let key: String?
    let danger: Bool
    @Environment(\.stationPalette) private var palette

    var body: some View {
        HStack(spacing: Token.Tile.gap) {
            MenuGlyph(symbol).frame(width: Token.Detail.menuStroke)
                .foregroundStyle(palette.glyph)
            Text(title).font(.system(size: Token.Size.text)).foregroundStyle(danger ? AnyShapeStyle(palette.bad) : AnyShapeStyle(Token.Ink.primary))
            Spacer()
            if let key {
                Text(key).font(.system(size: Token.Size.key)).foregroundStyle(Token.Ink.secondary)
            }
        }
        .padding(Token.Gap.item)
        .stationHover()
    }
}

// invariant: a gesture opens its sheet (a form, or a question at its
// call's level) when it has one, and runs at once when it has none.
func perform(_ action: Action, _ handlers: PopoverHandlers) {
    if action.confirm != nil || action.form != nil {
        handlers.confirm(action)
    } else {
        handlers.run(action, [:])
    }
}

// value: the sheet of the action being confirmed, if any, on this page.
struct SheetFor: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers

    var body: some View {
        if let id = state.confirming, let action = page.action(id) {
            let answers = Binding(get: { state.answers[id] ?? [:] }, set: { new in for (k, v) in new { handlers.answer(id, k, v) } })
            if let form = action.form {
                FormSheet(form, run: action.label ?? "", answers: answers) { yes in
                    handlers.confirm(nil)
                    if yes { handlers.run(action, state.answers[id] ?? [:]) }
                }
            } else if let confirm = action.confirm {
                let typed = Binding(get: { state.answers[id]?[ActionRunner.typedKey] ?? "" }, set: { handlers.answer(id, ActionRunner.typedKey, $0) })
                ConfirmSheet(confirm, typed: typed) { yes in
                    handlers.confirm(nil)
                    if yes { handlers.run(action, state.answers[id] ?? [:]) }
                }
            }
        }
    }
}

enum Symbols {
    static func of(_ a: Action) -> String {
        switch a.id {
        case "new-vm": Token.Symbol.plus
        case "open-station", "start": Token.Symbol.prompt
        case "stop-all": Token.Symbol.stop
        case "restart": Token.Symbol.restart
        case "unlock": Token.Symbol.unlock
        default: Token.Symbol.forward
        }
    }
}

extension Section {
    // value: a section's header in the vision's words: its title, a
    // colon cut into a separator, then its caption.
    var header: String {
        let words = (caption ?? []).map { Format.show($0) }
        return ([(title ?? "").replacingOccurrences(of: ": ", with: Token.Glyph.sep)] + words).filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
    }
}
