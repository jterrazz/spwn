import SwiftUI

// invariant: SPEC A6 Find's field: 30 tall, round, the accent rim at 55 %,
// the search glyph, its placeholder; it takes the keys as Find opens, its
// arrows move the light, its Return runs the line lit, ⌘↩ its alternate,
// esc closes Find. Drawn still, the text and a caret stand in for the
// AppKit field.
public struct FindField: View {
    @Binding var text: String
    let move: (Int) -> Void
    let submit: (Bool) -> Void
    let close: () -> Void
    @FocusState private var focused: Bool
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still

    public init(text: Binding<String>, move: @escaping (Int) -> Void = { _ in }, submit: @escaping (Bool) -> Void = { _ in }, close: @escaping () -> Void = {}) {
        _text = text
        self.move = move
        self.submit = submit
        self.close = close
    }

    public var body: some View {
        let m = Token.Metric.Find.self
        let f = Token.Find.self
        let shape = RoundedRectangle(cornerRadius: m.fieldRadius, style: .continuous)
        HStack(spacing: f.fieldGap) {
            Image(systemName: Token.Symbol.search)
                .font(.system(size: Token.Metric.Marks.glyph, weight: .medium))
                .frame(width: f.fieldGlyph, height: f.fieldGlyph)
                .foregroundStyle(tokens.color.t3)
            field
        }
        .font(Token.Role.findResult.font)
        .padding(f.fieldPadding)
        .frame(height: m.field)
        .background(shape.fill(tokens.color.q))
        .overlay(shape.strokeBorder(tokens.color.accent.opacity(m.focusShare), lineWidth: f.fieldRim))
    }

    @ViewBuilder
    var field: some View {
        if still {
            HStack(spacing: 0) {
                Text(text.isEmpty ? Token.FindCapsule.words : text)
                    .foregroundStyle(text.isEmpty ? tokens.color.t3 : tokens.color.t1)
                    .lineLimit(1)
                Spacer(minLength: 0)
            }
        } else {
            TextField(Token.FindCapsule.words, text: $text)
                .textFieldStyle(.plain)
                .foregroundStyle(tokens.color.t1)
                .focused($focused)
                .onSubmit { submit(false) }
                .onKeyPress(.downArrow) {
                    move(1)
                    return .handled
                }
                .onKeyPress(.upArrow) {
                    move(-1)
                    return .handled
                }
                .onKeyPress(.escape) {
                    close()
                    return .handled
                }
                .background {
                    Button("") { submit(true) }
                        .keyboardShortcut(.return, modifiers: .command)
                        .opacity(0)
                        .allowsHitTesting(false)
                        .accessibilityHidden(true)
                }
                .onAppear { DispatchQueue.main.async { focused = true } }
        }
    }
}

// invariant: SPEC A6 a result: its glyph in the 24 slot, its title with the
// part that matched in t1 semibold (a port's address after it, in mono),
// its scope trailing in t3 and the first to shorten, its key on the line
// lit; the line lit is accent with white text.
public struct FindResult: View {
    let entry: FindEntry
    let query: String
    let lit: Bool
    let asks: Bool
    let key: String?
    @Environment(\.stationTokens) private var tokens

    public init(_ entry: FindEntry, query: String, lit: Bool, asks: Bool = false, key: String? = nil) {
        self.entry = entry
        self.query = query
        self.lit = lit
        self.asks = asks
        self.key = key
    }

    var soft: Color { lit ? tokens.onTint.opacity(Token.Find.onTintSoft) : tokens.color.t2 }
    var strong: Color { lit ? tokens.onTint : tokens.color.t1 }
    var faint: Color { lit ? tokens.onTint.opacity(Token.Find.onTintSoft) : tokens.color.t3 }

