import Foundation

public enum ChartScale {
    public static func floor(unit: String?) -> Double {
        unit == "%" ? Token.Metric.Chart.percentFloor : Token.Metric.Chart.rateFloor
    }

    public static func top(_ series: [[Double]], unit: String?) -> Double {
        let high = series.flatMap { $0 }.max() ?? 0
        return max(high * Token.Metric.Chart.headroom, floor(unit: unit))
    }

    public static func peak(_ points: [Double]) -> Int? {
        guard let high = points.max() else { return nil }
        return points.firstIndex(of: high)
    }

    public static func spark(_ points: [Double]) -> ClosedRange<Double> {
        let c = Token.Metric.Chart.self
        guard let low = points.min(), let high = points.max() else { return 0...1 }
        let span = max(high - low, max(high, 1) * c.sparkSpan)
        let mid = (high + low) / 2
        return (mid - span / 2 * c.sparkPad)...(mid + span / 2 * c.sparkPad)
    }

    public static func eased(_ points: [Double]) -> [Double] {
        guard points.count > 2 else { return points }
        return points.indices.map { i in
            i == 0 || i == points.count - 1 ? points[i] : (points[i - 1] + 2 * points[i] + points[i + 1]) / 4
        }
    }

    public static func edge(_ x: CGFloat, width: CGFloat) -> CGFloat {
        let inset = min(Token.Metric.Chart.chipEdge, width / 2)
        return min(max(x, inset), width - inset)
    }

    public static func say(_ point: Double, unit: String?) -> String {
        if unit == "%" { return Format.show(percent(point)) }
        let number = String(format: point < 10 ? "%.1f" : "%.0f", point)
        guard let unit, !unit.isEmpty else { return number }
        return "\(number)\(Format.glued)\(unit)"
    }

    static func percent(_ point: Double) -> Value {
        var v = Value(kind: "percent")
        v.n = Int64((point.rounded() * Token.Live.tenths))
        return v
    }
}
