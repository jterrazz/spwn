import SwiftUI

public enum PopoverSide: Sendable {
    case above
    case below
    case tooltip
}

public struct PopoverPin {
    let anchor: Anchor<CGRect>
    let side: PopoverSide
    let width: CGFloat?
    let content: AnyView
}

public struct PopoverPins: PreferenceKey {
    public static let defaultValue: [PopoverPin] = []
    public static func reduce(value: inout [PopoverPin], nextValue: () -> [PopoverPin]) { value += nextValue() }
}

private struct StationTooltipKey: EnvironmentKey {
    static let defaultValue: String? = nil
}

extension EnvironmentValues {
    public var stationTooltip: String? {
        get { self[StationTooltipKey.self] }
        set { self[StationTooltipKey.self] = newValue }
    }
}

extension View {
    public func popoverPin<Content: View>(_ shown: Bool, side: PopoverSide = .below, width: CGFloat? = nil, @ViewBuilder content: () -> Content) -> some View {
        let drawn = shown ? AnyView(content()) : nil
        return anchorPreference(key: PopoverPins.self, value: .bounds) { anchor in
            drawn.map { [PopoverPin(anchor: anchor, side: side, width: width, content: $0)] } ?? []
        }
    }
}

struct PopoverLayer: View {
    let pins: [PopoverPin]
    @Environment(\.stationTokens) private var tokens

    var body: some View {
        GeometryReader { proxy in
            ForEach(Array(pins.enumerated()), id: \.offset) { _, pin in
                PopoverLayout(rect: proxy[pin.anchor], side: pin.side, width: pin.width) {
                    if pin.side != .tooltip {
                        PopoverCard { pin.content }
                    } else {
                        pin.content
                    }
                    PopoverArrow(up: true).fill(tokens.color.sheetSolid)
                    PopoverArrow(up: false).fill(tokens.color.sheetSolid)
                }
                .transition(.modifier(active: PopAppear(progress: 0, kind: .popover), identity: PopAppear(progress: 1, kind: .popover)))
            }
        }
        .animation(Token.Motion.snappy, value: pins.count)
    }
}

struct PopoverCard<Content: View>: View {
    @ViewBuilder let content: Content
    @Environment(\.stationTokens) private var tokens

    var body: some View {
        let shape = RoundedRectangle(cornerRadius: Token.Metric.Popover.radius, style: .continuous)
        content
            .background(shape.fill(tokens.color.sheetSolid))
            .overlay(shape.strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Row.hairline))
    }
}

struct PopoverArrow: Shape {
    let up: Bool

    func path(in rect: CGRect) -> Path {
        var p = Path()
        p.move(to: CGPoint(x: rect.minX, y: up ? rect.maxY : rect.minY))
        p.addLine(to: CGPoint(x: rect.midX, y: up ? rect.minY : rect.maxY))
        p.addLine(to: CGPoint(x: rect.maxX, y: up ? rect.maxY : rect.minY))
        p.closeSubpath()
        return p
    }
}

public struct PopoverLayout: SwiftUI.Layout {
    let rect: CGRect
    let side: PopoverSide
    let width: CGFloat?

    public init(rect: CGRect, side: PopoverSide, width: CGFloat?) {
        self.rect = rect
        self.side = side
        self.width = width
    }

    public struct Spot: Equatable {
        public var card: CGRect
        public var arrow: CGRect?
        public var below: Bool
    }

    public static func spot(_ card: CGSize, of rect: CGRect, in bounds: CGSize, side: PopoverSide) -> Spot {
        let m = Token.Metric.Popover.self
        let clamp = { (x: CGFloat) in min(max(x, m.margin), max(bounds.width - m.margin - card.width, m.margin)) }
        if side == .tooltip {
            let x = clamp(rect.midX - Token.Metric.Tooltip.lead)
            return Spot(card: CGRect(origin: CGPoint(x: x, y: rect.maxY + Token.Metric.Tooltip.gap), size: card), arrow: nil, below: true)
        }
        let fitsBelow = rect.maxY + m.arrowDepth + card.height + m.margin <= bounds.height
        let fitsAbove = rect.minY - m.arrowDepth - card.height >= m.margin
        let below = side == .below ? fitsBelow || !fitsAbove : !fitsAbove && fitsBelow
        let x = clamp(rect.midX - card.width / 2)
        let y = below ? rect.maxY + m.arrowDepth : rect.minY - m.arrowDepth - card.height
        let tip = min(max(rect.midX - m.arrow / 2, x + m.radius), x + card.width - m.radius - m.arrow)
        let arrow = CGRect(x: tip, y: below ? rect.maxY : rect.minY - m.arrowDepth, width: m.arrow, height: m.arrowDepth)
        return Spot(card: CGRect(origin: CGPoint(x: x, y: y), size: card), arrow: arrow, below: below)
    }

    public func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        proposal.replacingUnspecifiedDimensions()
    }

    public func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        guard subviews.count == 3 else { return }
        let card = subviews[0].sizeThatFits(ProposedViewSize(width: width, height: nil))
        let spot = Self.spot(card, of: rect, in: bounds.size, side: side)
        let origin = { (r: CGRect) in CGPoint(x: bounds.minX + r.minX, y: bounds.minY + r.minY) }
        subviews[0].place(at: origin(spot.card), anchor: .topLeading, proposal: ProposedViewSize(spot.card.size))
        for (i, up) in [(1, true), (2, false)] {
            let shown = spot.arrow.map { up == spot.below ? $0 : nil } ?? nil
            subviews[i].place(at: origin(shown ?? .zero), anchor: .topLeading, proposal: ProposedViewSize(shown?.size ?? .zero))
        }
    }
}
