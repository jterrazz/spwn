import SwiftUI

public struct Layout {
    static func chars(_ v: Value?) -> Int { v.map { Format.show($0).count } ?? 0 }

    // invariant: a row that does not fit drops its cells by Keep, lowest
    // first, never its mark nor its label (G4's order); what the drops free
    // goes back to a dropped cell that fits in it, the most kept first; the
    // detail is cut at the end, so it claims only its first 20 characters, as
    // the tty's does.
    static let detailMin = 20
    public static func kept(_ row: Row, columns: [Column], width: CGFloat) -> [Int] {
        let cells = row.cells ?? []
        // invariant: a column for a row's detail only (`detail`, a disk row's
        // folders) is no cell of a line here either.
        let shown = cells.indices.filter { $0 + 1 >= columns.count || columns[$0 + 1].show != "detail" }
        var keep = shown
        let room = Int((width - Token.Measure.mark - Token.Space.gap) / Token.Measure.char)
        func need() -> Int {
            (row.short ?? row.label ?? "").count + keep.reduce(0) { $0 + chars(cells[$1]) + 2 } + min(chars(row.detail), detailMin) + 2
        }
        let order = shown.sorted { a, b in
            let ka = a + 1 < columns.count ? columns[a + 1].keep ?? 0 : 0
            let kb = b + 1 < columns.count ? columns[b + 1].keep ?? 0 : 0
            return ka == kb ? a > b : ka < kb
        }
        var dropped: [Int] = []
        for i in order where need() > room {
            keep.removeAll { $0 == i }
            dropped.append(i)
        }
        for i in dropped.reversed() {
            keep.append(i)
            if need() > room { keep.removeLast() }
        }
        return keep.sorted()
    }
}

public struct RowView: View {
    let row: Row
    let columns: [Column]
    let width: CGFloat
    let hintAsButton: Bool
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationHints) private var hints

    public init(_ row: Row, columns: [Column], width: CGFloat, hintAsButton: Bool = false) {
        self.row = row
        self.columns = columns
        self.width = width
        self.hintAsButton = hintAsButton
    }

    public var body: some View {
        let cells = row.cells ?? []
        VStack(alignment: .leading, spacing: Token.Gap.tight) {
            HStack(spacing: Token.Space.gap) {
                HStack(spacing: Token.Space.gap / 2) {
                    MarkView(row.mark)
                    Text(row.short ?? row.label ?? "")
                        .font(.system(size: Token.Space.text))
                        .foregroundStyle(palette.text)
                        .lineLimit(1)
                }
                .layoutPriority(1)
                ForEach(Layout.kept(row, columns: columns, width: width), id: \.self) { i in
                    ValueText(cells[i], muted: i > 0)
                }
                Spacer(minLength: Token.Space.gap)
                if let detail = row.detail, !detail.isNone {
                    ValueText(detail, muted: true).layoutPriority(-1)
                }
            }
            .frame(minHeight: Token.Space.row - Token.Gap.small)
            if hints, let hint = row.hint {
                HintView(hint, asButton: hintAsButton).padding(.leading, Token.Measure.mark + Token.Space.gap / 2)
            }
        }
    }
}

public struct FactView: View {
    let fact: Fact
    @Environment(\.stationPalette) private var palette

    public init(_ fact: Fact) { self.fact = fact }

    public var body: some View {
        HStack(spacing: Token.Space.gap) {
            Text(fact.key ?? "").font(.system(size: Token.Space.text)).foregroundStyle(palette.muted)
            ValueText(fact.value)
            Spacer(minLength: 0)
        }
    }
}

public struct HintView: View {
    let hint: Hint
    let asButton: Bool
    @Environment(\.stationPalette) private var palette

    public init(_ hint: Hint, asButton: Bool = false) {
        self.hint = hint
        self.asButton = asButton
    }

    public var body: some View {
        HStack(spacing: Token.Space.gap / 2) {
            Text("\(Token.Glyph.arrow) \(hint.command ?? "")")
                .font(.system(size: Token.Space.small, design: .monospaced))
                .foregroundStyle(palette.muted)
                .lineLimit(2)
            if let text = hint.text {
                Text(text).font(.system(size: Token.Space.small)).foregroundStyle(Token.Ink.tertiary)
            }
            if asButton {
                Button {
                    NSPasteboard.general.clearContents()
                    NSPasteboard.general.setString(hint.command ?? "", forType: .string)
                } label: {
                    Image(systemName: Token.Symbol.copy).font(.system(size: Token.Space.small))
                }
                .buttonStyle(.plain)
                .foregroundStyle(palette.muted)
            }
        }
    }
}

