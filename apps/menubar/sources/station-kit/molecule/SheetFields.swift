import SwiftUI

// value: where a sheet's menu opens, for its preview: a popup field's,
// by its key, and the split button's arrow.
public enum SheetAnchor {
    public static func field(_ key: String) -> String { "field:" + key }
    public static let split = "split"
}

// value: a select with one choice: its value said, glyph, label and
// note, with nothing to open (Lend an account to the page's own VM).
struct FixedValue: View {
    let option: Option
    let answers: Answers
    @Environment(\.stationTokens) private var tokens

    var body: some View {
        HStack(spacing: S.popupGap) {
            if let glyph = option.glyph { OptionGlyph(glyph) }
            RoleText(option.label ?? "", .formValue)
            if let note = option.note { RoleText(answers.fill(note), .formNote) }
        }
        .frame(height: S.field)
    }
}

// value: an option's glyph at the size of the field's words.
struct OptionGlyph: View {
    let glyph: Glyph
    @Environment(\.stationTokens) private var tokens

    init(_ glyph: Glyph) { self.glyph = glyph }

    var body: some View {
        switch GlyphView.drawn(glyph) {
        case let .chip(kind, brand): LogoChip(kind, brand)
        case let .symbol(name):
            Image(systemName: name).font(.system(size: S.fixedGlyph, weight: .medium)).foregroundStyle(tokens.color.t2)
        default: GlyphView(glyph)
        }
    }
}

// value: a sheet's list ground: the quiet card its rows sit on, a hair
// between them.
struct SheetRows<Row: View>: View {
    let count: Int
    let row: (Int) -> Row
    @Environment(\.stationTokens) private var tokens

    init(count: Int, @ViewBuilder row: @escaping (Int) -> Row) {
        self.count = count
        self.row = row
    }

    var body: some View {
        VStack(spacing: 0) {
            ForEach(0..<count, id: \.self) { i in
                row(i).frame(minHeight: S.listRow)
                if i < count - 1 { Rectangle().fill(tokens.color.hair).frame(height: S.footerHair) }
            }
        }
        .padding(S.listPadding)
        .background(RoundedRectangle(cornerRadius: S.listRadius, style: .continuous).fill(tokens.color.q))
    }
}

// value: a row's words in a sheet's list: its name, its note under it.
struct ListWords: View {
    let label: String
    let note: String?
    let mono: Bool
    var wraps = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            RoleText(label, .listTitle)
            if let note, !note.isEmpty {
                Text(note).font(mono ? Token.Role.commandLine.font : Token.Role.listNote.font)
                    .foregroundStyle(Token.Ink.secondary).lineLimit(wraps ? nil : 1)
                    .fixedSize(horizontal: false, vertical: wraps)
            }
        }
    }
}

// value: a line a sheet says in small grey words, wrapped whole.
struct SheetLine: View {
    let text: String
    let role: Token.Role
    @Environment(\.stationTokens) private var tokens

    init(_ text: String, _ role: Token.Role = .sheetNote) {
        self.text = text
        self.role = role
    }

    var body: some View {
        Text(text).font(role.font).foregroundStyle(tokens.tone(role.tone)).fixedSize(horizontal: false, vertical: true)
    }
}

// invariant: SPEC A5 Lend an account: one answer picked among rows, each
// its logo, its name and its words, the picked one ticked; one it cannot
// take dimmed with its reason as its words.
public struct PickRows: View {
    let field: Field
    let answers: Answers
    let set: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ field: Field, _ answers: Answers, set: @escaping (String) -> Void) {
        self.field = field
        self.answers = answers
        self.set = set
    }

    public var body: some View {
        let options = field.options ?? []
        SheetRows(count: options.count) { i in
            let o = options[i]
            let on = answers.value(field) == o.value
            Button { set(o.value ?? "") } label: {
                HStack(spacing: S.listGap) {
                    if let glyph = o.glyph { OptionGlyph(glyph) }
                    ListWords(label: o.label ?? "", note: o.note.map(answers.fill), mono: false)
                    Spacer(minLength: 0)
                    if on { Image(systemName: Token.Symbol.check).font(.system(size: S.listCheck, weight: .semibold)).foregroundStyle(tokens.color.accent) }
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .disabled(o.disabled != nil)
            .opacity(o.disabled != nil ? Token.Fade.disabled : 1)
        }
    }
}

// invariant: SPEC A5 Share a port: what listens, each with Share, the
// action its option names; none, the line that says so.
public struct ListenerRows: View {
    let field: Field
    let verb: (String) -> String
    let run: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ field: Field, verb: @escaping (String) -> String, run: @escaping (String) -> Void) {
        self.field = field
        self.verb = verb
        self.run = run
    }

    public var body: some View {
        let options = field.options ?? []
        VStack(alignment: .leading, spacing: S.listLabelGap) {
            RoleText(field.label ?? "", .listLabel)
            if options.isEmpty {
                SheetLine(field.note ?? "", .listLabel)
            } else {
                SheetRows(count: options.count) { i in
                    let o = options[i]
                    HStack(spacing: S.listGap) {
                        if let glyph = o.glyph { OptionGlyph(glyph) }
                        ListWords(label: o.label ?? "", note: o.note, mono: true)
                        Spacer(minLength: 0)
                        PanelButton(verb(o.value ?? "")) { run(o.value ?? "") }.fixedSize()
                    }
                }
            }
        }
    }
}

