import Foundation
import SwiftUI

public struct ChartSheet: View {
    public enum Family: String, CaseIterable, Sendable { case lines, bars, small }

    let family: Family
    @Environment(\.stationTokens) private var tokens

    public init(_ family: Family) { self.family = family }

    static func wave(_ n: Int, base: Double, swing: Double, seed: Double, per: Double = 1) -> [Double] {
        (0..<n).map { i in
            let t = Double(i), s = seed / 10
            return max(base + swing * sin(t / 3.1 + s) + swing / 2 * sin(t / 1.3 + s * 2), 0) / per
        }
    }

    static let hour: [Double] = {
        var p = wave(60, base: 38, swing: 5, seed: 4)
        for (k, v) in [(27, 58.0), (28, 66), (29, 71), (30, 64), (31, 52)] { p[k] = v }
        return p
    }()

    static let day: [Double] = {
        var p = wave(48, base: 34, swing: 4, seed: 12)
        for (k, v) in [(18, 72.0), (19, 94), (20, 88), (21, 41)] { p[k] = v }
        return p
    }()

    static let quiet = wave(72, base: 40, swing: 15, seed: 21, per: 10)

    static let sent = wave(40, base: 35, swing: 12, seed: 7, per: 100)
    static let received = wave(40, base: 45, swing: 25, seed: 26, per: 100).enumerated().map { $0.offset == 23 ? 1 : $0.element }

    static func value(percent: Double) -> Value {
        var v = Value(kind: "percent")
        v.n = Int64(percent * Token.Live.tenths)
        return v
    }

    static func chart(_ kind: ActivityCharts.Kind, _ series: [[Double]], unit: String, ends: [String], peak at: String? = nil, anomaly: ClosedRange<Int>? = nil) -> PageChart {
        var chart = PageChart()
        chart.kind = kind.rawValue
        chart.series = series.map { points in
            var s = Series()
            s.points = points
            s.unit = unit
            return s
        }
        chart.ends = ends
        if let at, let lead = series.first, let i = ChartScale.peak(lead) {
            var peak = Peak()
            peak.index = i
            peak.value = value(percent: lead[i].rounded())
            peak.at = at
            chart.peak = peak
        }
        if let anomaly {
            var a = Anomaly()
            a.from = anomaly.lowerBound
            a.to = anomaly.upperBound
            chart.anomaly = a
        }
        return chart
    }

    static let hourEnds = ["1 hr ago", "now"]
    static let dayEnds = ["yesterday 14:30", "now"]

    static func day(_ i: Int, _ mark: String) -> Day {
        var d = Day()
        d.date = "2026-10-\(String(format: "%02d", i + 1))"
        d.mark = mark
        d.tip = "\(i + 1) Oct\(Token.Glyph.sep)\(mark)"
        return d
    }

    static let days: [Day] = (0..<30).map { i in day(i, i == 9 || i == 21 ? "restart" : i == 29 ? "down" : i < 3 ? "none" : "ok") }

    static let agent: Glyph = {
        var g = Glyph()
        g.kind = "agent"
        return g
    }()

    static func process(_ label: String, glyph kind: String, detail: String? = nil, cpu: Double, nav: String? = nil) -> Item {
        var row = Row()
        row.id = label
        row.label = label
        var g = Glyph()
        g.kind = kind
        row.glyph = g
        row.detail = detail.map { Value(kind: "text", text: $0) }
        row.cells = [value(percent: cpu)]
        row.nav = nav
        return Item(row: row)
    }

    static let processes: Section = {
        var s = Section()
        var name = Column()
        name.key = "name"
        var cpu = Column()
        cpu.key = TopProcesses.column
        s.columns = [name, cpu]
        s.items = [
            process("mds_stores", glyph: "symbol:cpu", cpu: 2),
            process("infra", glyph: "server", detail: "8 cores", cpu: 15, nav: "menubar.machine:infra"),
            process("linux-3", glyph: "agent", detail: "4 cores", cpu: 52, nav: "menubar.machine:linux-3"),
            process("review-bot", glyph: "agent", detail: "4 cores", cpu: 2, nav: "menubar.machine:review-bot"),
        ]
        return s
    }()

    func card<Content: View>(@ViewBuilder _ content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) { content() }
            .padding(Token.Metric.Card.padding)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(RoundedRectangle(cornerRadius: Token.Metric.Card.radius, style: .continuous).fill(tokens.color.card))
    }

    var lines: some View {
        VStack(spacing: Token.Metric.Panel.blocks) {
            card { ActivityCharts(Self.chart(.line, [Self.hour], unit: "%", ends: Self.hourEnds, peak: "30 min ago")) }
            card { ActivityCharts(Self.chart(.line, [Self.day], unit: "%", ends: Self.dayEnds, peak: "03:10", anomaly: 18...20)) }
            card { ActivityCharts(Self.chart(.line, [Self.hour], unit: "%", ends: Self.hourEnds, peak: "30 min ago"), scrub: 36) }
            card { ActivityCharts(Self.chart(.line, [Self.quiet], unit: "%", ends: ["6 min ago", "now"])) }
        }
    }

    var bars: some View {
        VStack(spacing: Token.Metric.Panel.blocks) {
            card { ActivityCharts(Self.chart(.bars, [Self.day], unit: "%", ends: Self.dayEnds, peak: "03:10")) }
            card { ActivityCharts(Self.chart(.bars, [Self.day], unit: "%", ends: Self.dayEnds, peak: "03:10", anomaly: 18...20), scrub: 30) }
            card { ActivityCharts(Self.chart(.mirror, [Self.sent, Self.received], unit: "MB/s", ends: Self.hourEnds)) }
            card { ActivityCharts(Self.chart(.mirror, [Self.sent, Self.received], unit: "MB/s", ends: Self.hourEnds), scrub: 23) }
        }
    }

    var small: some View {
        VStack(spacing: Token.Metric.Panel.blocks) {
            card {
                Bar(.plain(37 / 100))
                Bar(.memory(60 / 100))
                Bar(.memory(90 / 100))
                Bar(.memory(97 / 100))
                Bar(.capacity(os: 9, vms: [16, 8, 12], total: 64))
                Bar(.capacity(os: 9, vms: [16, 8, 12, 12], total: 64))
                UptimeCells(Self.days)
            }
            TopProcesses(Self.processes)
        }
    }

    public var body: some View {
        SwiftUI.Group {
            switch family {
            case .lines: lines
            case .bars: bars
            case .small: small
            }
        }
        .padding(Token.Metric.Panel.body)
        .frame(width: Token.Metric.Panel.width, alignment: .leading)
        .background(tokens.color.panelSolid)
    }
}