    public var body: some View {
        let f = Token.Find.self
        HStack(spacing: f.gap) {
            glyph.frame(width: f.glyphSlot)
            title.lineLimit(1).truncationMode(.tail).layoutPriority(1)
            Spacer(minLength: 0)
            if let scope = entry.scope {
                Text(scope).font(.system(size: f.scope)).foregroundStyle(faint).lineLimit(1).truncationMode(.tail)
                    .padding(.leading, f.scopeGap)
            }
            if lit, let key { KeyCap(key, onTint: true) }
        }
        .padding(f.result)
        .frame(height: Token.Metric.Find.result)
        .background {
            if lit { RoundedRectangle(cornerRadius: Token.Metric.Find.resultRadius, style: .continuous).fill(tokens.color.accent) }
        }
        .contentShape(Rectangle())
    }

    @ViewBuilder
    var glyph: some View {
        if let g = entry.row.glyph {
            switch GlyphView.drawn(g) {
            case let .chip(kind, brand): LogoChip(kind, brand)
            case let .symbol(name): mark(Image(systemName: name), g)
            case let .custom(name): mark(Image(name, bundle: .module), g)
            case let .mac(lid): mark(Image(systemName: Token.Object.mac(lid: lid)), g)
            }
        } else if let mark = entry.row.mark {
            MarkView(mark)
        }
    }

    func mark(_ image: Image, _ g: Glyph) -> some View {
        let tone = entry.group == FindIndex.needs ? tokens.signal(g.signal) : tokens.color.t2
        return image.font(.system(size: Token.Metric.Marks.glyph, weight: .medium)).foregroundStyle(lit ? strong : tone)
    }

    var title: Text {
        let words = asks ? entry.title + Token.Glyph.ellipsis : entry.title
        let q = query.trimmingCharacters(in: .whitespaces)
        let matched = !q.isEmpty && (Self.split(words, q) != nil || entry.address.flatMap { Self.split($0, q) } != nil)
        var out = said(words, q, plain: matched ? soft : strong, font: Token.Role.findResult.font)
        if let address = entry.address {
            out = out + Text(Token.Glyph.sep).foregroundStyle(matched ? soft : strong)
                + said(address, q, plain: matched ? soft : strong, font: .system(size: Token.Find.address, design: .monospaced))
        }
        return out
    }

    func said(_ words: String, _ q: String, plain: Color, font: Font) -> Text {
        guard let (before, hit, after) = Self.split(words, q) else { return Text(words).font(font).foregroundStyle(plain) }
        return Text(before).font(font).foregroundStyle(plain)
            + Text(hit).font(font.weight(.semibold)).foregroundStyle(strong)
            + Text(after).font(font).foregroundStyle(plain)
    }

    // value: words cut around the first place the query is found in them,
    // case aside.
    public static func split(_ words: String, _ q: String) -> (String, String, String)? {
        guard !q.isEmpty, let r = words.range(of: q, options: .caseInsensitive) else { return nil }
        return (String(words[..<r.lowerBound]), String(words[r]), String(words[r.upperBound...]))
    }
}

// value: a group's "Show N more": its chevron and its words in t2.
public struct FindMore: View {
    let count: Int
    let lit: Bool
    @Environment(\.stationTokens) private var tokens

    public init(_ count: Int, lit: Bool) {
        self.count = count
        self.lit = lit
    }

    public var body: some View {
        let f = Token.Find.self
        let tone = lit ? tokens.onTint : tokens.color.t2
        HStack(spacing: f.gap) {
            Image(systemName: f.moreSymbol).font(.system(size: f.moreGlyph, weight: .semibold)).foregroundStyle(tone).frame(width: f.glyphSlot)
            Text(f.more(count)).font(Token.Role.findResult.font).foregroundStyle(tone).lineLimit(1)
            Spacer(minLength: 0)
        }
        .padding(f.result)
        .frame(height: Token.Metric.Find.result)
        .background {
            if lit { RoundedRectangle(cornerRadius: Token.Metric.Find.resultRadius, style: .continuous).fill(tokens.color.accent) }
        }
        .contentShape(Rectangle())
    }
}