public struct EmptyLine: View {
    let empty: Empty
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationHints) private var hints

    public init(_ empty: Empty) { self.empty = empty }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Gap.tight) {
            Text(empty.text ?? "no \(empty.noun ?? "")").font(.system(size: Token.Space.text)).foregroundStyle(palette.muted)
            if hints, let hint = empty.hint { HintView(hint) }
        }
    }
}

public struct ErrorLine: View {
    let error: PageError
    @Environment(\.stationPalette) private var palette

    public init(_ error: PageError) { self.error = error }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Gap.tight) {
            HStack(spacing: Token.Space.gap / 2) {
                Image(systemName: Token.Symbol.error).font(.system(size: Token.Space.text)).foregroundStyle(palette.bad)
                Text(error.message ?? "").font(.system(size: Token.Space.text)).foregroundStyle(palette.text).fixedSize(horizontal: false, vertical: true)
            }
            if let hint = error.hint, !hint.isEmpty {
                Text("\(Token.Glyph.arrow) \(hint)").font(.system(size: Token.Space.small, design: .monospaced)).foregroundStyle(palette.muted)
            }
        }
    }
}

public struct CountsView: View {
    let counts: [Tally]
    @Environment(\.stationPalette) private var palette

    public init(_ counts: [Tally]) { self.counts = counts }

    public var body: some View {
        HStack(spacing: Token.Space.gap / 2) {
            ForEach(Array(counts.enumerated()), id: \.offset) { _, t in
                HStack(spacing: Token.Gap.tight) {
                    Text("\(t.n ?? 0)").font(.system(size: Token.Space.small)).monospacedDigit().foregroundStyle(palette.muted)
                    MarkView(t.mark)
                }
            }
        }
    }
}

public struct ConfirmPanel: View {
    let confirm: Confirm
    let onAnswer: (Bool) -> Void
    @Environment(\.stationPalette) private var palette

    public init(_ confirm: Confirm, onAnswer: @escaping (Bool) -> Void) {
        self.confirm = confirm
        self.onAnswer = onAnswer
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Gap.small) {
            Text(confirm.question ?? "").font(.system(size: Token.Space.text, weight: .semibold)).foregroundStyle(palette.text)
            ForEach(Array((confirm.lines ?? []).enumerated()), id: \.offset) { _, line in
                ValueText(line, muted: true)
            }
            HStack {
                Spacer()
                Button(confirm.no ?? "No") { onAnswer(false) }.keyboardShortcut(.defaultAction)
                Button(confirm.yes ?? "Yes") { onAnswer(true) }.foregroundStyle(confirm.danger == true ? palette.bad : palette.text)
            }
        }
    }
}

public struct FormView: View {
    let form: Form
    @Binding var answers: [String: String]
    @Environment(\.stationPalette) private var palette

    public init(_ form: Form, answers: Binding<[String: String]>) {
        self.form = form
        _answers = answers
    }

    func binding(_ f: Field) -> Binding<String> {
        Binding(get: { answers[f.key ?? ""] ?? f.default ?? "" }, set: { answers[f.key ?? ""] = $0 })
    }

    public var body: some View {
        HStack(spacing: Token.Space.gap) {
            ForEach(Array((form.fields ?? []).enumerated()), id: \.offset) { _, f in
                if f.kind == "confirm" {
                    Toggle(f.label ?? "", isOn: Binding(get: { binding(f).wrappedValue == "true" }, set: { binding(f).wrappedValue = $0 ? "true" : "false" }))
                        .toggleStyle(.checkbox)
                        .font(.system(size: Token.Space.small))
                } else {
                    Picker(f.label ?? "", selection: binding(f)) {
                        ForEach(Array((f.options ?? []).enumerated()), id: \.offset) { _, o in
                            Text(o.label ?? o.value ?? "").tag(o.value ?? "")
                        }
                    }
                    .pickerStyle(.menu)
                    .font(.system(size: Token.Space.small))
                    .fixedSize()
                }
            }
        }
    }
}
