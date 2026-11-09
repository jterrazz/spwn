import SwiftUI

// invariant: SPEC A4 Home 2: the three tiles side by side, as tall; a
// tile opens its page, its switch acts on its own.
public struct TileStrip: View {
    let section: Section
    let actions: [Action]
    let run: (Action) -> Void
    let open: (String) -> Void

    public init(_ section: Section, actions: [Action] = [], run: @escaping (Action) -> Void = { _ in }, open: @escaping (String) -> Void = { _ in }) {
        self.section = section
        self.actions = actions
        self.run = run
        self.open = open
    }

    var tiles: [Tile] { (section.items ?? []).compactMap(\.tile) }

    public var body: some View {
        HStack(spacing: Token.Metric.Tiles.gap) {
            ForEach(Array(tiles.enumerated()), id: \.offset) { _, tile in
                TileView(tile, actions: actions, run: run)
                    .onTapGesture { if let nav = tile.nav { open(nav) } }
                    .stationContextMenu(MenuModel.tile(tile, actions: actions), nav: tile.nav)
                    .menuAnchor(MenuAnchors.row(actions.first { $0.id == tile.toggle }?.rows?.first ?? tile.caption ?? ""))
                    .accessibilityElement(children: .contain)
                    .accessibilityLabel(Text(Self.said(tile)))
                    .accessibilityAddTraits(.isButton)
            }
        }
    }

    // value: what VoiceOver says of a tile: its caption, its number and
    // its word ("Ports, 5 shared").
    public static func said(_ tile: Tile) -> String {
        let number = tile.number.map { Format.show($0) } ?? ""
        return "\(tile.caption ?? ""), \([number, tile.word ?? ""].filter { !$0.isEmpty }.joined(separator: " "))"
    }
}

// invariant: a flush card of Home: its label and its caption, then its
// rows in the page's order, each navigable row opening its page; the
// rows past the section's Top fold behind the fold's own words.
struct HomeRowsCard<Extra: View>: View {
    let section: Section
    let actions: [Action]
    let open: Bool
    let lit: String?
    let fold: () -> Void
    let run: (Action, Row?) -> Void
    let go: (String) -> Void
    let extra: Extra

    init(_ section: Section, actions: [Action], open: Bool = false, lit: String? = nil, fold: @escaping () -> Void = {},
         run: @escaping (Action, Row?) -> Void, go: @escaping (String) -> Void, @ViewBuilder extra: () -> Extra) {
        self.section = section
        self.actions = actions
        self.open = open
        self.lit = lit
        self.fold = fold
        self.run = run
        self.go = go
        self.extra = extra()
    }

    var rows: [Row] { NeedsCard.shown(section, open: open).filter { $0.id != HomeCards.addMac }.map(HomeCards.quiet) }

    var body: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
            CardLabel(section.title ?? "", value: section.caption ?? [])
            Card {
                RowStack(rows, columns: section.columns ?? [], actions: actions, highlighted: lit, open: { row in
                    if let nav = row.nav { go(nav) }
                }) { action in
                    run(action, rows.first { $0.verb == action.id || $0.quick == action.id })
                }
                if let rest = section.rest?.row {
                    FoldLine(open ? rest.detail?.text ?? "" : HomeCards.folded(rest), open: open, tap: fold)
                }
                extra
            }
        }
    }
}

public enum HomeCards {
    public static let addMac = "add-mac"
    public static let tiles = "tiles"
    public static let agents = "agents"
    public static let servers = "servers"
    public static let macs = "macs"

    // value: a finished agent's row as Home draws it, one quiet line: its
    // name, then how it ended.
    public static func quiet(_ row: Row) -> Row {
        guard row.signal == "done", let detail = row.detail, !detail.isNone else { return row }
        var r = row
        r.label = [row.label ?? "", Format.show(detail)].joined(separator: Token.Glyph.sep)
        r.detail = nil
        return r
    }

    // value: a fold's closed words: how many it hides, then their names.
    public static func folded(_ rest: Row) -> String {
        ([rest.label ?? ""] + (rest.cells ?? []).map { Format.show($0) }).filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
    }

    // value: the rows of Home's cards the arrows move through, as drawn:
    // the navigable rows, a fold's hidden rows left out.
    public static func navigable(_ page: Page, unfolded: Set<String>) -> [Row] {
        [agents, servers, macs].compactMap { page.section($0) }.flatMap { s in
            NeedsCard.shown(s, open: unfolded.contains(s.id ?? "")).filter { $0.nav != nil }
        }
    }
}

