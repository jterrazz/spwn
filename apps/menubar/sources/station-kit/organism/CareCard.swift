import SwiftUI

// invariant: SPEC A4's care card: the Mac's Awake with its switch, then
// its upkeep's one line, which unfolds in place to every item and its
// verb.
public struct CareCard: View {
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

    var awake: Row? { section.rows.first }
    var upkeep: Group? { (section.items ?? []).compactMap(\.group).first }

    public var body: some View {
        let m = Token.Metric.Row.self
        Card {
            if let awake { awakeRow(awake) }
            if let upkeep {
                if awake != nil { hairline(m.insetGlyph) }
                upkeepRow(upkeep)
                if open {
                    hairline(m.insetGlyph)
                    RowStack((upkeep.items ?? []).compactMap(\.row), columns: section.columns ?? [], actions: actions, run: run)
                }
            }
        }
    }

    func hairline(_ inset: CGFloat) -> some View {
        Rectangle().fill(tokens.color.hair)
            .frame(height: Token.Metric.Row.hairline)
            .padding(.leading, inset)
            .padding(.trailing, Token.Metric.Row.insetEnd)
    }

    func awakeRow(_ row: Row) -> some View {
        let m = Token.Metric.Row.self
        let toggle = RowView.action(row.verb, in: actions)
        return HStack(spacing: m.gap) {
            if let glyph = row.glyph { GlyphView(glyph).frame(width: m.accessory) }
            VStack(alignment: .leading, spacing: m.subGap) {
                Text(row.label ?? "").font(Token.Role.rowName.font).foregroundStyle(tokens.tone(Token.Role.rowName.tone)).lineLimit(1)
                if let sub = row.detail, !sub.isNone { FactsLine(sub, role: .rowSub) }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            Switch(on: row.cells?.first?.kind == "on") { if let toggle { run(toggle) } }
                .padding(.trailing, Token.Metric.Care.switchTrail)
                .accessibilityLabel(toggle?.label ?? row.label ?? "")
        }
        .padding(.horizontal, m.padding)
        .frame(minHeight: m.twoLines)
    }

    func upkeepRow(_ group: Group) -> some View {
        let m = Token.Metric.Row.self
        let head = group.head ?? Row()
        return Button(action: fold) {
            HStack(spacing: m.gap) {
                if let glyph = head.glyph { GlyphView(glyph).frame(width: m.accessory) }
                Text(head.label ?? "").font(Token.Role.rowName.font).foregroundStyle(tokens.tone(Token.Role.rowName.tone))
                Spacer(minLength: 0)
                if let value = head.cells?.first { LiveText(value, role: .rowValue).fixedSize() }
                Image(systemName: Token.Symbol.fold)
                    .font(.system(size: m.chevron, weight: .semibold))
                    .foregroundStyle(tokens.color.t3)
                    .rotationEffect(open ? Token.Metric.Fold.turned : .zero)
                    .animation(tokens.motion(Token.Motion.smooth), value: open)
            }
            .lineLimit(1)
            .padding(.horizontal, m.padding)
            .frame(minHeight: m.height)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(head.label ?? "")
        .accessibilityValue(open ? "open" : "closed")
    }
}
