import SwiftUI

// invariant: a field the owner types into: the text so far and a caret when
// drawn still (an offscreen snapshot), a live text field in the app.
public struct TypedField: View {
    let placeholder: String
    let symbol: String?
    let key: String?
    let autofocus: Bool
    let move: ((Int) -> Void)?
    let submit: (() -> Void)?
    @Binding var text: String
    @FocusState private var focused: Bool
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationStill) private var still
    @Environment(\.colorScheme) private var scheme

    // value: a field that takes the keys as it opens (the switcher's
    // search), its Up and Down moving through what it finds, its Return
    // opening it.
    public init(_ placeholder: String, text: Binding<String>, symbol: String? = nil, key: String? = nil, autofocus: Bool = false,
                move: ((Int) -> Void)? = nil, submit: (() -> Void)? = nil) {
        self.placeholder = placeholder
        self.symbol = symbol
        self.key = key
        self.autofocus = autofocus
        self.move = move
        self.submit = submit
        _text = text
    }

    public var body: some View {
        HStack(spacing: Token.Gap.pair) {
            if let symbol { Image(systemName: symbol).font(.system(size: Token.Size.text, weight: .medium)).foregroundStyle(palette.glyph) }
            field
            if let key {
                Text(key).font(.system(size: Token.Size.key)).foregroundStyle(Token.Ink.secondary)
            }
        }
        .font(.system(size: Token.Size.text))
        .padding(.horizontal, Token.Gap.block).padding(.vertical, Token.Gap.row)
        .stationField()
    }

    @ViewBuilder
    var field: some View {
        SwiftUI.Group {
            if still {
                HStack(spacing: Token.Gap.tight) {
                    if text.isEmpty { Rectangle().fill(palette.accent).frame(width: Token.Stroke.caret, height: Token.Detail.caretHeight) }
                    Text(text.isEmpty ? placeholder : text).foregroundStyle(text.isEmpty ? AnyShapeStyle(Token.Ink.secondary) : AnyShapeStyle(Token.Ink.primary))
                    if !text.isEmpty { Rectangle().fill(palette.accent).frame(width: Token.Stroke.caret, height: Token.Detail.caretHeight) }
                    Spacer(minLength: 0)
                }
            } else {
                TextField(placeholder, text: $text).textFieldStyle(.plain)
                    .focused($focused)
                    .onSubmit { submit?() }
                    .onKeyPress(.downArrow) {
                        guard let move else { return .ignored }
                        move(1)
                        return .handled
                    }
                    .onKeyPress(.upArrow) {
                        guard let move else { return .ignored }
                        move(-1)
                        return .handled
                    }
                    .onAppear {
                        guard autofocus else { return }
                        DispatchQueue.main.async { focused = true }
                    }
            }
        }
    }
}

// value: where a sheet sits: under a divider, in the flow of the page,
// never a card of its own; scrolled into sight, it keeps the panel's
// margin under its buttons.
struct SheetGround: ViewModifier {
    func body(content: Content) -> some View {
        VStack(alignment: .leading, spacing: Token.Gap.block) {
            SectionDivider()
            content
        }
        .background { Color.clear.id(PanelScroll.sheetID).padding(.bottom, -Token.Space.margin) }
    }
}

// invariant: a select field drawn as its options side by side, the chosen
// one filled, at most four to a row: every option is visible and has its
// room, and it draws offscreen.
public struct Choice: View {
    public static let perRow = 4

    let field: Field
    @Binding var value: String
    @Environment(\.stationPalette) private var palette

    public init(_ field: Field, value: Binding<String>) {
        self.field = field
        _value = value
    }

    public var body: some View {
        let all = (field.options ?? []).map { ($0.label ?? $0.value ?? "", ($0.value ?? "") == value, { [o = $0] in value = o.value ?? "" }) }
        VStack(spacing: Token.Gap.small) {
            ForEach(Array(stride(from: 0, to: all.count, by: Self.perRow)), id: \.self) { i in
                Segmented(options: Array(all[i..<min(i + Self.perRow, all.count)]))
            }
        }
    }
}

// invariant: SPEC §9.8 in the menu bar: a sheet of the call's level: yes or
// no naming what happens, or a name to type whose confirm stays off until
// it matches exactly.
public struct ConfirmSheet: View {
    let confirm: Confirm
    @Binding var typed: String
    let onAnswer: (Bool) -> Void
    @Environment(\.stationPalette) private var palette

    public init(_ confirm: Confirm, typed: Binding<String>, onAnswer: @escaping (Bool) -> Void) {
        self.confirm = confirm
        _typed = typed
        self.onAnswer = onAnswer
    }

    // value: a confirm's line as it is drawn: a command set off as code
    // (`station engine start`), any other line as its words.
    public static func line(_ v: Value) -> (words: String, code: Bool) {
        v.kind == "command" ? (v.text ?? "", true) : (Format.show(v), false)
    }

