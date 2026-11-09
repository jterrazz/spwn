import SwiftUI

// invariant: SPEC A4 Ports: one section of ports by its kind: the
// exposed card in red with its verbs, the HTTPS links switch, or a list
// under its label, its info button and its count (Public, Private,
// Listening, not shared), its rows the port rows, a group's head its grey
// words, nothing at all one line.
public struct PortsSection: View {
    let section: Section
    let actions: [Action]
    let run: (Action) -> Void
    let qr: (String) -> Void
    let about: (About) -> Void
    let shown: String?
    let opened: Bool
    @Environment(\.stationTokens) private var tokens

    public init(_ section: Section, actions: [Action] = [], run: @escaping (Action) -> Void = { _ in }, qr: @escaping (String) -> Void = { _ in },
                about: @escaping (About) -> Void = { _ in }, shown: String? = nil, opened: Bool = false) {
        self.section = section
        self.actions = actions
        self.run = run
        self.qr = qr
        self.about = about
        self.shown = shown
        self.opened = opened
    }

    // value: the link a row unfolds under it: the one its QR shows, while
    // the panel shows that QR.
    public static func unfolds(_ row: Row, shown: String?, in actions: [Action]) -> URL? {
        guard let shown, let quick = RowView.action(row.quick, in: actions), qr(quick, of: row, in: actions) == shown else { return nil }
        return URL(string: shown)
    }

    func port(_ row: Row) -> some View {
        let url = Self.unfolds(row, shown: shown, in: actions)
        return PortRow(row, actions: actions, highlighted: url != nil, run: tap(row))
            .popoverPin(url != nil, side: .above, width: Token.Metric.Popover.widthQR) {
                if let url { QRPopover(url: url, row: row, actions: actions, run: popped(url)) }
            }
    }

    func popped(_ url: URL) -> (Action) -> Void {
        { action in
            qr(url.absoluteString)
            run(action)
        }
    }

    // value: what a row's tap does: its verb's run, or, for the QR beside
    // a link, the QR of the link its row opens.
    public static func qr(_ action: Action, of row: Row, in actions: [Action]) -> String? {
        guard action.run == nil, action.id == row.quick, let link = PortRow.link(row, in: actions) else { return nil }
        return link.run?.open ?? link.run?.copy
    }

    func tap(_ row: Row) -> (Action) -> Void {
        { action in
            if let url = Self.qr(action, of: row, in: actions) { qr(url) } else { run(action) }
        }
    }

    public var body: some View {
        if section.id == "https" {
            Card { ForEach(Array(section.rows.enumerated()), id: \.offset) { _, row in switchRow(row) } }
        } else if section.card == "exposed" {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
                exposedLabel
                ForEach(Array(section.rows.enumerated()), id: \.offset) { _, row in
                    Card(flush: false) { exposed(row) }
                }
            }
        } else {
            VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
                header
                Card { list(section.items ?? []) }
            }
        }
    }

    // value: a list's label line: its words, its info button, its count.
    public var header: CardLabel { CardLabel(Self.label(section), value: section.caption ?? [], about: section.about, open: opened, tap: about) }

    // value: a list's label: its words, joined, or its title.
    public static func label(_ section: Section) -> String {
        let words = (section.glance ?? []).compactMap(\.text)
        return words.isEmpty ? section.title ?? "" : words.joined(separator: Token.Glyph.sep)
    }

    // value: an exposed port's title: its number, then the app that listens.
    public static func exposedTitle(_ row: Row) -> String {
        ([row.label ?? ""] + (row.cells ?? []).compactMap(\.text)).filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
    }

    var exposedLabel: some View {
        let red = tokens.signalText("failing")
        return HStack(alignment: .firstTextBaseline) {
            Text(section.title ?? "").font(Token.Role.sectionLabel.font).foregroundStyle(red)
            Spacer(minLength: 0)
            Text((section.caption ?? []).compactMap(\.text).joined(separator: Token.Glyph.sep)).font(Token.Role.sectionValue.font).foregroundStyle(red)
        }
        .lineLimit(1)
        .frame(height: Token.Metric.CardLabel.height)
        .padding(.horizontal, Token.Metric.CardLabel.inset)
    }

    func exposed(_ row: Row) -> some View {
        let m = Token.Metric.Ports.self
        return VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .center, spacing: m.exposedGap) {
                if let glyph = row.glyph { GlyphView(glyph) }
                VStack(alignment: .leading, spacing: Token.Metric.Row.subGap) {
                    Text(Self.exposedTitle(row)).font(Token.Role.exposedTitle.font).foregroundStyle(tokens.tone(Token.Role.exposedTitle.tone)).lineLimit(1)
                    if let sub = row.detail, !sub.isNone { FactsLine(sub, role: .rowSub) }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            HStack(spacing: m.exposedButtonsGap) {
                ForEach(Array((row.menu ?? []).compactMap { RowView.action($0, in: actions) }.enumerated()), id: \.offset) { _, a in
                    PanelButton((a.label ?? "") + (a.form != nil ? Token.Glyph.ellipsis : "")) { run(a) }
                }
            }
            .padding(.leading, m.exposedButtons)
            .padding(.top, m.exposedButtonsTop)
        }
    }

    func switchRow(_ row: Row) -> some View {
        let m = Token.Metric.Row.self
        let toggle = RowView.action(row.verb, in: actions)
        return HStack(spacing: m.gap) {
            if let glyph = row.glyph { GlyphView(glyph).frame(width: m.accessory) }
            Text(row.label ?? "").font(Token.Role.rowName.font).foregroundStyle(tokens.tone(Token.Role.rowName.tone)).lineLimit(1)
                .frame(maxWidth: .infinity, alignment: .leading)
            Switch(on: row.cells?.first?.kind == "on") { if let toggle { run(toggle) } }
                .padding(.trailing, Token.Metric.Care.switchTrail)
                .accessibilityLabel(toggle?.label ?? row.label ?? "")
        }
        .padding(.horizontal, m.padding)
        .frame(minHeight: m.height)
    }

    // value: a list's lines in order: each port row under its inset
    // hairline but the first of a run, a group's head and its rows, an
    // empty line.
    func list(_ items: [Item]) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(items.enumerated()), id: \.offset) { i, item in
                if let row = item.row {
                    port(row).overlay(alignment: .top) { if i > 0, items[i - 1].row != nil { hairline } }
                } else if let group = item.group {
                    if i > 0 { hairline }
                    Text(group.head?.label ?? "").font(Token.Role.portsSubhead.font).foregroundStyle(tokens.tone(Token.Role.portsSubhead.tone))
                        .padding(Token.Metric.Ports.subhead)
                    rows(group.items ?? [])
                } else if let empty = item.empty {
                    emptyLine(empty)
                }
            }
        }
    }

    func rows(_ items: [Item]) -> some View {
        let rows = items.compactMap(\.row)
        return VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(rows.enumerated()), id: \.offset) { i, row in
                port(row).overlay(alignment: .top) { if i > 0 { hairline } }
            }
        }
    }

    func emptyLine(_ empty: Empty) -> some View {
        Text(empty.text ?? "").font(Token.Role.emptyGrey.font).foregroundStyle(tokens.tone(Token.Role.emptyGrey.tone))
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(Token.Metric.Ports.empty)
    }

    var hairline: some View {
        Rectangle().fill(tokens.color.hair)
            .frame(height: Token.Metric.Row.hairline)
            .padding(.leading, Token.Metric.Row.insetGlyph)
            .padding(.trailing, Token.Metric.Row.insetEnd)
    }
}
