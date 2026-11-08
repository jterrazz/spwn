import SwiftUI

// invariant: SPEC A4: a machine page's header: the way back, the
// machine's glyph and name, its buttons (New VM, its menu), its facts with
// the stale marking after them, and its segments.
public struct MachineHeader: View {
    let page: Page
    let segment: String?
    let pick: (String) -> Void
    let back: () -> Void
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still

    public init(_ page: Page, segment: String?, pick: @escaping (String) -> Void = { _ in }, back: @escaping () -> Void = {}, run: @escaping (Action) -> Void = { _ in }) {
        self.page = page
        self.segment = segment
        self.pick = pick
        self.back = back
        self.run = run
    }

    var summary: Section? { page.section("summary") }

    // value: the facts, then the stale marking, each part dimmed with it.
    public static func facts(_ page: Page) -> Value {
        var parts = page.section("summary")?.caption ?? []
        for part in (page.header?.stale?.items ?? page.header?.stale.map { [$0] } ?? []) {
            var p = part
            p.signal = "stale"
            parts.append(p)
        }
        var line = Value(kind: "list")
        line.items = parts
        return line
    }

    public var body: some View {
        let m = Token.Metric.PageHeader.self
        VStack(alignment: .leading, spacing: 0) {
            if let words = page.header?.back {
                Button(action: back) {
                    HStack(spacing: m.backGap) {
                        Image(systemName: Token.Symbol.back).font(.system(size: m.backChevron, weight: .semibold))
                        Text(words).font(Token.Role.back.font)
                    }
                    .foregroundStyle(tokens.tone(Token.Role.back.tone))
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .padding(.bottom, m.backBottom)
            }
            HStack(spacing: m.titleGap) {
                if let glyph = page.header?.glyph {
                    switch GlyphView.drawn(glyph) {
                    case let .symbol(name):
                        Image(systemName: name).font(.system(size: m.glyph, weight: .regular))
                            .foregroundStyle(tokens.color.t2)
                            .frame(width: m.glyphSlot)
                    case let .custom(name):
                        Image(name, bundle: .module).font(.system(size: m.glyph, weight: .regular))
                            .foregroundStyle(tokens.color.t2)
                            .frame(width: m.glyphSlot)
                    default: EmptyView()
                    }
                }
                Text(summary?.title ?? "").font(Token.Role.pageTitle.font).foregroundStyle(tokens.tone(Token.Role.pageTitle.tone)).lineLimit(1)
                Spacer(minLength: 0)
                HStack(spacing: m.buttonsGap) {
                    ForEach(Array((page.header?.buttons ?? []).compactMap { page.action($0) }.enumerated()), id: \.offset) { _, a in
                        if a.icon == Token.Symbol.menu {
                            menu(a)
                        } else {
                            PanelButton(a.label ?? "", symbol: a.icon) { run(a) }
                        }
                    }
                }
            }
            .frame(height: m.title)
            FactsLine(Self.facts(page)).padding(.top, m.factsTop)
            if let segments = page.segments, !segments.isEmpty {
                Segmented(options: segmented(segments))
                .padding(.top, m.segmentsTop)
            }
        }
    }

    // value: what a page's menu lists: each action of its submenu the page
    // carries, in order.
    public static func menu(_ page: Page, _ action: Action) -> [Action] {
        (action.submenu ?? []).compactMap { page.action($0) }
    }

    // value: the page's menu: its submenu's verbs on a click, each run as
    // its own; drawn still as the round button it opens from.
    @ViewBuilder
    func menu(_ a: Action) -> some View {
        let items = Self.menu(page, a)
        if still || items.isEmpty {
            IconButton(Token.Symbol.menu) { run(a) }.accessibilityLabel(a.label ?? "")
        } else {
            Menu {
                ForEach(Array(items.enumerated()), id: \.offset) { _, item in
                    Button((item.label ?? "") + (item.form != nil || item.confirm != nil ? Token.Glyph.ellipsis : "")) { run(item) }
                }
            } label: {
                Image(systemName: Token.Symbol.menu).font(.system(size: Token.Metric.Control.iconGlyph, weight: .regular))
                    .foregroundStyle(tokens.color.t2)
                    .frame(width: Token.Metric.Control.icon, height: Token.Metric.Control.icon)
                    .contentShape(Circle())
            }
            .menuStyle(.button)
            .buttonStyle(.plain)
            .menuIndicator(.hidden)
            .fixedSize()
            .accessibilityLabel(a.label ?? "")
        }
    }

    func segmented(_ segments: [Segment]) -> [(label: String, on: Bool, tap: () -> Void)] {
        segments.map { s in (label: s.label ?? "", on: s.id == segment, tap: { pick(s.id ?? "") }) }
    }
}
