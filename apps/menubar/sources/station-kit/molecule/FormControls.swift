import SwiftUI

typealias S = Token.Metric.Sheet

struct RoleText: View {
    let text: String
    let role: Token.Role
    @Environment(\.stationTokens) private var tokens

    init(_ text: String, _ role: Token.Role) {
        self.text = text
        self.role = role
    }

    var body: some View {
        Text(text).font(role.font).foregroundStyle(tokens.tone(role.tone)).lineLimit(1)
    }
}

// value: a sheet field's ground: the quiet fill every control of the form
// sits on.
struct FieldBox: ViewModifier {
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        content
            .padding(.horizontal, S.fieldPadding)
            .frame(height: S.field)
            .background(RoundedRectangle(cornerRadius: S.fieldRadius, style: .continuous).fill(tokens.color.q))
    }
}

// invariant: a popup (the On of New VM): its pick and that pick's note,
// the system's menu of every option in the app, its stand-in drawn still.
public struct PopupField: View {
    let field: Field
    let answers: Answers
    let pick: (String) -> Void
    @Environment(\.stationStill) private var still
    @Environment(\.stationTokens) private var tokens

    public init(_ field: Field, _ answers: Answers, pick: @escaping (String) -> Void) {
        self.field = field
        self.answers = answers
        self.pick = pick
    }

    var chosen: Option? { field.options?.first { $0.value == answers.value(field) } }

    var face: some View {
        HStack(spacing: S.popupGap) {
            if let glyph = chosen?.glyph { OptionGlyph(glyph) }
            RoleText(chosen?.label ?? "", .formValue)
            if let note = chosen?.note { RoleText(answers.fill(note), .formNote) }
            Image(systemName: Token.Symbol.popup).font(.system(size: S.chevron, weight: .semibold)).foregroundStyle(tokens.color.t2)
        }
        .modifier(FieldBox())
        .menuAnchor(SheetAnchor.field(field.key ?? ""))
    }

    public var body: some View {
        if still {
            face
        } else {
            Menu {
                ForEach(Array(MenuModel.popup(field, answers).items.enumerated()), id: \.offset) { _, item in
                    Button([item.title, item.note].compactMap { $0 }.joined(separator: "  ")) { pick(item.id) }.disabled(!item.enabled)
                }
            } label: { face }
                .menuStyle(.button).buttonStyle(.plain).fixedSize()
        }
    }
}

// value: a disclosure's row: what it holds said in one line, its chevron
// pointing where it goes.
public struct DisclosureRow: View {
    let summary: String
    let open: Bool
    let toggle: () -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ summary: String, open: Bool, toggle: @escaping () -> Void) {
        self.summary = summary
        self.open = open
        self.toggle = toggle
    }

    public var body: some View {
        Button(action: toggle) {
            HStack(spacing: S.popupGap) {
                RoleText(summary, .formValue)
                Spacer(minLength: 0)
                Image(systemName: open ? Token.Symbol.up : Token.Symbol.down).font(.system(size: S.chevron, weight: .semibold)).foregroundStyle(tokens.color.t2)
            }
            .modifier(FieldBox())
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityValue(open ? "open" : "closed")
    }
}

// invariant: SPEC A9 1-8: the memory slider on a log scale, every stop
// labeled, the part past the Mac's free memory hatched from where that
// memory truly falls, its helper on its own line in the engine's words.
public struct MemorySlider: View {
    let field: Field
    let answers: Answers
    let pick: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ field: Field, _ answers: Answers, pick: @escaping (String) -> Void) {
        self.field = field
        self.answers = answers
        self.pick = pick
    }

    // value: where n falls between the first stop and the last, 0 to 1.
    public static func share(_ n: Double, stops: [Int]) -> Double {
        guard let first = stops.first, let last = stops.last, first > 0, last > first, n > 0 else { return 0 }
        return min(max(log2(n / Double(first)) / log2(Double(last) / Double(first)), 0), 1)
    }

    var stops: [Int] { field.stops ?? [] }
    var options: [Option] { field.options ?? [] }
    var picked: Int { options.firstIndex { $0.value == answers.value(field) } ?? 0 }

    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            GeometryReader { g in
                let w = g.size.width
                let at = Self.share(Double(stops.indices.contains(picked) ? stops[picked] : 0), stops: stops) * w
                ZStack(alignment: .leading) {
                    Capsule().fill(tokens.color.track).frame(height: S.trackHeight)
                    if let limit = field.limit, limit > 0, Double(limit) < Double(stops.last ?? 0) {
                        let from = Self.share(Double(limit), stops: stops) * w
                        Hatch().stroke(tokens.color.t4, lineWidth: S.hatchStroke)
                            .frame(width: w - from, height: S.trackHeight)
                            .clipShape(Capsule())
                            .offset(x: from)
                    }
                    Capsule().fill(tokens.color.accent).frame(width: at, height: S.trackHeight)
                    Circle().fill(tokens.knob).frame(width: S.knob, height: S.knob)
                        .shadow(color: Token.Shadow.knob.color, radius: Token.Shadow.knob.radius, y: Token.Shadow.knob.y)
                        .offset(x: at - S.knob / 2)
                }
                .frame(height: S.knob)
            }
            .frame(height: S.knob)
            GeometryReader { g in
                let w = g.size.width
                ZStack(alignment: .topLeading) {
                    ForEach(Array(stops.enumerated()), id: \.offset) { i, n in
                        let x = Self.share(Double(n), stops: stops) * w
                        let option = options.indices.contains(i) ? options[i] : Option()
                        VStack(spacing: S.labelsGap) {
                            Rectangle().fill(i == picked ? tokens.color.accent : tokens.color.t3).frame(width: S.tick.width, height: S.tick.height)
                                .frame(maxWidth: .infinity, alignment: i == 0 ? .leading : i == stops.count - 1 ? .trailing : .center)
                            Button { pick(option.value ?? "") } label: {
                                RoleText(option.label ?? "", i == picked ? .stopPicked : .stopLabel)
                            }
                            .buttonStyle(.plain).disabled(option.disabled != nil)
                        }
                        .fixedSize()
                        .alignmentGuide(.leading) { d in i == 0 ? 0 : i == stops.count - 1 ? d.width - w : d.width / 2 - x }
                    }
                }
            }
            .frame(height: S.tick.height + S.labelsGap + Token.Role.stopLabel.size + S.tickGap)
            if let note = field.limit_note {
                RoleText(note, .formCaption).padding(.top, S.helperTop)
            }
        }
    }
}

