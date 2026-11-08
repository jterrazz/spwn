import SwiftUI

// invariant: SPEC A4 Home 1: the hero is one card the engine decides: its
// glyph, its title, its facts fitted by their priority, its one verb (red
// tinted for a red need), then the needs it lists, each with its verb, and
// the link to the rest. A Mac still answering spins beside the count. The
// empty fleet is the empty state with New VM.
public struct Hero: View {
    let section: Section
    let actions: [Action]
    let stale: Value?
    let run: (Action) -> Void
    let open: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ section: Section, actions: [Action] = [], stale: Value? = nil, run: @escaping (Action) -> Void = { _ in }, open: @escaping (String) -> Void = { _ in }) {
        self.section = section
        self.actions = actions
        self.stale = stale
        self.run = run
        self.open = open
    }

    var items: [Item] { section.items ?? [] }
    var head: Row? { items.first?.row }
    var rest: [Item] { Array(items.dropFirst()) }
    var facts: Value? { head?.detail.flatMap { $0.isNone ? nil : Self.checking($0) } }

    // value: a Mac still being checked reads on with an ellipsis.
    static func checking(_ v: Value) -> Value {
        guard v.signal == "stale", let text = v.text else { return v }
        var out = v
        out.text = text + Token.Glyph.ellipsis
        return out
    }
    var verb: Action? { RowView.action(head?.verb, in: actions) }

    // value: what VoiceOver says when the hero changes: its title, then its
    // facts as the line says them all.
    public static func announcement(_ section: Section?) -> String? {
        guard let section, let first = section.items?.first else { return nil }
        if let empty = first.empty { return [empty.text, empty.sub].compactMap { $0 }.joined(separator: Token.Glyph.sep) }
        guard let head = first.row else { return nil }
        let facts = head.detail.map { FactsLine.words($0.items ?? [$0]) } ?? ""
        return [title(head), facts].filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
    }

    // value: a title of work going on reads on with an ellipsis.
    static func title(_ head: Row) -> String {
        (head.label ?? "") + (head.glyph?.spin == true ? Token.Glyph.ellipsis : "")
    }

    public var body: some View {
        if let empty = items.first?.empty {
            EmptyState(empty, actions: actions, run: run)
                .modifier(HeroCard())
        } else if let head {
            VStack(alignment: .leading, spacing: 0) {
                top(head)
                if !rest.isEmpty { rows }
            }
            .modifier(HeroCard())
            .accessibilityElement(children: .contain)
        }
    }

    func top(_ head: Row) -> some View {
        let m = Token.Metric.Hero.self
        let p = Token.HeroParts.self
        return HStack(spacing: Token.Metric.Row.gap) {
            glyph(head).frame(width: m.slot.width, height: m.slot.height)
            VStack(alignment: .leading, spacing: 0) {
                HStack(spacing: p.titleGap) {
                    Text(Self.title(head))
                        .font(Token.Role.heroTitle.font)
                        .tracking(Token.Role.heroTitle.tracking)
                        .foregroundStyle(tokens.tone(Token.Role.heroTitle.tone))
                        .lineLimit(p.titleLines)
                        .fixedSize(horizontal: false, vertical: true)
                    if let stale, facts == nil {
                        Spinner(small: true).accessibilityLabel(Text(Format.show(stale) + Token.Glyph.ellipsis))
                    }
                }
                if let facts {
                    FactsLine(facts, role: .facts).padding(.top, p.factsTop)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityElement(children: .combine)
            .accessibilityAddTraits(.isHeader)
            if let verb {
                if head.mark == .broken {
                    PanelButton(verb.label ?? "", look: .redTint) { run(verb) }
                } else {
                    VerbButton(verb.label ?? "") { run(verb) }
                }
            }
        }
        .frame(minHeight: facts == nil && verb == nil ? m.topTitleOnly : m.top)
    }

    @ViewBuilder
    func glyph(_ head: Row) -> some View {
        let p = Token.HeroParts.self
        if head.glyph?.spin == true {
            Spinner(size: p.spinner, stroke: p.spinnerStroke)
        } else if let glyph = head.glyph, case let .symbol(name) = GlyphView.drawn(glyph) {
            Image(systemName: name)
                .font(.system(size: Token.Metric.Hero.glyph, weight: .medium))
                .foregroundStyle(tokens.signal(glyph.signal))
                .accessibilityHidden(true)
        }
    }

    var rows: some View {
        let p = Token.HeroParts.self
        return VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(rest.enumerated()), id: \.offset) { _, item in
                if let row = item.row {
                    NeedRow(row, actions: actions, run: run)
                } else if let link = item.link {
                    HeroLink(link) { open(link.page ?? "") }
                }
            }
        }
        .padding(.top, p.rowsPad)
        .overlay(alignment: .top) { Rectangle().fill(tokens.color.hair).frame(height: Token.Metric.Row.hairline) }
        .padding(.horizontal, -p.rowsBleed)
        .padding(.top, p.rowsTop)
        .padding(.bottom, -p.rowsBottom)
    }
}

// invariant: the hero's link to the rest: alone, the tinted "N more need
// you ›"; under a red or busy hero, the line that folds the other needs,
// its orange mark, its count, then the subjects it holds.
struct HeroLink: View {
    let link: Link
    let tap: () -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    init(_ link: Link, tap: @escaping () -> Void) {
        self.link = link
        self.tap = tap
    }

    var names: String? { link.detail.flatMap { $0.isNone ? nil : Format.show($0) } }

    var body: some View {
        let m = Token.Metric.Need.self
        Button(action: tap) {
            HStack(spacing: Token.Metric.Row.gap) {
                if let names {
                    Image(systemName: Token.HeroParts.alsoMark)
                        .font(.system(size: m.mark, weight: .medium))
                        .foregroundStyle(tokens.color.orange)
                        .frame(width: Token.Metric.Row.accessory)
                    (Text(verbatim: link.label ?? "").font(Token.Role.needSubject.font)
                        + Text(verbatim: Token.Glyph.sep + names).font(Token.Role.needWhat.font).foregroundStyle(tokens.color.t2))
                        .foregroundStyle(tokens.color.t1)
                        .lineLimit(1)
                        .frame(maxWidth: .infinity, alignment: .leading)
                } else {
                    Text(verbatim: link.label ?? "").font(Token.Role.needWhat.font).foregroundStyle(tokens.color.accentText).lineLimit(1)
                }
                Image(systemName: Token.HeroParts.more)
                    .font(.system(size: Token.Metric.Row.chevron, weight: .semibold))
                    .foregroundStyle(names == nil ? tokens.color.accentText : tokens.color.t3)
                if names == nil { Spacer(minLength: 0) }
            }
            .modifier(RowSurface(height: Token.Metric.Hero.more, padding: m.padding, tinted: hovering))
        }
        .buttonStyle(.plain)
        .onHover { hovering = $0 }
    }
}

struct HeroCard: ViewModifier {
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        let shape = RoundedRectangle(cornerRadius: Token.Metric.Hero.radius, style: .continuous)
        content
            .padding(Token.Metric.Hero.padding)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(shape.fill(tokens.color.hero))
            .overlay { if tokens.increased { shape.strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Card.contrastStroke) } }
            .clipShape(shape)
    }
}
