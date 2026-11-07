import Charts
import SwiftUI

public struct ActivityCharts: View {
    let chart: PageChart
    let scrub: Int?
    let onScrub: (Int?) -> Void
    @Environment(\.stationTokens) private var tokens

    public init(_ chart: PageChart, scrub: Int? = nil, onScrub: @escaping (Int?) -> Void = { _ in }) {
        self.chart = chart
        self.scrub = scrub
        self.onScrub = onScrub
    }

    public enum Kind: String, Sendable { case bars, line, mirror }

    var kind: Kind { Kind(rawValue: chart.kind ?? "") ?? .line }
    var series: [[Double]] { (chart.series ?? []).map { $0.points ?? [] } }
    var unit: String? { chart.series?.first?.unit }
    var count: Int { series.map(\.count).max() ?? 0 }
    var top: Double { ChartScale.top(series, unit: unit) }
    var lead: [Double] { series.first ?? [] }
    var picked: Int? { scrub.flatMap { (0..<count).contains($0) ? $0 : nil } }
    var peak: Int? { chart.peak?.index.flatMap { (0..<count).contains($0) ? $0 : nil } }
    var anomaly: ClosedRange<Int>? {
        guard let a = chart.anomaly, let from = a.from, let to = a.to, from <= to, from >= 0, to < count else { return nil }
        return from...to
    }

    var xDomain: ClosedRange<Double> {
        kind == .bars ? -Token.Metric.Chart.barSlot...(Double(max(count, 1)) - Token.Metric.Chart.barSlot) : 0...Double(max(count - 1, 1))
    }

    var yDomain: ClosedRange<Double> { kind == .mirror ? -top...top : 0...top }

    var height: CGFloat { kind == .mirror ? Token.Metric.Chart.network : Token.Metric.Chart.cpu }

    public var body: some View {
        let c = Token.Metric.Chart.self
        VStack(alignment: .leading, spacing: c.endsGap) {
            plot
                .frame(height: height)
                .padding(.top, c.chips)
            if let ends = chart.ends, ends.count == 2 {
                HStack {
                    Text(ends[0])
                    Spacer(minLength: 0)
                    Text(ends[1])
                }
                .font(.system(size: c.ends).monospacedDigit())
                .foregroundStyle(tokens.color.t3)
                .lineLimit(1)
            }
        }
    }

    @ViewBuilder var plot: some View {
        switch kind {
        case .bars: framed(Chart { grid; bars })
        case .line: framed(Chart { grid; line })
        case .mirror: framed(Chart { grid; mirror })
        }
    }

    func framed(_ plotted: some View) -> some View {
        plotted
            .chartXAxis(.hidden)
            .chartYAxis(.hidden)
            .chartLegend(.hidden)
            .chartXScale(domain: xDomain)
            .chartYScale(domain: yDomain)
            .chartXSelection(value: Binding(get: { picked.map(Double.init) }, set: { pick($0) }))
            .chartOverlay(alignment: .topLeading) { proxy in overlay(proxy) }
            .accessibilityElement(children: .ignore)
            .accessibilityLabel(chart.peak.map { "\(Format.show($0.value ?? Value())) \($0.at ?? "")" } ?? "")
    }

    func pick(_ x: Double?) {
        let index = x.map { Int($0.rounded()) }.flatMap { (0..<count).contains($0) ? $0 : nil }
        if index != picked { onScrub(index) }
    }

    @ChartContentBuilder var grid: some ChartContent {
        let c = Token.Metric.Chart.self
        RuleMark(y: .value("grid", top / 2))
            .foregroundStyle(tokens.color.hair)
            .lineStyle(StrokeStyle(lineWidth: c.rule, dash: c.gridDash))
        if kind == .mirror {
            RuleMark(y: .value("grid", -top / 2))
                .foregroundStyle(tokens.color.hair)
                .lineStyle(StrokeStyle(lineWidth: c.rule, dash: c.gridDash))
        }
        RuleMark(y: .value("axis", 0))
            .foregroundStyle(tokens.color.hair)
            .lineStyle(StrokeStyle(lineWidth: c.rule))
    }

    func barInk(_ i: Int) -> Color {
        if let anomaly, anomaly.contains(i) { return tokens.color.red }
        if i == (picked ?? count - 1) { return tokens.color.accent }
        return tokens.color.t3
    }