    public static func allows(_ confirm: Confirm, typed: String) -> Bool {
        guard let name = confirm.type else { return true }
        return typed == name
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Gap.item) {
            Text(confirm.question ?? "").font(.body.weight(.semibold)).foregroundStyle(Token.Ink.primary)
            ForEach(Array((confirm.lines ?? []).enumerated()), id: \.offset) { _, v in
                let line = Self.line(v)
                if line.code {
                    Text(line.words).font(.system(size: Token.Size.sub, design: .monospaced)).foregroundStyle(Token.Ink.primary)
                        .textSelection(.enabled)
                        .padding(.horizontal, Token.Gap.item).padding(.vertical, Token.Gap.inner)
                        .background(RoundedRectangle(cornerRadius: Token.Detail.codeCorner, style: .continuous).fill(Token.Ink.track))
                } else {
                    SubText(line.words)
                }
            }
            if let name = confirm.type {
                TypedField(name, text: $typed, autofocus: true, submit: { if Self.allows(confirm, typed: typed) { onAnswer(true) } })
            }
            HStack {
                Spacer()
                SheetButton(confirm.no ?? "Cancel", role: .plain) { onAnswer(false) }.keyboardShortcut(.cancelAction)
                SheetButton(confirm.yes ?? "Yes", role: confirm.danger == true ? .danger : .accent) { onAnswer(true) }
                    .disabled(!Self.allows(confirm, typed: typed))
                    .opacity(Self.allows(confirm, typed: typed) ? 1 : Token.Fade.disabled)
                    .keyboardShortcut(.defaultAction)
            }
        }
        .modifier(SheetGround())
    }
}

// invariant: a form as a sheet: each select as its options, each text as a
// field, then its action's own words to run it.
public struct FormSheet: View {
    let form: Form
    let run: String
    @Binding var answers: [String: String]
    let onAnswer: (Bool) -> Void
    @Environment(\.stationPalette) private var palette

    public init(_ form: Form, run: String, answers: Binding<[String: String]>, onAnswer: @escaping (Bool) -> Void) {
        self.form = form
        self.run = run
        _answers = answers
        self.onAnswer = onAnswer
    }

    func binding(_ f: Field) -> Binding<String> {
        Binding(get: { answers[f.key ?? ""] ?? f.default ?? "" }, set: { answers[f.key ?? ""] = $0 })
    }

    // value: the field the keys go to as the sheet opens: its first text
    // field, so the owner types at once.
    public static func first(_ form: Form) -> String? { (form.fields ?? []).first { $0.kind != "select" }?.key }

    var complete: Bool { Self.complete(form, answers) }

    // invariant: a form runs once each required field holds something,
    // and a port holds a port: a number from 1 to 65535.
    public static func complete(_ form: Form, _ answers: [String: String]) -> Bool {
        (form.fields ?? []).allSatisfy { f in
            refused(f, answers) == nil && (f.required != true || !value(f, answers).isEmpty)
        }
    }

    static func value(_ f: Field, _ answers: [String: String]) -> String {
        (answers[f.key ?? ""] ?? f.default ?? "").trimmingCharacters(in: .whitespaces)
    }

    // value: why a field's answer cannot run, said under it; nil while it
    // is empty or holds what it asks.
    public static func refused(_ f: Field, _ answers: [String: String]) -> String? {
        let v = value(f, answers)
        guard f.key == portKey, f.kind != "select", !v.isEmpty, !(Int(v).map { (1...65535).contains($0) } ?? false) else { return nil }
        return "A port is a number from 1 to 65535."
    }

    public static let portKey = "Port"

    // value: the confirm's words: a form that picks a term says the term
    // picked (`Until 17:00`), one that makes something new says Create,
    // any other its action's own words.
    public static func verb(_ run: String, _ form: Form, _ answers: [String: String]) -> String {
        if let f = (form.fields ?? []).first(where: { $0.kind == "select" }), let word = f.label, !word.isEmpty, run.hasPrefix(word),
           let chosen = f.options?.first(where: { $0.value == (answers[f.key ?? ""] ?? f.default) })?.label {
            return "\(word) \(chosen)"
        }
        return run.hasPrefix("New ") ? "Create" : run
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Gap.pair) {
            Text(form.title ?? "").font(.body.weight(.semibold)).foregroundStyle(Token.Ink.primary)
            ForEach(Array((form.fields ?? []).enumerated()), id: \.offset) { _, f in
                VStack(alignment: .leading, spacing: Token.Gap.inner) {
                    Text(f.label ?? "").font(.system(size: Token.Space.small)).foregroundStyle(palette.muted)
                    if f.kind == "select" {
                        Choice(f, value: binding(f))
                        if let note = f.options?.first(where: { $0.value == binding(f).wrappedValue })?.note {
                            Text(note).font(.system(size: Token.Space.small)).foregroundStyle(palette.muted)
                        }
                    } else {
                        TypedField(f.note ?? "", text: binding(f), autofocus: Self.first(form) == f.key, submit: { if complete { onAnswer(true) } })
                        if let why = Self.refused(f, answers) {
                            Text(why).font(.system(size: Token.Space.small)).foregroundStyle(palette.bad)
                        }
                    }
                }
            }
            HStack {
                Spacer()
                SheetButton("Cancel", role: .plain) { onAnswer(false) }.keyboardShortcut(.cancelAction)
                SheetButton(Self.verb(run, form, answers), role: .accent) { onAnswer(true) }.disabled(!complete).opacity(complete ? 1 : Token.Fade.disabled).keyboardShortcut(.defaultAction)
            }
        }
        .modifier(SheetGround())
    }
}

public struct SheetButton: View {
    public enum Role { case plain, accent, danger }
    let title: String
    let role: Role
    let action: () -> Void
    @Environment(\.stationPalette) private var palette

    public init(_ title: String, role: Role, action: @escaping () -> Void) {
        self.title = title
        self.role = role
        self.action = action
    }

    public var body: some View {
        StationButton(title, role: role == .accent ? .primary : role == .danger ? .danger : .plain, action: action)
    }
}

