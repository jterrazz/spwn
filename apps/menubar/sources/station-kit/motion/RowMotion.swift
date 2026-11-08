import SwiftUI

// value: M7, a fold opening in place: its height opens on smooth, its rows
// fade in over the first 120 ms and settle from 4 pt above.
public struct Fold: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.height = progress
        look.opacity = reduce ? progress : Look.faded(progress, over: Token.Motion.Move.foldFade)
        if !reduce { look.y = -Token.Motion.Move.foldSettle * CGFloat(1 - progress) }
        return look
    }

    public static func frozen(_ time: Double, reduce: Bool) -> Double {
        reduce ? Token.Motion.freeze(time, Token.Motion.reducedCurve) : Token.Motion.freeze(time, Token.Motion.smoothCurve)
    }

    public func body(content: Content) -> some View {
        let look = Self.look(progress, reduce: tokens.reduceMotion)
        Collapse(share: look.height) { content.moved(look) }.clipped()
    }
}

// value: M7's chevron, a half turn as its fold opens.
public struct FoldChevron: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.turn = reduce ? (progress < 1 ? 0 : Token.Motion.Move.chevronTurn) : Token.Motion.Move.chevronTurn * progress
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, reduce: tokens.reduceMotion))
    }
}

// value: M8, the hero morphing in place: its top cross-fades from 3 pt
// above, its rows follow 60 ms later, a red glyph arrives at .6 scale on
// the success spring.
public struct HeroMorph: Movement {
    public enum Part: Sendable { case top, rows, glyph }

    public var progress: Double
    public var part: Part
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double, part: Part) {
        self.progress = progress
        self.part = part
    }

    public static func look(_ progress: Double, part: Part, reduce: Bool) -> Look {
        var look = Look()
        switch part {
        case .top:
            look.opacity = progress
            if !reduce { look.y = -Token.Motion.Move.heroDrop * CGFloat(1 - progress) }
        case .rows:
            look.opacity = reduce ? progress : Look.delayed(progress, by: Token.Motion.Move.heroRowsDelay)
        case .glyph:
            if reduce { look.opacity = min(progress, 1) } else { look.scale = Look.scaled(Look.lerp(Token.Motion.Move.heroGlyphFrom, 1, progress)) }
        }
        return look
    }

    public static func frozen(_ time: Double, part: Part, reduce: Bool) -> Double {
        if reduce { return Token.Motion.freeze(time, Token.Motion.reducedCurve) }
        return part == .glyph ? Token.Motion.freeze(time, .bouncy) : Token.Motion.freeze(time, Token.Motion.smoothCurve)
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, part: part, reduce: tokens.reduceMotion))
    }
}

// value: M9, a row leaving: it collapses to no height, slides 10 pt and
// fades; at a removal's moment t it is at 1 - pace(t).
public struct RowLeave: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = progress
        look.height = progress
        if !reduce { look.x = Token.Motion.Move.leaveShift * CGFloat(1 - progress) }
        return look
    }

    public static func frozen(_ time: Double, reduce: Bool) -> Double {
        1 - (reduce ? Token.Motion.freeze(time, Token.Motion.reducedCurve) : Token.Motion.freeze(time, Token.Motion.smoothCurve))
    }

    public func body(content: Content) -> some View {
        let look = Self.look(progress, reduce: tokens.reduceMotion)
        Collapse(share: look.height) { content.moved(look) }.clipped()
    }
}

// value: M10, a row landing from .97 scale and 3 pt above.
public struct RowLand: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = progress
        if !reduce {
            look.scale = Look.scaled(Look.lerp(Token.Motion.Move.landFrom, 1, progress))
            look.y = -Token.Motion.Move.landDrop * CGFloat(1 - progress)
        }
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, reduce: tokens.reduceMotion), anchor: .top)
    }
}

// value: M11, a fresh row's flash of the accent, held for 35 % of its
// 2.6 s and faded out; a colour, so Reduce Motion keeps it.
public struct FreshTint: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static var pace: Token.Motion.Pace { .curve(Token.Motion.easeOutCurve) }

    public static func look(_ progress: Double) -> Look {
        var look = Look()
        let hold = Token.Motion.freshHold
        look.tint = progress <= hold ? Token.Motion.freshShare : Look.lerp(Token.Motion.freshShare, 0, (progress - hold) / (1 - hold))
        return look
    }

    public func body(content: Content) -> some View {
        content.background(tokens.color.accent.opacity(Self.look(progress).tint))
    }
}

// value: M12, the success check born from .3 scale on the success spring.
public struct Born: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static var pace: Token.Motion.Pace { .bouncy }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = min(progress, 1)
        if !reduce { look.scale = Look.scaled(Look.lerp(Token.Motion.Move.bornFrom, 1, progress)) }
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, reduce: tokens.reduceMotion))
    }
}

// value: M18, values that land fade in where they were stale.
public struct Reveal: Movement {
    public var progress: Double

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double) -> Look {
        var look = Look()
        look.opacity = progress
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress))
    }
}

// value: M19, an empty state's glyph drawing itself from its leading edge
// after 100 ms; under Reduce Motion it is shown whole.
public struct GlyphDraw: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        if !reduce { look.reveal = Look.delayed(progress, by: Token.Motion.drawOnShare) }
        return look
    }

    public func body(content: Content) -> some View {
        content.mask(alignment: .leading) { Unroll(share: Self.look(progress, reduce: tokens.reduceMotion).reveal, edge: .leading) }
    }
}

// value: M25, a row's quick action fading in on hover and sliding 4 pt;
// under Reduce Motion only the tint moves.
public struct HoverReveal: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static var pace: Token.Motion.Pace { .curve(Token.Motion.snappyCurve) }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = progress
        if !reduce { look.x = Token.Motion.Move.hoverSlide * CGFloat(1 - progress) }
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, reduce: tokens.reduceMotion))
    }
}
