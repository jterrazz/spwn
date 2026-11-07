import Charts
import SwiftUI

// invariant: a measure's graph over its window: its points 0 to 100,
// oldest first, a 1.5 pt catmull-rom line over an area fading to nothing,
// no axes, as wide as it is given.
public struct Graph: View {
    let points: [Int]
    let height: CGFloat
    let tint: Color?
    let fit: Bool
    @Environment(\.stationPalette) private var palette

    public init(_ points: [Int], height: CGFloat = Token.Space.graphHeight, tint: Color? = nil, fit: Bool = false) {
        self.points = points
        self.height = height
        self.tint = tint
        self.fit = fit
    }

    // value: the graph's scale: 0 to 100, its top brought down to a low
    // series' (never under 20), so a measure that stays low (a swap of
    // 7 %) is still drawn as an area; for a sparkline, the whole 0 to 100,
    // so a meter's line sits at its share (a disk of 12 % reads low).
    var domain: ClosedRange<Int> {
        guard !fit, let hi = points.max() else { return 0...100 }
        return 0...min(max(hi * 2, 20), 100)
    }

    public var body: some View {
        let ink = tint ?? palette.accent
        Chart(Array(points.enumerated()), id: \.offset) { i, p in
            AreaMark(x: .value("t", i), yStart: .value("floor", domain.lowerBound), yEnd: .value("v", min(max(p, 0), 100)))
                .interpolationMethod(.catmullRom)
                .foregroundStyle(LinearGradient(colors: [ink.opacity(Token.Fade.graphTop), ink.opacity(Token.Fade.graphFloor)], startPoint: .top, endPoint: .bottom))
            LineMark(x: .value("t", i), y: .value("v", min(max(p, 0), 100)))
                .interpolationMethod(.catmullRom)
                .lineStyle(StrokeStyle(lineWidth: Token.Stroke.graph, lineCap: .round, lineJoin: .round))
                .foregroundStyle(ink)
        }
        .chartXAxis(.hidden)
        .chartYAxis(.hidden)
        .chartLegend(.hidden)
        .chartYScale(domain: domain)
        .chartXScale(domain: 0...max(points.count - 1, 1))
        .chartPlotStyle { $0.frame(maxWidth: .infinity) }
        .frame(height: height)
    }
}

extension Value {
    // value: the points a spark value carries after its lead value.
    public var points: [Int] {
        guard kind == "spark" else { return [] }
        return (items ?? []).dropFirst().map { Int($0.n ?? 0) }
    }
}

// invariant: a measure as the references' bar meter: one rounded bar a
// point, oldest first, grey, the latest `accented` in the accent; Swift
// Charts' bars, axes hidden, as wide as it is given.
public struct Bars: View {
    let points: [Int]
    let height: CGFloat
    let accented: Int
    let tint: Color?
    let chart: Bool
    var waiting = false
    var full = false
    var ceiling: Int? = nil
    @Environment(\.stationPalette) private var palette

    public init(_ points: [Int], height: CGFloat = Token.Tile.meter, accented: Int = 1, tint: Color? = nil, chart: Bool = false) {
        self.points = points
        self.height = height
        self.accented = accented
        self.tint = tint
        self.chart = chart
    }

    // value: a bar's height, 0 to 1, over the band its points move in, so
    // a steady series still shows its swing; a level one, half.
    func level(_ p: Int) -> Double {
        guard !waiting else { return 0.25 }
        guard !full else { return 1 }
        if let ceiling { return min(max(Double(p) / Double(max(ceiling, 1)), 0.06), 1) }
        if chart, let hi = points.max() { return max(Double(p) / Double(max(hi, 1)), 0.06) }
        guard let hi = points.max(), let low = points.min(), hi != low else { return 0.5 }
        let top = Double(max(hi, 1)), lo = Double(low)
        let floor = max(lo - max(top - lo, top / 10), 0)
        return max((Double(p) - floor) / max(top - floor, 1), 0.1)
    }

    // value: a measure's bars against a share's scale, 0 to 100 brought
    // down to a low series' (never under 20), so a steady measure keeps
    // its level instead of filling the band.
    public static func share(_ points: [Int], height: CGFloat) -> Bars {
        var bars = Bars(points, height: height, chart: true)
        bars.ceiling = min(max((points.max() ?? 0) * 2, 20), 100)
        return bars
    }

    // value: the meter of a measure not read yet: its bars low and level,
    // none accented, so a tile waiting keeps its shape.
    public static func pending(height: CGFloat = Token.Tile.meter) -> Bars {
        var bars = Bars(Array(repeating: 1, count: 12), height: height, accented: 0)
        bars.waiting = true
        return bars
    }

    // value: the meter of a figure with no series: `slots` level bars, the
    // last `lit` of them in the tint, every bar at full height, so a count reads as a filled share.
    public static let slots = 12

    public static func meter(_ lit: Int, of count: Int, height: CGFloat = Token.Tile.meter, tint: Color? = nil) -> Bars {
        var bars = Bars(Array(repeating: 1, count: count), height: height, accented: min(max(lit, 0), count), tint: tint)
        bars.full = true
        return bars
    }

    public var body: some View {
        let ink = tint ?? palette.accent
        let recent = points.count - accented
        Chart(Array(points.enumerated()), id: \.offset) { i, p in
            BarMark(x: .value("t", "\(i)"), yStart: .value("floor", 0), yEnd: .value("v", level(p)), width: .ratio(chart ? Token.Bar.chartShare : Token.Bar.sparkShare))
                .clipShape(RoundedRectangle(cornerRadius: chart ? Token.Bar.chartCorner : Token.Bar.sparkCorner, style: .continuous))
                .foregroundStyle(i >= recent ? AnyShapeStyle(ink) : waiting ? Token.Ink.track : Token.Ink.idle)
        }
        .chartXAxis(.hidden)
        .chartYAxis(.hidden)
        .chartLegend(.hidden)
        .chartYScale(domain: 0...1)
        .chartPlotStyle { $0.frame(maxWidth: .infinity) }
        .frame(height: height)
    }
}
