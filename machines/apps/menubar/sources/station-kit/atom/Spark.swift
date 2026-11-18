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
