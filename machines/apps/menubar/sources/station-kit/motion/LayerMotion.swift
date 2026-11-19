import SwiftUI

// value: M20, a sheet unrolling from under the page header: its own header
// is there from the first frame, a mask lets the body down with a 6 pt
// drop, the scrim fades; it leaves on snappy.
public struct SheetReveal: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        var look = Look()
        if reduce {
            look.opacity = progress
        } else {
            look.reveal = progress
            look.y = -Token.Motion.sheetDrop * CGFloat(1 - progress)
        }
        return look
    }

    public static func frozen(_ time: Double, leaving: Bool = false, reduce: Bool = false) -> Double {
        if reduce { return Token.Motion.freeze(time, Token.Motion.reducedCurve) }
        return leaving ? 1 - Token.Motion.freeze(time, Token.Motion.snappyCurve) : Token.Motion.freeze(time, Token.Motion.smoothCurve)
    }

    public func body(content: Content) -> some View {
        let look = Self.look(progress, reduce: tokens.reduceMotion)
        content
            .mask(alignment: .top) { Unroll(share: look.reveal, least: Token.Motion.sheetHeader, radius: Token.Motion.sheetRadius) }
            .moved(look)
    }
}

// value: M20 and M21's scrim, faded with its layer.
public struct ScrimFade: Movement {
    public var progress: Double

    public init(progress: Double) { self.progress = progress }

    public func body(content: Content) -> some View {
        content.opacity(progress)
    }
}

// value: M21, M23, M24, a layer appearing in place: a popover from .94, a
// menu's preview from .97, an alert from 1.08; each leaves to its own
// slighter scale on snappy.
public struct PopAppear: Movement {
    public enum Kind: Sendable { case popover, menu, alert }

    public var progress: Double
    public var kind: Kind
    public var leaving: Bool
    public var anchor: UnitPoint
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double, kind: Kind = .popover, leaving: Bool = false, anchor: UnitPoint = .center) {
        self.progress = progress
        self.kind = kind
        self.leaving = leaving
        self.anchor = anchor
    }

    public static var pace: Token.Motion.Pace { .curve(Token.Motion.snappyCurve) }

    public static func from(_ kind: Kind, leaving: Bool) -> Double {
        let m = Token.Motion.Move.self
        switch kind {
        case .popover: return leaving ? m.popoverLeave : m.popoverFrom
        case .menu: return m.menuFrom
        case .alert: return leaving ? m.alertLeave : m.alertFrom
        }
    }

    public static func look(_ progress: Double, kind: Kind, leaving: Bool = false, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = progress
        if !reduce { look.scale = Look.scaled(Look.lerp(from(kind, leaving: leaving), 1, progress)) }
        return look
    }

    // invariant: an alert arrives on smooth; every layer leaves on snappy.
    public static func frozen(_ time: Double, kind: Kind, leaving: Bool = false, reduce: Bool = false) -> Double {
        if reduce { return Token.Motion.freeze(time, Token.Motion.reducedCurve) }
        if leaving { return 1 - Token.Motion.freeze(time, Token.Motion.snappyCurve) }
        return Token.Motion.freeze(time, kind == .alert ? Token.Motion.smoothCurve : Token.Motion.snappyCurve)
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, kind: kind, leaving: leaving, reduce: tokens.reduceMotion), anchor: anchor)
    }
}

// value: M22, Find: its layer fades on snappy, its field settles from 6 pt
// above on smooth.
public struct FindAppear: Movement {
    public var progress: Double
    public var field: Bool
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double, field: Bool = false) {
        self.progress = progress
        self.field = field
    }

    public static func look(_ progress: Double, field: Bool, reduce: Bool) -> Look {
        var look = Look()
        look.opacity = progress
        if field && !reduce { look.y = -Token.Motion.Move.findDrop * CGFloat(1 - progress) }
        return look
    }

    public static func frozen(_ time: Double, field: Bool, reduce: Bool = false) -> Double {
        if reduce { return Token.Motion.freeze(time, Token.Motion.reducedCurve) }
        return Token.Motion.freeze(time, field ? Token.Motion.smoothCurve : Token.Motion.snappyCurve)
    }

    public func body(content: Content) -> some View {
        content.moved(Self.look(progress, field: field, reduce: tokens.reduceMotion))
    }
}

// value: M30, the HUD over its 2.6 s: it opens from a sliver under the
// menu bar in its first 10 %, holds, and folds back from 85 %.
public struct HUDReveal: Movement {
    public var progress: Double
    @Environment(\.stationTokens) private var tokens

    public init(progress: Double) { self.progress = progress }

    public static func look(_ progress: Double, reduce: Bool) -> Look {
        let open = Token.Motion.hudOpen, close = Token.Motion.hudClose
        let shown = progress < open ? progress / open : progress <= close ? 1 : max(1 - (progress - close) / (1 - close), 0)
        var look = Look()
        look.opacity = shown
        if !reduce { look.reveal = shown }
        return look
    }

    public func body(content: Content) -> some View {
        let look = Self.look(progress, reduce: tokens.reduceMotion)
        content
            .mask(alignment: .top) { HUDMask(share: look.reveal) }
            .moved(look)
    }
}

struct HUDMask: Shape {
    var share: Double

    var animatableData: Double {
        get { share }
        set { share = newValue }
    }

    func path(in rect: CGRect) -> Path {
        let s = min(max(share, 0), 1)
        let side = rect.width * Token.Motion.hudSide * (1 - s)
        let shown = CGRect(x: rect.minX + side, y: rect.minY, width: rect.width - side * 2, height: rect.height * s)
        return Path(roundedRect: shown, cornerRadius: min(Token.Metric.HUD.corner, shown.height / 2), style: .continuous)
    }
}
