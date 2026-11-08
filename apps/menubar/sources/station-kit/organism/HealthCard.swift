import SwiftUI

// invariant: SPEC A4's Health card of a server: one head (its mark, its
// words, the chevron) that unfolds in place to the four checks of every
// boot, one 16 pt slot each in one text column (SPEC A9 3-9), then its
// verbs; a restart's card is open by itself; a down server's is the red
// banner with Restart, a stopped one's says so with Start.
public struct HealthCard: View {
    let section: Section
    let actions: [Action]
    let open: Bool
    let fold: () -> Void
    let run: (Action) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ section: Section, actions: [Action] = [], open: Bool = false, fold: @escaping () -> Void = {}, run: @escaping (Action) -> Void = { _ in }) {
        self.section = section
        self.actions = actions
        self.open = open
        self.fold = fold
        self.run = run
    }

    var group: Group? { (section.items ?? []).compactMap(\.group).first }
    var head: Row { group?.head ?? Row() }
    var checks: [Row] { (group?.items ?? []).compactMap(\.row) }
    var folds: Bool { group?.fold == "folded" || group?.fold == "open" }

    // invariant: a card the page opens (a restart) stays open; a folded
    // one opens on its head.
    public static func shown(_ group: Group?, open: Bool) -> Bool {
        group?.fold == "open" || (group?.fold == "folded" && open)
    }

    var unfolded: Bool { Self.shown(group, open: open) }

    public var body: some View {
        let m = Token.Metric.Health.self
        Card(flush: false) {
            if let verb = RowView.action(head.verb, in: actions) {
                banner(verb)
            } else {
                Button(action: fold) { headLine }
                    .buttonStyle(.plain)
                    .allowsHitTesting(group?.fold == "folded")
                    .accessibilityValue(unfolded ? "open" : "closed")
                if unfolded {
                    VStack(alignment: .leading, spacing: 0) {
                        ForEach(Array(checks.enumerated()), id: \.offset) { _, row in CheckRow(row) }
                        if !(head.menu ?? []).isEmpty { verbs.padding(.top, m.buttonsTop) }
                    }
                    .padding(.leading, m.checksIndent)
                    .padding(.top, m.checksTop)
                }
            }
        }
    }

    @ViewBuilder var mark: some View {
        let m = Token.Metric.Health.self
        let glyph = head.glyph ?? Glyph()
        SwiftUI.Group {
            if glyph.spin == true {
                Spinner()
            } else if glyph.kind == nil, let dot = glyph.dot {
                StatusDot(dot, signal: glyph.signal, size: Token.Metric.Marks.check - Token.Metric.Marks.ringStroke * 2)
            } else if case let .symbol(name) = GlyphView.drawn(glyph) {
                Image(systemName: name)
                    .font(.system(size: Token.Metric.Marks.check))
                    .symbolRenderingMode(.palette)
                    .foregroundStyle(tokens.onTint, tokens.signal(glyph.signal))
            }
        }
        .frame(width: m.mark, height: m.mark)
    }

    var words: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Health.headLines) {
            Text(head.label ?? "").font(Token.Role.healthTitle.font).foregroundStyle(tokens.tone(Token.Role.healthTitle.tone)).lineLimit(1)
            if let sub = head.detail, !sub.isNone { FactsLine(sub, role: .healthSub) }
        }
    }

    var headLine: some View {
        HStack(spacing: Token.Metric.Health.headGap) {
            mark
            words.frame(maxWidth: .infinity, alignment: .leading)
            if folds, group?.fold != "open" || unfolded {
                Image(systemName: Token.Symbol.fold)
                    .font(.system(size: Token.Metric.Row.chevron, weight: .semibold))
                    .foregroundStyle(tokens.color.t3)
                    .rotationEffect(unfolded ? Token.Metric.Fold.turned : .zero)
                    .animation(tokens.motion(Token.Motion.smooth), value: unfolded)
            }
        }
        .contentShape(Rectangle())
    }

    func banner(_ verb: Action) -> some View {
        let red = head.mark == .broken
        return HStack(spacing: Token.Metric.Health.headGap) {
            mark
            VStack(alignment: .leading, spacing: Token.Metric.Health.headLines) {
                Text(head.label ?? "").font((red ? Token.Role.bannerTitle : Token.Role.healthTitle).font).foregroundStyle(tokens.color.t1).lineLimit(1)
                if let sub = head.detail, !sub.isNone { FactsLine(sub, role: .healthSub) }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            PanelButton(verb.label ?? "", look: red ? .redTint : .plain) { run(verb) }
        }
        .frame(minHeight: Token.Metric.Health.banner)
    }

    var verbs: some View {
        HStack(spacing: Token.Metric.Health.buttonsGap) {
            ForEach(Array((head.menu ?? []).compactMap { RowView.action($0, in: actions) }.enumerated()), id: \.offset) { _, a in
                PanelButton((a.label ?? "") + (a.confirm != nil ? Token.Glyph.ellipsis : "")) { run(a) }
            }
            Spacer(minLength: 0)
        }
    }
}