    func bar(_ i: Int, to value: Double, ink: Color, shown: Double) -> some ChartContent {
        let c = Token.Metric.Chart.self
        let x = Double(i), half = c.barShare / 2
        let start: PlottableValue<Double> = .value("t", x - half)
        let end: PlottableValue<Double> = .value("t", x + half)
        let floor: PlottableValue<Double> = .value("floor", 0)
        let top: PlottableValue<Double> = .value("v", value)
        return RectangleMark(xStart: start, xEnd: end, yStart: floor, yEnd: top)
            .clipShape(RoundedRectangle(cornerRadius: c.barCorner, style: .continuous))
            .foregroundStyle(ink)
            .opacity(shown)
    }

    @ChartContentBuilder var bars: some ChartContent {
        let c = Token.Metric.Chart.self
        let least = top * c.barLeast / Double(height)
        ForEach(Array(lead.enumerated()), id: \.offset) { i, v in
            bar(i, to: top, ink: tokens.color.track, shown: 1)
            bar(i, to: max(v, least), ink: barInk(i), shown: picked == nil || picked == i ? 1 : c.dimShare)
        }
    }

    @ChartContentBuilder var line: some ChartContent {
        let c = Token.Metric.Chart.self
        let ink = tokens.color.accent
        ForEach(Array(lead.enumerated()), id: \.offset) { i, v in
            AreaMark(x: .value("t", Double(i)), yStart: .value("floor", 0), yEnd: .value("v", v), series: .value("s", "area"))
                .interpolationMethod(.monotone)
                .foregroundStyle(ink.opacity(c.areaShare))
            LineMark(x: .value("t", Double(i)), y: .value("v", v), series: .value("s", "line"))
                .interpolationMethod(.monotone)
                .lineStyle(StrokeStyle(lineWidth: c.line, lineCap: .round, lineJoin: .round))
                .foregroundStyle(ink)
        }
        if let anomaly {
            ForEach(Array(max(anomaly.lowerBound - 1, 0)...min(anomaly.upperBound + 1, count - 1)), id: \.self) { i in
                LineMark(x: .value("t", Double(i)), y: .value("v", lead[i]), series: .value("s", "anomaly"))
                    .interpolationMethod(.monotone)
                    .lineStyle(StrokeStyle(lineWidth: c.anomalyLine, lineCap: .round, lineJoin: .round))
                    .foregroundStyle(tokens.color.orange)
            }
        }
        if let i = picked {
            scrubRule(i)
            dot(i, lead[i], fill: tokens.color.t1, size: c.endDot)
        } else {
            if let i = peak {
                RuleMark(x: .value("t", Double(i)), yStart: .value("v", lead[i]), yEnd: .value("top", top))
                    .foregroundStyle(tokens.color.t3)
                    .lineStyle(StrokeStyle(lineWidth: c.rule))
                PointMark(x: .value("t", Double(i)), y: .value("v", lead[i]))
                    .symbol {
                        Circle().fill(tokens.color.panelSolid)
                            .overlay(Circle().stroke(anomaly == nil ? tokens.color.accent : tokens.color.orange, lineWidth: c.peakStroke))
                            .frame(width: c.peakDot, height: c.peakDot)
                    }
            }
            if let last = lead.indices.last { dot(last, lead[last], fill: ink, size: c.endDot) }
        }
    }

    @ChartContentBuilder var mirror: some ChartContent {
        let c = Token.Metric.Chart.self
        let sent = lead
        let received = series.count > 1 ? series[1] : []
        ForEach(Array(sent.enumerated()), id: \.offset) { i, v in
            AreaMark(x: .value("t", Double(i)), yStart: .value("floor", 0), yEnd: .value("v", v), series: .value("s", "sent"))
                .interpolationMethod(.monotone)
                .foregroundStyle(tokens.color.indigo.opacity(c.sentShare))
            LineMark(x: .value("t", Double(i)), y: .value("v", v), series: .value("s", "sent line"))
                .interpolationMethod(.monotone)
                .lineStyle(StrokeStyle(lineWidth: c.line, lineCap: .round, lineJoin: .round))
                .foregroundStyle(tokens.color.indigo)
        }
        ForEach(Array(received.enumerated()), id: \.offset) { i, v in
            AreaMark(x: .value("t", Double(i)), yStart: .value("floor", 0), yEnd: .value("v", -v), series: .value("s", "received"))
                .interpolationMethod(.monotone)
                .foregroundStyle(tokens.color.accent.opacity(c.receivedShare))
            LineMark(x: .value("t", Double(i)), y: .value("v", -v), series: .value("s", "received line"))
                .interpolationMethod(.monotone)
                .lineStyle(StrokeStyle(lineWidth: c.line, lineCap: .round, lineJoin: .round))
                .foregroundStyle(tokens.color.accent)
        }
        if let i = picked {
            scrubRule(i)
            if i < sent.count { dot(i, sent[i], fill: tokens.color.indigo, size: c.peakDot) }
            if i < received.count { dot(i, -received[i], fill: tokens.color.accent, size: c.peakDot) }
        }
    }

