import SwiftUI

// invariant: SPEC A8's rule: every movement is a modifier of one progress,
// 0 where the view is absent or the movement starts, 1 where it is drawn
// as at rest, so the live app runs it with withAnimation(token) and a story
// draws it at any moment through Token.Motion.freeze.
public protocol Movement: ViewModifier, Animatable where AnimatableData == Double {
    var progress: Double { get set }
    static var pace: Token.Motion.Pace { get }
}

extension Movement {
    public var animatableData: Double {
        get { progress }
        set { progress = newValue }
    }

    public static var pace: Token.Motion.Pace { .curve(Token.Motion.smoothCurve) }

    // invariant: under Reduce Motion a movement keeps its dissolve, on the
    // reduced ease.
    public static func frozen(_ time: Double, reduce: Bool = false) -> Double {
        reduce ? Token.Motion.freeze(time, Token.Motion.reducedCurve) : Token.Motion.freeze(time, pace)
    }
}

extension AnyTransition {
    // invariant: SPEC A8's transition form: an insertion runs a movement
    // from 0 to 1 and a removal from 1 to 0, so either is drawn at a progress.
    public static func movement<M: Movement>(_ make: @escaping (Double) -> M) -> AnyTransition {
        .modifier(active: make(0), identity: make(1))
    }
}

// value: what a movement draws at one progress; every field at rest is
// the view as it stands.
public struct Look: Equatable, Sendable {
    public var opacity = 1.0
    public var scale = Look.scaled(1)
    public var x: CGFloat = 0
    public var y: CGFloat = 0
    public var slide = 0.0
    public var turn = 0.0
    public var height = 1.0
    public var reveal = 1.0
    public var tint = 0.0
    public var shade = 0.0

    public init() {}

    static func lerp(_ from: Double, _ to: Double, _ share: Double) -> Double { from + (to - from) * share }

    static func scaled(_ s: Double, tall: Double? = nil) -> CGSize { CGSize(width: s, height: tall ?? s) }

    static func delayed(_ progress: Double, by delay: Double) -> Double {
        delay >= 1 ? (progress >= 1 ? 1 : 0) : min(max((progress - delay) / (1 - delay), 0), 1)
    }

    static func clamped(_ share: Double) -> Double { min(max(share, 0), 1) }

    static func faded(_ progress: Double, over share: Double) -> Double { min(max(progress / share, 0), 1) }
}

struct Moved: ViewModifier {
    let look: Look
    var anchor: UnitPoint = .center

    func body(content: Content) -> some View {
        let slide = look.slide
        content
            .scaleEffect(look.scale, anchor: anchor)
            .rotationEffect(.degrees(look.turn))
            .visualEffect { view, frame in view.offset(x: frame.size.width * slide) }
            .offset(x: look.x, y: look.y)
            .opacity(look.opacity)
    }
}

extension View {
    func moved(_ look: Look, anchor: UnitPoint = .center) -> some View { modifier(Moved(look: look, anchor: anchor)) }
}

// invariant: a fold or a leaving row is as tall as its share of what it
// lays out at, measured by its own layout: a maxHeight of nil cannot be
// eased, and a measure read back into state cannot be drawn in one pass.
struct Collapse: SwiftUI.Layout {
    var share: Double

    var animatableData: Double {
        get { share }
        set { share = newValue }
    }

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let ideal = subviews.first?.sizeThatFits(ProposedViewSize(width: proposal.width, height: nil)) ?? .zero
        return CGSize(width: proposal.width ?? ideal.width, height: ideal.height * Look.clamped(share))
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        guard let content = subviews.first else { return }
        let ideal = content.sizeThatFits(ProposedViewSize(width: bounds.width, height: nil))
        content.place(at: bounds.origin, anchor: .topLeading, proposal: ProposedViewSize(width: bounds.width, height: ideal.height))
    }
}

// value: a reveal from one edge: the share of the view shown, past the
// part shown from the first frame.
struct Unroll: Shape {
    var share: Double
    var edge: Edge = .top
    var least: CGFloat = 0
    var radius: CGFloat = 0

    var animatableData: Double {
        get { share }
        set { share = newValue }
    }

    func path(in rect: CGRect) -> Path {
        let s = min(max(share, 0), 1)
        var shown = rect
        switch edge {
        case .top, .bottom: shown.size.height = min(least, rect.height) + max(rect.height - least, 0) * s
        case .leading, .trailing: shown.size.width = min(least, rect.width) + max(rect.width - least, 0) * s
        }
        if edge == .bottom { shown.origin.y = rect.maxY - shown.height }
        if edge == .trailing { shown.origin.x = rect.maxX - shown.width }
        return Path(roundedRect: shown, cornerRadius: radius, style: .continuous)
    }
}
