import SwiftUI

// invariant: SPEC A4: one card of accounts under its label and count:
// what needs you, each with its verb (Sign in reads its busy words while
// Terminal runs the login); the copies shared, each with its time left,
// then the row that shares one more, or the line that says none is; what a Mac is
// signed in to; a footnote under the card.
public struct AccountsSection: View {
    let section: Section
    let actions: [Action]
    let busy: Set<String>
    let run: (Action, Row?) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ section: Section, actions: [Action] = [], busy: Set<String> = [], run: @escaping (Action, Row?) -> Void = { _, _ in }) {
        self.section = section
        self.actions = actions
        self.busy = busy
        self.run = run
    }

    var needs: Bool { (section.id ?? "").split(separator: ":").last.map(String.init) == TodoBadge.section }

    // value: the action a row adds one more with: a verb that opens the
    // share sheet.
    func adds(_ row: Row) -> Action? {
        guard let verb = RowView.action(row.verb, in: actions), verb.sheet == Self.shareSheet else { return nil }
        return verb
    }

    static let shareSheet = "share-account"

    public var body: some View {
        let items = section.items ?? []
        VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
            CardLabel(section.title ?? "", value: section.caption ?? [])
            Card {
                ForEach(Array(items.enumerated()), id: \.offset) { i, item in
                    if let row = item.row {
                        line(row).overlay(alignment: .top) { if i > 0, !needs { hairline(row) } }
                    } else if let empty = item.empty {
                        Text(empty.text ?? "").font(Token.Role.noneShared.font).foregroundStyle(tokens.tone(Token.Role.noneShared.tone))
                            .padding(.horizontal, Token.Metric.Row.padding)
                            .frame(maxWidth: .infinity, minHeight: Token.Metric.Row.height, alignment: .leading)
                    }
                }
            }
            ForEach(Array(section.notes.enumerated()), id: \.offset) { _, note in
                Text(note).font(Token.Role.footnote.font).foregroundStyle(tokens.tone(Token.Role.footnote.tone))
                    .lineSpacing(Token.Metric.Footnote.lineSpacing)
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(Token.Metric.Footnote.padding)
            }
        }
    }

    @ViewBuilder
    func line(_ row: Row) -> some View {
        if needs {
            NeedRow(row, actions: actions, busy: busy) { run($0, row) }
        } else if let add = adds(row) {
            AddRow((add.label ?? "") + Token.Glyph.ellipsis, trail: RowView.trail(row, columns: section.columns ?? []).map { Format.show($0) }) { run(add, row) }
        } else {
            AccountRow(row, actions: actions) { run($0, row) }
        }
    }

    func hairline(_ row: Row) -> some View {
        let m = Token.Metric.Row.self
        return Rectangle().fill(tokens.color.hair)
            .frame(height: m.hairline)
            .padding(.leading, m.insetGlyph)
            .padding(.trailing, m.insetEnd)
    }
}
