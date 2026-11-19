import Charts
import SwiftUI

public struct Spark: View {
    let points: [Double]
    let height: CGFloat
    let tint: Color?
    let hot: Bool
    @Environment(\.stationTokens) private var tokens

    public init(_ points: [Int], height: CGFloat = Token.Metric.Chart.spark.height, tint: Color? = nil, hot: Bool = false) {
        self.init(points.map(Double.init), height: height, tint: tint, hot: hot)
    }

    public init(_ points: [Double], height: CGFloat = Token.Metric.Chart.spark.height, tint: Color? = nil, hot: Bool = false) {
        self.points = points
        self.height = height
        self.tint = tint
        self.hot = hot
    }

    public var body: some View {
        let c = Token.Metric.Chart.self
        let ink = hot ? tokens.color.orange : tint ?? tokens.color.accent
        let shown = Array(ChartScale.eased(points).enumerated())
        let last = Double(max(points.count - 1, 1))
        let pad = last * Double(c.sparkDot) / Double(c.spark.width)
        Chart {
            ForEach(shown, id: \.offset) { i, v in
                LineMark(x: .value("t", Double(i)), y: .value("v", v))
                    .interpolationMethod(.monotone)
                    .lineStyle(StrokeStyle(lineWidth: c.line, lineCap: .round, lineJoin: .round))
                    .foregroundStyle(ink)
            }
            if let end = shown.last {
                PointMark(x: .value("t", Double(end.offset)), y: .value("v", end.element))
                    .symbol { Circle().fill(ink).frame(width: c.sparkDot, height: c.sparkDot) }
            }
        }
        .chartXAxis(.hidden)
        .chartYAxis(.hidden)
        .chartLegend(.hidden)
        .chartYScale(domain: ChartScale.spark(points))
        .chartXScale(domain: (-pad / 2)...(last + pad))
        .chartPlotStyle { $0.frame(maxWidth: .infinity) }
        .frame(height: height)
        .accessibilityHidden(true)
    }
}

extension Value {
    public var points: [Int] {
        guard kind == "spark" else { return [] }
        return (items ?? []).dropFirst().map { Int($0.n ?? 0) }
    }
}

// invariant: a row's CPU spark, 40 by 14 (This Mac's measures 72 by 16),
// drawn as it fills: one slot per reading the engine keeps, the last point
// on the right edge, so a series still filling is a short line, one point
// its dot, none nothing; the line in t4, its end the accent's dot.
public struct RowSpark: View {
    let points: [Double]
    let size: CGSize
    @Environment(\.stationTokens) private var tokens

    public init(_ points: [Int], size: CGSize = Token.Metric.Chart.rowSpark) {
        self.points = Self.kept(points).map(Double.init)
        self.size = size
    }

    public static func kept(_ points: [Int]) -> [Int] { Array(points.suffix(Token.Metric.Chart.rowSparkPoints)) }

    // value: the slot each of n points takes, the last at the right edge.
    public static func slots(_ n: Int) -> [Int] {
        let first = Token.Metric.Chart.rowSparkPoints - min(n, Token.Metric.Chart.rowSparkPoints)
        return (0..<min(n, Token.Metric.Chart.rowSparkPoints)).map { first + $0 }
    }

    public var body: some View {
        let c = Token.Metric.Chart.self
        let last = Double(c.rowSparkPoints - 1)
        let pad = last * Double(c.sparkDot) / Double(size.width)
        let shown = Array(zip(Self.slots(points.count), ChartScale.eased(points)))
        Chart {
            ForEach(Array(shown.enumerated()), id: \.offset) { _, p in
                LineMark(x: .value("t", Double(p.0)), y: .value("v", p.1))
                    .interpolationMethod(.monotone)
                    .lineStyle(StrokeStyle(lineWidth: c.line, lineCap: .round, lineJoin: .round))
                    .foregroundStyle(tokens.color.t4)
            }
            if let end = shown.last {
                PointMark(x: .value("t", Double(end.0)), y: .value("v", end.1))
                    .symbol { Circle().fill(tokens.color.accent).frame(width: c.sparkDot, height: c.sparkDot) }
            }
        }
        .chartXAxis(.hidden)
        .chartYAxis(.hidden)
        .chartLegend(.hidden)
        .chartYScale(domain: ChartScale.spark(points))
        .chartXScale(domain: (-pad / 2)...(last + pad / 2))
        .frame(width: size.width, height: size.height)
        .accessibilityHidden(true)
    }
}
