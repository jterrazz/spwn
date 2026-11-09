import SwiftUI

// value: M1, the panel grows out of its status item; it closes to a
// slighter scale on snappy, in the panel of its own (the system's window
// closes at once).
public struct PanelAppear: Movement {
    public var progress: Double
    public var closing: Bool
    public var anchor: UnitPoint
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double, closing: Bool = false, anchor: UnitPoint = .top) {
        self.progress = progress
        self.closing = closing
        self.anchor = anchor
    }

    public static func look(_ progress: Double, closing: Bool = false, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = progress
        if !reduce { look.scale = Look.scaled(Look.lerp(closing ? Token.Motion.Move.panelClose : Token.Motion.Move.panelFrom, 1, progress)) }
        return look
    }

    public static func frozen(_ time: Double, closing: Bool, reduce: Bool = false) -> Double {
        if reduce { return Token.Motion.freeze(time, Token.Motion.reducedCurve) }
        return closing ? 1 - Token.Motion.freeze(time, Token.Motion.snappyCurve) : Token.Motion.freeze(time, Token.Motion.smoothCurve)
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, closing: closing, reduce: tokens.reduceMotion), anchor: anchor)
    }
}

// invariant: M1 in the live app: each time the system shows the panel's
// window it grows out of its status item on smooth, from nothing; once
// hidden it is at its start again, so the next opening's first frame is
// never the panel whole.
public struct PanelOpening: ViewModifier {
    @State private var shown = 0.0
    @Environment(\.stationTokens) private var tokens

    public init() {}

    public func body(content: Content) -> some View {
        content
            .modifier(PanelAppear(progress: shown))
            .onAppear { withAnimation(tokens.motion(Token.Motion.smooth)) { shown = 1 } }
            .onDisappear { shown = 0 }
    }
}

// value: M2 and M3, a page pushed or popped: the top page slides on the
// trailing edge, opaque and shaded; the page under it moves a third and
// dims. Progress is how far each page is shown, so at a push's moment t the
// top page is at pace(t) and the page under at 1 - pace(t).
public struct PushTransition: Movement {
    public enum Role: Sendable { case top, under }
    public enum Way: Sendable { case push, pop }

    public var progress: Double
    public var role: Role
    public var way: Way
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double, role: Role, way: Way = .push) {
        self.progress = progress
        self.role = role
        self.way = way
    }

    // invariant: under Reduce Motion the leaving page is gone at 45 % of
    // the time before the arriving one shows, so two pages never print
    // over each other.
    public static func look(_ progress: Double, role: Role, way: Way, reduce: Bool) -> Look {
        let m = Token.Motion.Move.self
        var look = Look()
        let leaving = (way == .push) == (role == .under)
        if reduce {
            let gone = Token.Motion.reducedPageGone
            look.opacity = leaving ? max((progress - (1 - gone)) / gone, 0) : max((progress - gone) / (1 - gone), 0)
            return look
        }
        switch (role, way) {
        case (.top, _):
            look.slide = 1 - progress
            look.opacity = way == .push ? 1 : Look.lerp(m.popFade, 1, progress)
            look.shade = progress < 1 ? 1 : 0
        case (.under, .push):
            let gone = 1 - progress
            look.slide = -m.pushUnder * gone
            look.opacity = gone <= m.pushDimAt ? Look.lerp(1, m.pushDim, gone / m.pushDimAt) : Look.lerp(m.pushDim, 0, (gone - m.pushDimAt) / (1 - m.pushDimAt))
        case (.under, .pop):
            look.slide = -m.pushUnder * (1 - progress)
            look.opacity = progress
        }
        return look
    }

    public static func frozen(_ time: Double, role: Role, way: Way, reduce: Bool = false) -> Double {
        let shown = reduce ? Token.Motion.freeze(time, Token.Motion.reducedCurve) : Token.Motion.freeze(time, Token.Motion.smoothCurve)
        return (way == .push) == (role == .top) ? shown : 1 - shown
    }

    public func body(content: Content) -> some View {
        let look = Self.look(progress, role: role, way: way, reduce: tokens.reduceMotion)
        content
            .background(look.shade > 0 ? Token.Motion.slide(tokens.scheme) : Color.clear)
            .shadow(color: Token.Motion.slideShade(look.shade), radius: Token.Motion.Move.slideShadowRadius, x: Token.Motion.Move.slideShadowX)
            .moved(look)
    }
}

// value: M4, a sideways jump (⌘0, ⌘1 and the rest): the old page is gone in 130 ms
// before the new one shows, over 320 ms with nothing in its first 40 %.
public struct JumpTransition: Movement {
    public enum Role: Sendable { case leaving, arriving }

    public var progress: Double
    public var role: Role

    public init(progress: Double, role: Role) {
        self.progress = progress
        self.role = role
    }

    public static var pace: Token.Motion.Pace { .curve(Token.Motion.easeOutCurve) }

    public static func look(_ progress: Double, role: Role) -> Look {
        var look = Look()
        look.opacity = role == .leaving ? progress : Look.delayed(progress, by: Token.Motion.jumpHold)
        return look
    }

    // invariant: a jump's moment is counted on the arriving page's 320 ms.
    public static func frozen(_ time: Double, role: Role) -> Double {
        switch role {
        case .arriving: return Token.Motion.freeze(time, Token.Motion.easeOutCurve)
        case .leaving: return 1 - Token.Motion.freeze(time * Token.Motion.jumpInDuration / Token.Motion.jumpOutDuration, Token.Motion.easeOutCurve)
        }
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, role: role))
    }
}

// value: M6, the panel's frame follows its content between two heights.
public struct HeightMorph: Movement {
    public var progress: Double
    public var from: CGFloat
    public var to: CGFloat

    public init(progress: Double, from: CGFloat, to: CGFloat) {
        self.progress = progress
        self.from = from
        self.to = to
    }

    public static func height(_ progress: Double, from: CGFloat, to: CGFloat) -> CGFloat {
        CGFloat(Look.lerp(Double(from), Double(to), progress))
    }

    public static func frozen(_ time: Double, reduce: Bool) -> Double {
        reduce ? Token.Motion.freeze(time, Token.Motion.reducedCurve) : Token.Motion.freeze(time, Token.Motion.smoothCurve)
    }

    public func body(content: Content) -> some View {
        content.frame(height: Self.height(progress, from: from, to: to), alignment: .top).clipped()
    }
}

// value: M28, the footer's buttons rise and fade in with each page.
public struct FooterRise: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = progress
        if !reduce { look.y = Token.Motion.Move.footerRise * CGFloat(1 - progress) }
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, reduce: tokens.reduceMotion))
    }
}