// invariant: SPEC A5 Clean up: a group's label and why, then its rows
// each checked or not; a row its leader carries is checked and dimmed.
public struct CheckRows: View {
    let field: Field
    let answers: Answers
    let set: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ field: Field, _ answers: Answers, set: @escaping (String) -> Void) {
        self.field = field
        self.answers = answers
        self.set = set
    }

    func toggled(_ value: String) -> String {
        var now = answers.value(field).split(separator: ",").map(String.init)
        if let i = now.firstIndex(of: value) { now.remove(at: i) } else { now.append(value) }
        return (field.options ?? []).compactMap(\.value).filter(now.contains).joined(separator: ",")
    }

    public var body: some View {
        let options = field.options ?? []
        VStack(alignment: .leading, spacing: S.listLabelGap) {
            VStack(alignment: .leading, spacing: 0) {
                RoleText(field.label ?? "", .listLabel)
                if let note = field.note { SheetLine(note) }
            }
            SheetRows(count: options.count) { i in
                let o = options[i]
                let carried = answers.follows(field, o)
                let on = answers.picked(field).contains(o.value ?? "")
                Button { set(toggled(o.value ?? "")) } label: {
                    HStack(spacing: S.listGap) {
                        CheckBox(on: on)
                        ListWords(label: o.label ?? "", note: o.note.map(answers.fill), mono: false, wraps: true)
                        Spacer(minLength: 0)
                    }
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .disabled(carried || o.disabled != nil)
                .opacity(carried || o.disabled != nil ? Token.Fade.disabled : 1)
            }
        }
    }
}

struct CheckBox: View {
    let on: Bool
    @Environment(\.stationTokens) private var tokens

    var body: some View {
        ZStack {
            RoundedRectangle(cornerRadius: S.boxRadius, style: .continuous).fill(on ? tokens.color.accent : Color.clear)
            if !on { RoundedRectangle(cornerRadius: S.boxRadius, style: .continuous).strokeBorder(tokens.color.t3, lineWidth: S.boxStroke) }
            if on { Image(systemName: Token.Symbol.check).font(.system(size: S.boxMark, weight: .bold)).foregroundStyle(tokens.onTint) }
        }
        .frame(width: S.box, height: S.box)
    }
}

// invariant: SPEC A5 Hints: a command the owner runs, the page's own
// words, in its mono box with the round Copy beside it; the line that
// leads it above.
public struct CommandBox: View {
    let field: Field
    let copy: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ field: Field, copy: @escaping (String) -> Void) {
        self.field = field
        self.copy = copy
    }

    public var body: some View {
        let command = field.hint?.command ?? ""
        VStack(alignment: .leading, spacing: S.listLabelGap) {
            if let lead = field.note { RoleText(lead, .listLabel) }
            HStack(alignment: .center, spacing: S.listGap) {
                Text(verbatim: command).font(Token.Role.commandLine.font).foregroundStyle(tokens.tone(Token.Role.commandLine.tone))
                    .fixedSize(horizontal: false, vertical: true)
                    .frame(maxWidth: .infinity, alignment: .leading)
                Button { copy(command) } label: {
                    Image(systemName: Token.Symbol.copy).font(.system(size: S.commandGlyph, weight: .medium)).foregroundStyle(tokens.color.t2)
                        .frame(width: S.commandCopy, height: S.commandCopy)
                        .background(Circle().fill(tokens.color.qq))
                }
                .buttonStyle(.plain)
                .accessibilityLabel(command)
            }
            .padding(S.commandPadding)
            .background(RoundedRectangle(cornerRadius: S.commandRadius, style: .continuous).fill(tokens.color.q))
        }
    }
}

// value: a sheet's primary button as the answers word it: red once what
// it runs deletes, tinted otherwise, off while no wording holds.
struct SheetPrimary: View {
    let primary: Primary?
    let fallback: String
    let run: () -> Void

    var body: some View {
        PanelButton(primary?.label ?? fallback, look: primary?.danger == true ? .destructive : .tint, action: run)
            .keyboardShortcut(.defaultAction)
            .disabled(primary == nil)
            .fixedSize()
    }
}

// value: SPEC A5: the page under a sheet: blurred under the scrim, and out
// of reach.
public struct UnderSheet: ViewModifier {
    let on: Bool

    public init(_ on: Bool) { self.on = on }

    public func body(content: Content) -> some View {
        content.blur(radius: on ? S.blur : 0).disabled(on)
    }
}
