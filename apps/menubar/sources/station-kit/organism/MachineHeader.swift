import SwiftUI

// invariant: SPEC A4: a machine page's header: the way back, the
// machine's glyph and name, its buttons (New VM, its menu), its facts with
// the stale marking after them, a fact that links pushing its page, and
// its segments.
public struct MachineHeader: View {
    let page: Page
    let segment: String?
    let pick: (String) -> Void
    let back: () -> Void
    let run: (Action) -> Void
    let go: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ page: Page, segment: String?, pick: @escaping (String) -> Void = { _ in }, back: @escaping () -> Void = {}, run: @escaping (Action) -> Void = { _ in }, go: @escaping (String) -> Void = { _ in }) {
        self.page = page
        self.segment = segment
        self.pick = pick
        self.back = back
        self.run = run
        self.go = go
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
                if let glyph = page.header?.glyph, case let .symbol(name) = GlyphView.drawn(glyph) {
                    Image(systemName: name).font(.system(size: m.glyph, weight: .regular))
                        .foregroundStyle(tokens.color.t2)
                        .frame(width: m.glyphSlot)
                }
                Text(summary?.title ?? "").font(Token.Role.pageTitle.font).foregroundStyle(tokens.tone(Token.Role.pageTitle.tone)).lineLimit(1)
                Spacer(minLength: 0)
                HStack(spacing: m.buttonsGap) {
                    ForEach(Array((page.header?.buttons ?? []).compactMap { page.action($0) }.enumerated()), id: \.offset) { _, a in
                        if a.icon == Token.Symbol.menu {
                            IconButton(Token.Symbol.menu) { run(a) }.accessibilityLabel(a.label ?? "")
                        } else {
                            PanelButton(a.label ?? "", symbol: a.icon) { run(a) }
                        }
                    }
                }
            }
            .frame(height: m.title)
            FactsLine(Self.facts(page)).padding(.top, m.factsTop)
                .tint(tokens.color.accentText)
                .environment(\.openURL, OpenURLAction { url in
                    guard let nav = FactsLine.nav(url) else { return .systemAction }
                    go(nav)
                    return .handled
                })
            if let segments = page.segments, !segments.isEmpty {
                Segmented(options: segmented(segments))
                .padding(.top, m.segmentsTop)
            }
        }
    }

    func segmented(_ segments: [Segment]) -> [(label: String, on: Bool, tap: () -> Void)] {
        segments.map { s in (label: s.label ?? "", on: s.id == segment, tap: { pick(s.id ?? "") }) }
    }
}