    func scrubRule(_ i: Int) -> some ChartContent {
        let c = Token.Metric.Chart.self
        return RuleMark(x: .value("t", Double(i)))
            .foregroundStyle(tokens.color.t2)
            .lineStyle(StrokeStyle(lineWidth: c.rule, dash: c.ruleDash))
    }

    func dot(_ i: Int, _ v: Double, fill: Color, size: CGFloat) -> some ChartContent {
        PointMark(x: .value("t", Double(i)), y: .value("v", v))
            .symbol { Circle().fill(fill).frame(width: size, height: size) }
    }

    func overlay(_ proxy: ChartProxy) -> some View {
        GeometryReader { geo in
            let plot = proxy.plotFrame.map { geo[$0] } ?? CGRect(origin: .zero, size: geo.size)
            ZStack(alignment: .topLeading) {
                Rectangle().fill(Color.clear).contentShape(Rectangle())
                    .onContinuousHover { phase in
                        switch phase {
                        case .active(let at): pick(proxy.value(atX: at.x - plot.minX, as: Double.self))
                        case .ended: pick(nil)
                        }
                    }
                gridLabel(proxy, plot)
                chip(proxy, plot, width: geo.size.width)
            }
        }
    }

    @ViewBuilder func gridLabel(_ proxy: ChartProxy, _ plot: CGRect) -> some View {
        let c = Token.Metric.Chart.self
        if unit != nil, let y = proxy.position(forY: top / 2) {
            Text(ChartScale.say(top / 2, unit: unit))
                .font(.system(size: c.gridLabel).monospacedDigit())
                .foregroundStyle(tokens.color.t3)
                .fixedSize()
                .frame(height: c.gridRow, alignment: .bottom)
                .offset(x: plot.minX, y: plot.minY + y - c.gridRow)
                .allowsHitTesting(false)
        }
    }

    @ViewBuilder func chip(_ proxy: ChartProxy, _ plot: CGRect, width: CGFloat) -> some View {
        let c = Token.Metric.Chart.self
        if let i = picked, let x = proxy.position(forX: Double(i)) {
            callout(i)
                .fixedSize()
                .position(x: ChartScale.edge(plot.minX + x, width: width), y: -c.chips / 2)
                .allowsHitTesting(false)
        } else if let i = peak, let at = chart.peak, let x = proxy.position(forX: Double(i)) {
            let hot = anomaly != nil
            Text([Format.show(at.value ?? Value()), at.at ?? ""].filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep))
                .font(.system(size: c.ends, weight: .semibold).monospacedDigit())
                .foregroundStyle(hot ? tokens.color.orangeText : tokens.color.t2)
                .padding(c.chipPadding)
                .background(RoundedRectangle(cornerRadius: c.chipRadius, style: .continuous).fill(hot ? tokens.color.orange.opacity(c.chipTint) : tokens.color.q))
                .fixedSize()
                .position(x: ChartScale.edge(plot.minX + x, width: width), y: -c.chips / 2)
                .allowsHitTesting(false)
        }
    }

    func callout(_ i: Int) -> some View {
        let c = Token.Metric.Chart.self
        let words = series.enumerated().compactMap { k, points -> String? in
            guard i < points.count else { return nil }
            let said = ChartScale.say(points[i], unit: chart.series?[k].unit)
            guard kind == .mirror else { return said }
            return "\(k == 0 ? Token.Glyph.sent : Token.Glyph.received)\(said)"
        }
        return Text(words.joined(separator: Token.Glyph.sep))
            .font(.system(size: c.ends, weight: .semibold).monospacedDigit())
            .foregroundStyle(tokens.color.t1)
            .padding(c.calloutPadding)
            .background(
                RoundedRectangle(cornerRadius: c.calloutRadius, style: .continuous).fill(tokens.color.panelSolid)
                    .shadow(color: Token.Shadow.callout.color, radius: Token.Shadow.callout.radius, y: Token.Shadow.callout.y))
    }
}
