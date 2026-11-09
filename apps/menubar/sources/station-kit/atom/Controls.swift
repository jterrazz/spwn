import SwiftUI

public struct Switch: View {
    let on: Bool
    let tap: () -> Void
    @Environment(\.stationTokens) private var tokens

    public init(on: Bool, tap: @escaping () -> Void = {}) {
        self.on = on
        self.tap = tap
    }

    public var body: some View {
        let c = Token.Metric.Control.self
        Button(action: tap) {
            ZStack(alignment: on ? .trailing : .leading) {
                Capsule().fill(on ? tokens.color.accent : tokens.switchOff)
                    .overlay { if tokens.increased { Capsule().strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Card.contrastStroke) } }
                    .frame(width: c.switchSize.width, height: c.switchSize.height)
                Circle().fill(tokens.knob).frame(width: c.knob, height: c.knob)
                    .shadow(color: Token.Shadow.switchKnob.color, radius: Token.Shadow.switchKnob.radius, y: Token.Shadow.switchKnob.y)
                    .padding(.horizontal, c.knobInset)
            }
            .contentShape(Capsule())
        }
        .buttonStyle(.plain)
        .animation(tokens.motion(Token.Motion.snappy), value: on)
        .accessibilityAddTraits(.isToggle)
        .accessibilityValue(on ? "on" : "off")
    }
}

public struct Segmented: View {
    let options: [(label: String, on: Bool, tap: () -> Void)]
    let keys: [String?]
    let leaving: Int?
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationCaught) private var caught
    @Namespace private var thumb

    // value: leaving, the segment the thumb slides away from while a
    // moment is caught.
    public init(options: [(label: String, on: Bool, tap: () -> Void)], keys: [String?] = [], leaving: Int? = nil) {
        self.options = options
        self.keys = keys
        self.leaving = leaving
    }

    var sliding: (from: Int, to: Int, time: Double)? {
        guard let caught, let leaving, let to = options.firstIndex(where: \.on), leaving != to else { return nil }
        return (leaving, to, caught.time)
    }

    func thumbShape() -> some View {
        let c = Token.Metric.Control.self
        return RoundedRectangle(cornerRadius: c.thumbRadius, style: .continuous)
            .fill(tokens.thumb)
            .overlay(RoundedRectangle(cornerRadius: c.thumbRadius, style: .continuous).strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Row.hairline))
            .shadow(color: Token.Shadow.segmentThumb.color, radius: Token.Shadow.segmentThumb.radius, y: Token.Shadow.segmentThumb.y)
    }

    func shortcut(_ i: Int) -> KeyboardShortcut? {
        keys.indices.contains(i) ? keys[i].flatMap(MenuModel.shortcut) : nil
    }

    public var body: some View {
        let c = Token.Metric.Control.self
        HStack(spacing: 0) {
            ForEach(Array(options.enumerated()), id: \.offset) { i, o in
                Button(action: o.tap) {
                    Text(o.label)
                        .font(.system(size: c.segmentedSize).monospacedDigit())
                        .foregroundStyle(o.on ? tokens.color.t1 : tokens.color.t2)
                        .lineLimit(1)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .background {
                            if o.on && sliding == nil {
                                thumbShape().matchedGeometryEffect(id: "thumb", in: thumb)
                            }
                        }
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .keyboardShortcut(shortcut(i))
                .accessibilityLabel(o.label)
                .accessibilityAddTraits(o.on ? .isSelected : [])
            }
        }
        .background(alignment: .topLeading) {
            if let sliding {
                GeometryReader { g in
                    let w = g.size.width / CGFloat(max(options.count, 1))
                    let at = { (i: Int) in CGRect(x: w * CGFloat(i), y: 0, width: w, height: g.size.height) }
                    thumbShape().modifier(SegmentThumb(progress: SegmentThumb.frozen(sliding.time, reduce: tokens.reduceMotion), from: at(sliding.from), to: at(sliding.to)))
                }
            }
        }
        .padding(c.segmented.padding)
        .frame(height: c.segmented.height)
        .background(RoundedRectangle(cornerRadius: c.segmented.radius, style: .continuous).fill(tokens.color.q))
        .overlay { if tokens.increased { RoundedRectangle(cornerRadius: c.segmented.radius, style: .continuous).strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Card.contrastStroke) } }
        .animation(tokens.motion(Token.Motion.snappy), value: options.map(\.on))
    }
}

public struct PanelButton: View {
    public enum Look: Sendable { case plain, tint, soft, redTint, destructive }