struct Hatch: Shape {
    func path(in rect: CGRect) -> Path {
        var p = Path()
        var x = rect.minX - rect.height
        while x < rect.maxX {
            p.move(to: CGPoint(x: x, y: rect.maxY))
            p.addLine(to: CGPoint(x: x + rect.height, y: rect.minY))
            x += S.hatch
        }
        return p
    }
}

// invariant: SPEC A9 1-9: the accounts to share as chips, each checked or
// not, one the Mac cannot share refused with why under them.
public struct ChipPicker: View {
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
        var now = answers.picked(field)
        if let i = now.firstIndex(of: value) { now.remove(at: i) } else { now.append(value) }
        return (field.options ?? []).compactMap(\.value).filter(now.contains).joined(separator: ",")
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: S.caption) {
            HStack(spacing: S.chipGap) {
                ForEach(Array((field.options ?? []).enumerated()), id: \.offset) { _, o in
                    let on = answers.picked(field).contains(o.value ?? "")
                    Button { set(toggled(o.value ?? "")) } label: {
                        HStack(spacing: S.popupGap) {
                            ZStack {
                                RoundedRectangle(cornerRadius: S.boxRadius, style: .continuous).fill(on ? tokens.color.accent : Color.clear)
                                if !on { RoundedRectangle(cornerRadius: S.boxRadius, style: .continuous).strokeBorder(tokens.color.t3, lineWidth: S.boxStroke) }
                                if on { Image(systemName: Token.Symbol.check).font(.system(size: S.boxMark, weight: .bold)).foregroundStyle(tokens.onTint) }
                            }
                            .frame(width: S.box, height: S.box)
                            RoleText(o.label ?? "", .formValue)
                        }
                        .padding(.horizontal, S.chipPadding)
                        .frame(height: S.chip)
                        .background(Capsule().fill(tokens.color.q))
                    }
                    .buttonStyle(.plain)
                    .disabled(o.disabled != nil)
                    .opacity(o.disabled != nil ? Token.Fade.disabled : 1)
                }
            }
            ForEach(Array((field.options ?? []).compactMap(\.disabled).enumerated()), id: \.offset) { _, why in
                RoleText(why, .formCaption)
            }
        }
    }
}

// invariant: the split button: its action, then the arrow that offers
// the other ways (the system's menu in the app, drawn still).
public struct SplitButton: View {
    let title: String
    let items: [Action]
    let run: () -> Void
    let other: (Action) -> Void
    @Environment(\.stationStill) private var still
    @Environment(\.stationTokens) private var tokens

    public init(_ title: String, items: [Action], run: @escaping () -> Void, other: @escaping (Action) -> Void) {
        self.title = title
        self.items = items
        self.run = run
        self.other = other
    }

    var arrow: some View {
        Image(systemName: Token.Symbol.down).font(.system(size: S.chevron, weight: .bold)).foregroundStyle(tokens.onTint)
            .frame(width: S.splitArrow, height: Token.Metric.Control.button.height).contentShape(Rectangle())
    }

    public var body: some View {
        let c = Token.Metric.Control.button
        HStack(spacing: 0) {
            Button(action: run) {
                Text(title).font(Token.Role.button.font).foregroundStyle(tokens.onTint)
                    .padding(.horizontal, c.padding).frame(height: c.height).contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .keyboardShortcut(.defaultAction)
            Rectangle().fill(tokens.onTint.opacity(Token.Fade.disabled)).frame(width: S.splitDivider, height: c.height)
            if still || items.isEmpty {
                arrow.menuAnchor(SheetAnchor.split)
            } else {
                Menu {
                    ForEach(Array(items.enumerated()), id: \.offset) { _, a in
                        Button(a.label ?? "") { other(a) }
                    }
                } label: { arrow }
                    .menuStyle(.button).buttonStyle(.plain).menuIndicator(.hidden).fixedSize()
            }
        }
        .background(Capsule().fill(tokens.color.accent))
    }
}

// invariant: a sheet's text field on the form's 26 pt ground: the text and
// its caret drawn still, a live text field in the app.
public struct FormText: View {
    @Binding var text: String
    let submit: () -> Void
    @Environment(\.stationStill) private var still
    @Environment(\.stationTokens) private var tokens

    public init(text: Binding<String>, submit: @escaping () -> Void) {
        _text = text
        self.submit = submit
    }

    public var body: some View {
        HStack(spacing: 0) {
            if still {
                RoleText(text, .formValue)
                Rectangle().fill(tokens.color.accent).frame(width: Token.Stroke.caret, height: Token.Detail.caretHeight)
            } else {
                TextField("", text: $text).textFieldStyle(.plain).font(Token.Role.formValue.font).onSubmit(submit)
            }
            Spacer(minLength: 0)
        }
        .modifier(FieldBox())
    }
}
