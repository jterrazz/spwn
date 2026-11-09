import SwiftUI

extension Token.Motion {
    // value: how a movement is paced from its start to its end: one of the
    // prototype's curves, or the success spring.
    public enum Pace: Sendable {
        case curve(UnitCurve)
        case bouncy

        public func at(_ progress: Double) -> Double {
            let t = min(max(progress, 0), 1)
            switch self {
            case .curve(let curve): return curve.value(at: t)
            case .bouncy: return Token.Motion.bouncy(at: t)
            }
        }
    }

    // invariant: a movement drawn at a moment of its run: the share of its
    // way it has made at that share of its time, through the curve the live
    // app animates it on.
    public static func freeze(_ progress: Double, _ curve: UnitCurve) -> Double {
        curve.value(at: min(max(progress, 0), 1))
    }

    public static func freeze(_ progress: Double, _ pace: Pace) -> Double { pace.at(progress) }

    public static let snappyDuration = 0.20
    public static let smoothDuration = 0.34

    public static let reducedCurve = UnitCurve.easeInOut
    public static let easeOutCurve = UnitCurve.easeOut

    public static let jumpOut = Animation.easeOut(duration: jumpOutDuration)
    public static let jumpIn = Animation.easeOut(duration: jumpInDuration)
    public static let jumpOutDuration = 0.13
    public static let jumpInDuration = 0.32
    public static let jumpHold = 0.4

    public static let fresh = Animation.easeOut(duration: freshDuration)
    public static let freshDuration = 2.6
    public static let freshHold = 0.35
    public static let freshShare = 0.16

    public static let plusOne = Animation.timingCurve(smoothCurve, duration: 1)
    public static let plusOneRise: CGFloat = 6
    public static let plusOneLift: CGFloat = 12
    public static let plusOnePeak = 0.25

    public static let hud = Animation.timingCurve(smoothCurve, duration: freshDuration)
    public static let hudOpen = 0.1
    public static let hudClose = 0.85
    public static let hudSide = 0.4

    public static let skeleton = Animation.linear(duration: skeletonPeriod).repeatForever(autoreverses: false)
    public static let skeletonPeriod = 1.2
    public static let shimmerFrom = 1.2
    public static let shimmerBand = 2.2
    public static let shimmerStops: [(at: Double, share: Double)] = [(0.30, 0), (0.50, 0.12), (0.70, 0)]

    public static let sheet = Animation.timingCurve(smoothCurve, duration: 0.34).delay(sheetDelay)
    public static let sheetDelay = 0.04
    public static let sheetHeader: CGFloat = 46
    public static let sheetDrop: CGFloat = 6
    public static let sheetRadius: CGFloat = 14

    public static let reducedLayer = Animation.easeInOut(duration: 0.15)
    public static let drawOn = Animation.timingCurve(smoothCurve, duration: 0.34).delay(drawOnDelay)
    public static let drawOnDelay = 0.1
    public static let drawOnShare = drawOnDelay / (drawOnDelay + 0.34)
}

// value: SPEC A8's distances, scales and shares, each named for the
// movement it belongs to.
extension Token.Motion {
    public enum Move {
        public static let panelFrom = 0.96
        public static let panelClose = 0.98

        public static let pushUnder = 0.30
        public static let pushDimAt = 0.30
        public static let pushDim = 0.25
        public static let popFade = 0.6
        public static let slideShadow = 0.16
        public static let slideShadowRadius: CGFloat = 24
        public static let slideShadowX: CGFloat = -10

        public static let segmentShift: CGFloat = 10

        public static let foldSettle: CGFloat = 4
        public static let foldFade = 0.12 / 0.34
        public static let chevronTurn = 180.0

        public static let heroDrop: CGFloat = 3
        public static let heroRowsDelay = 0.06 / 0.34
        public static let heroGlyphFrom = 0.6

        public static let leaveShift: CGFloat = 10
        public static let landFrom = 0.97
        public static let landDrop: CGFloat = 3

        public static let bornFrom = 0.3

        public static let pingTo = 2.6
        public static let pingShare = 0.55

        public static let popoverFrom = 0.94
        public static let popoverLeave = 0.97
        public static let menuFrom = 0.97
        public static let alertFrom = 1.08
        public static let alertLeave = 0.96

        public static let findDrop: CGFloat = 6
        public static let hoverSlide: CGFloat = 4
        public static let regrowFrom = 0.92
        public static let regrowShare = 0.3
        public static let footerRise: CGFloat = 4
    }

    public static func slide(_ scheme: ColorScheme) -> Color { scheme == .dark ? Token.rgba(30, 31, 38, 1) : Token.rgba(232, 236, 248, 1) }
    public static func slideShade(_ share: Double) -> Color { Token.rgba(0, 0, 0, Move.slideShadow * share) }
    public static func shimmer(_ share: Double) -> Color { Token.rgba(255, 255, 255, share) }
}

// value: the motion sheet: every movement drawn at one moment of its run,
// on a stage of its own.
extension Token.Motion {
    public enum Sheet {
        public static let moment = 0.35
        public static let columns = 3
        public static let stage = CGSize(width: 96, height: 72)
        public static let line: CGFloat = 6
        public static let lineWidths: [CGFloat] = [48, 32, 24]
        public static let gap: CGFloat = 4
        public static let inset: CGFloat = 6
        public static let radius: CGFloat = 6
        public static let fillFrom = 0.2
        public static let fillTo = 0.8
        public static let morphFrom = 0.5
        public static let segments = 2
        public static let bars: [Double] = [0.4, 0.7, 0.55, 0.9, 0.6, 0.8]
    }
}