    let title: String
    let symbol: String?
    let key: String?
    let look: Look
    let big: Bool
    let grow: Bool
    let action: () -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.isEnabled) private var enabled
    @State private var hovering = false

    public init(_ title: String, symbol: String? = nil, key: String? = nil, look: Look = .plain, big: Bool = false, grow: Bool = false, action: @escaping () -> Void = {}) {
        self.title = title
        self.symbol = symbol
        self.key = key
        self.look = look
        self.big = big
        self.grow = grow
        self.action = action
    }

    var ink: Color {
        switch look {
        case .tint: tokens.onTint
        case .soft: tokens.color.accentText
        case .redTint, .destructive: tokens.color.redText
        case .plain: tokens.color.t1
        }
    }

    var fill: Color {
        let c = Token.Metric.Control.self
        switch look {
        case .tint: return tokens.color.accent
        case .soft: return tokens.color.accent.opacity(c.softShare)
        case .redTint: return tokens.color.red.opacity(c.redShare)
        case .plain, .destructive: return hovering ? tokens.color.qq : tokens.color.q
        }
    }

    public var body: some View {
        let c = Token.Metric.Control.self
        let size = big ? c.big : c.button
        let shape = RoundedRectangle(cornerRadius: size.radius, style: .continuous)
        Button(action: action) {
            HStack(spacing: c.buttonGap) {
                if let symbol { Image(systemName: symbol).font(.system(size: c.buttonGlyph, weight: .semibold)) }
                Text(title).font(Token.Role.button.font).lineLimit(1)
                if let key { KeyCap(key, onTint: look == .tint) }
            }
            .fixedSize()
            .foregroundStyle(ink)
            .padding(.horizontal, size.padding)
            .frame(maxWidth: grow ? .infinity : nil)
            .frame(height: size.height)
            .background(shape.fill(fill))
            .overlay {
                if tokens.increased {
                    shape.strokeBorder(look == .tint ? tokens.contrastRim : tokens.color.hair, lineWidth: c.contrastRim)
                }
            }
            .opacity(enabled ? 1 : c.disabledShare)
            .contentShape(shape)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(title)
        .onHover { hovering = $0 }
        .animation(tokens.motion(Token.Motion.tint), value: hovering)
    }
}

public struct VerbButton: View {
    let title: String
    let busy: Bool
    let action: () -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    public init(_ title: String, busy: Bool = false, action: @escaping () -> Void = {}) {
        self.title = title
        self.busy = busy
        self.action = action
    }

    public var body: some View {
        let c = Token.Metric.Control.self
        let shape = RoundedRectangle(cornerRadius: c.verb.radius, style: .continuous)
        Button(action: action) {
            Text(title).font(Token.Role.verb.font).lineLimit(1)
                .foregroundStyle(busy ? tokens.color.t3 : tokens.color.t1)
                .padding(.horizontal, c.verbPadding)
                .frame(minWidth: c.verbMinWidth)
                .frame(height: c.verb.height)
                .background(shape.fill(hovering ? tokens.color.qq : tokens.color.q))
                .overlay { if tokens.increased { shape.strokeBorder(tokens.color.hair, lineWidth: c.contrastRim) } }
                .contentShape(shape)
        }
        .buttonStyle(.plain)
        .fixedSize()
        .accessibilityLabel(title)
        .onHover { hovering = $0 }
        .animation(tokens.motion(Token.Motion.tint), value: hovering)
    }
}

public struct RoundButton: View {
    public enum Size: Sendable { case regular, small, topBar }

    let symbol: String
    let size: Size
    let quiet: Bool
    let action: () -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    public init(_ symbol: String, size: Size = .regular, quiet: Bool = false, action: @escaping () -> Void = {}) {
        self.symbol = symbol
        self.size = size
        self.quiet = quiet
        self.action = action
    }

    var measure: (box: CGFloat, glyph: CGFloat) {
        let c = Token.Metric.Control.self
        switch size {
        case .regular: return (c.round, c.roundGlyph)
        case .small: return (c.roundSmall, c.roundSmallGlyph)
        case .topBar: return (c.roundTopBar, c.roundTopBarGlyph)
        }
    }

    var fill: Color {
        if quiet { return hovering ? tokens.color.q : Color.clear }
        return hovering ? tokens.color.qq : tokens.color.q
    }

    public var body: some View {
        Button(action: action) {
            Image(systemName: symbol).font(.system(size: measure.glyph, weight: .semibold))
                .foregroundStyle(hovering ? tokens.color.t1 : quiet ? tokens.color.t3 : tokens.color.t2)
                .frame(width: measure.box, height: measure.box)
                .background(Circle().fill(fill))
                .overlay { if tokens.increased && !quiet { Circle().strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Control.contrastRim) } }
                .contentShape(Circle())
        }
        .buttonStyle(.plain)
        .onHover { hovering = $0 }
        .animation(tokens.motion(Token.Motion.tint), value: hovering)
    }
}

public struct IconButton: View {
    let symbol: String
    let action: () -> Void
    @Environment(\.stationTokens) private var tokens
    @State private var hovering = false

    public init(_ symbol: String, action: @escaping () -> Void = {}) {
        self.symbol = symbol
        self.action = action
    }

    public var body: some View {
        let c = Token.Metric.Control.self
        Button(action: action) {
            Image(systemName: symbol).font(.system(size: c.iconGlyph, weight: .regular))
                .foregroundStyle(hovering ? tokens.color.t1 : tokens.color.t2)
                .frame(width: c.icon, height: c.icon)
                .background(Circle().fill(hovering ? tokens.color.q : Color.clear))
                .contentShape(Circle())
        }
        .buttonStyle(.plain)
        .onHover { hovering = $0 }
        .animation(tokens.motion(Token.Motion.tint), value: hovering)
    }
}
