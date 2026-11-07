import SwiftUI

extension Token.Metric.Chart {
    public static let headroom = 1.1
    public static let percentFloor = 10.0
    public static let rateFloor = 0.1
    public static let sparkSpan = 0.25
    public static let sparkPad = 1.15
    public static let line: CGFloat = 1.5
    public static let anomalyLine: CGFloat = 2.2
    public static let areaShare = 0.16
    public static let sentShare = 0.22
    public static let receivedShare = 0.24
    public static let endDot: CGFloat = 5.2
    public static let peakDot: CGFloat = 4.8
    public static let peakStroke: CGFloat = 1.6
    public static let sparkDot: CGFloat = 3.6
    public static let rule: CGFloat = 1
    public static let ruleDash: [CGFloat] = [2, 2]
    public static let gridDash: [CGFloat] = [2, 3]
    public static let barShare = 0.8
    public static let barSlot = 0.5
    public static let barCorner: CGFloat = 1.6
    public static let barLeast = 2.0
    public static let dimShare = 0.35
    public static let chipPadding = EdgeInsets(top: 1, leading: 5, bottom: 1, trailing: 5)
    public static let chipRadius: CGFloat = 5
    public static let chipEdge: CGFloat = 44
    public static let chipTint = 0.15
    public static let calloutRadius: CGFloat = 6
    public static let calloutPadding = EdgeInsets(top: 1, leading: 6, bottom: 1, trailing: 6)
    public static let gridLabel: CGFloat = 10
    public static let gridRow: CGFloat = 13
    public static let ends: CGFloat = 11
    public static let endsGap: CGFloat = 4
    public static let uptimeShare = 0.34
    public static let uptimeMarkShare = 0.70
    public static let plainBar: CGFloat = 4
    public static let memoryBar: CGFloat = 5
    public static let capacityBar: CGFloat = 6
    public static let vmShares = [0.62, 0.46]
    public static let warn = 0.85
    public static let critical = 0.95
}

extension Token.Shadow {
    public static let callout = Token.Shadow(color: Token.rgba(0, 0, 0, 0.25), radius: 4, y: 1)
}