// invariant: SPEC A4 Home 3: the agents, folded past three live ones.
public struct AgentsCard: View {
    let section: Section
    let actions: [Action]
    let open: Bool
    let lit: String?
    let fold: () -> Void
    let run: (Action, Row?) -> Void
    let go: (String) -> Void

    public init(_ section: Section, actions: [Action] = [], open: Bool = false, lit: String? = nil, fold: @escaping () -> Void = {},
                run: @escaping (Action, Row?) -> Void = { _, _ in }, go: @escaping (String) -> Void = { _ in }) {
        self.section = section
        self.actions = actions
        self.open = open
        self.lit = lit
        self.fold = fold
        self.run = run
        self.go = go
    }

    public var body: some View {
        HomeRowsCard(section, actions: actions, open: open, lit: lit, fold: fold, run: run, go: go) { EmptyView() }
    }
}

// invariant: SPEC A4 Home 4: one row per server, its dot and its words.
public struct ServersCard: View {
    let section: Section
    let actions: [Action]
    let lit: String?
    let go: (String) -> Void

    public init(_ section: Section, actions: [Action] = [], lit: String? = nil, go: @escaping (String) -> Void = { _ in }) {
        self.section = section
        self.actions = actions
        self.lit = lit
        self.go = go
    }

    public var body: some View {
        HomeRowsCard(section, actions: actions, lit: lit, run: { _, _ in }, go: go) { EmptyView() }
    }
}

// invariant: SPEC A4 Home 5: one row per Mac, then, for a fleet of one,
// the accent line that adds another.
public struct MacsCard: View {
    let section: Section
    let actions: [Action]
    let lit: String?
    let run: (Action, Row?) -> Void
    let go: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ section: Section, actions: [Action] = [], lit: String? = nil, run: @escaping (Action, Row?) -> Void = { _, _ in }, go: @escaping (String) -> Void = { _ in }) {
        self.section = section
        self.actions = actions
        self.lit = lit
        self.run = run
        self.go = go
    }

    var add: Row? { section.rows.first { $0.id == HomeCards.addMac } }

    public var body: some View {
        HomeRowsCard(section, actions: actions, lit: lit, run: run, go: go) {
            if let add, let action = RowView.action(add.verb, in: actions) {
                Button { run(action, add) } label: {
                    HStack(spacing: Token.Metric.Row.gap) {
                        Image(systemName: Token.Symbol.plus)
                            .font(.system(size: Token.Metric.Row.chevron, weight: .semibold))
                            .frame(width: Token.Metric.Row.accessory)
                        Text(add.label ?? "").font(Token.Role.rowName.font).lineLimit(1)
                        Spacer(minLength: 0)
                    }
                    .foregroundStyle(tokens.color.accentText)
                    .padding(.horizontal, Token.Metric.Row.padding)
                    .frame(minHeight: Token.HomeCard.line)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .overlay(alignment: .top) {
                    Rectangle().fill(tokens.color.hair).frame(height: Token.Metric.Row.hairline)
                        .padding(.leading, RowView.inset(.glyph)).padding(.trailing, Token.Metric.Row.insetEnd)
                }
            }
        }
    }
}

// invariant: a card's fold on Home: its chevron in the accessory column,
// its words where the names start ("3 more", "Show less"), the chevron
// turned while it is open.
struct FoldLine: View {
    let label: String
    let open: Bool
    let tap: () -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    init(_ label: String, open: Bool, tap: @escaping () -> Void) {
        self.label = label
        self.open = open
        self.tap = tap
    }

    var body: some View {
        let m = Token.Metric.Row.self
        Button(action: tap) {
            HStack(spacing: m.gap) {
                Image(systemName: Token.Symbol.down)
                    .font(.system(size: Token.Metric.Fold.chevron, weight: .semibold))
                    .rotationEffect(open ? Token.Metric.Fold.turned : .zero)
                    .animation(tokens.motion(Token.Motion.smooth), value: open)
                    .frame(width: m.accessory)
                Text(label).font(Token.Role.rowName.font).lineLimit(1)
                Spacer(minLength: 0)
            }
            .foregroundStyle(hovering ? tokens.color.t1 : tokens.color.t2)
            .padding(.horizontal, m.padding)
            .frame(minHeight: Token.HomeCard.line)
            .background(RoundedRectangle(cornerRadius: m.radius, style: .continuous).fill(hovering ? tokens.color.q : Color.clear))
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .onHover { hovering = $0 }
        .animation(tokens.motion(Token.Motion.tint), value: hovering)
        .accessibilityValue(open ? "open" : "closed")
    }
}
