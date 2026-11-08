import SwiftUI

// value: M5, the segment's thumb sliding from one segment's measured frame
// to another's: its x and width, never a matched geometry, which cannot
// be drawn at a progress.
public struct SegmentThumb: Movement {
    public var progress: Double
    public var from: CGRect
    public var to: CGRect

    public init(progress: Double, from: CGRect, to: CGRect) {
        self.progress = progress
        self.from = from
        self.to = to
    }

    public static var pace: Token.Motion.Pace { .curve(Token.Motion.snappyCurve) }

    public static func frame(_ progress: Double, from: CGRect, to: CGRect) -> CGRect {
        let s = CGFloat(progress)
        return CGRect(x: from.minX + (to.minX - from.minX) * s, y: from.minY, width: from.width + (to.width - from.width) * s, height: from.height)
    }

    public func body(content: Content) -> some View {
        let frame = Self.frame(progress, from: from, to: to)
        content.frame(width: frame.width, height: frame.height).offset(x: frame.minX, y: frame.minY)
    }
}

// value: M5, the segment's body arriving 10 pt from the side it travels to.
public struct SegmentSlide: Movement {
    public var progress: Double
    public var forward: Bool
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double, forward: Bool = true) {
        self.progress = progress
        self.forward = forward
    }

    public static var pace: Token.Motion.Pace { .curve(Token.Motion.snappyCurve) }

    public static func look(_ progress: Double, forward: Bool, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = progress
        if !reduce { look.x = Token.Motion.Move.segmentShift * CGFloat(1 - progress) * (forward ? 1 : -1) }
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, forward: forward, reduce: tokens.reduceMotion))
    }
}

// value: M15, one ring of the new colour, from the mark's size to 2.6
// times it, fading out; none under Reduce Motion.
public struct Ping: Movement {
    public var progress: Double
    public var ink: Color
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double, ink: Color) {
        self.progress = progress
        self.ink = ink
    }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.scale = Look.scaled(Look.lerp(1, Token.Motion.Move.pingTo, progress))
        look.opacity = reduce ? 0 : Look.lerp(Token.Motion.Move.pingShare, 0, progress)
        return look
    }

    public func body(content: Content) -> some View {
        content.overlay { Circle().fill(ink).moved(Self.look(progress, reduce: tokens.reduceMotion)) }
    }
}

// value: M17, a placeholder's shapes with a diagonal shimmer crossing them
// once per 1.2 s; still under Reduce Motion.
public struct Skeleton: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static var pace: Token.Motion.Pace { .curve(UnitCurve.linear) }

    // invariant: the band's centre, as a share of the width from the
    // middle: from past the trailing edge to past the leading one.
    public static func band(_ progress: Double, reduce: Bool) -> Double? {
        reduce ? nil : Token.Motion.shimmerFrom * (1 - progress * 2)
    }

    public func body(content: Content) -> some View {
        let band = Self.band(progress, reduce: tokens.reduceMotion)
        content
            .redacted(reason: .placeholder)
            .overlay {
                if let band {
                    LinearGradient(stops: Token.Motion.shimmerStops.map { Gradient.Stop(color: Token.Motion.shimmer($0.share), location: $0.at) },
                                   startPoint: .topLeading, endPoint: .bottomTrailing)
                        .visualEffect { view, frame in
                            view.scaleEffect(x: Token.Motion.shimmerBand).offset(x: frame.size.width * band)
                        }
                        .mask { content.redacted(reason: .placeholder) }
                        .allowsHitTesting(false)
                }
            }
    }
}

// value: M26, a tile's "+1" over its one second: it rises from 6 pt below
// into view by a quarter, then on to 12 pt above as it fades.
public struct PlusOne: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        let m = Token.Motion.self
        var look = Look()
        if progress < m.plusOnePeak {
            let s = progress / m.plusOnePeak
            look.opacity = s
            look.y = m.plusOneRise * CGFloat(1 - s)
        } else {
            let s = (progress - m.plusOnePeak) / (1 - m.plusOnePeak)
            look.opacity = 1 - s
            look.y = -m.plusOneLift * CGFloat(s)
        }
        if reduce { look.y = 0 }
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, reduce: tokens.reduceMotion))
    }
}

// value: M27, a chart re-growing from its base when its window changes.
public struct ChartRegrow: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = Look.lerp(Token.Motion.Move.regrowShare, 1, progress)
        if !reduce { look.scale = Look.scaled(1, tall: Look.lerp(Token.Motion.Move.regrowFrom, 1, progress)) }
        return look
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, reduce: tokens.reduceMotion), anchor: .bottom)
    }
}

// value: M31, a fill easing from one share to another: a VM's birth bar,
// a ring, a time left, the memory slider.
public struct ProgressEase: Movement {
    public var progress: Double
    public var from: Double
    public var to: Double

    public init(progress: Double, from: Double, to: Double) {
        self.progress = progress
        self.from = from
        self.to = to
    }

    public static func share(_ progress: Double, from: Double, to: Double) -> Double { Look.lerp(from, to, progress) }

    public static func frozen(_ time: Double, reduce: Bool) -> Double {
        reduce ? Token.Motion.freeze(time, Token.Motion.reducedCurve) : Token.Motion.freeze(time, Token.Motion.smoothCurve)
    }

    public func body(content: Content) -> some View {
        content.mask(alignment: .leading) { Unroll(share: Self.share(progress, from: from, to: to), edge: .leading) }
    }
}
