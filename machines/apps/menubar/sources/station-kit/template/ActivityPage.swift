import SwiftUI

// invariant: SPEC §10.2 Activity: the machine's measures as graphs over
// the window picked, the picker of 6 min, 1 h and 24 h on top with the page's
// window picked, then the top processes. Picking hands the window's word to
// the app; nothing runs from here.
public struct ActivityPage: View {
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationPalette) private var palette

    // value: SPEC §6.4's picker: the window's word on the page, and the
    // word the call's window takes.
    public static let windows: [(words: String, flag: String)] = [("6 min", "6m"), ("1 h", "1h"), ("24 h", "24h")]

    public init(_ state: PanelState, handlers: PopoverHandlers = PopoverHandlers()) {
        self.state = state
        self.handlers = handlers
    }

    var page: Page { state.page ?? Page() }

    public static func picked(_ page: Page) -> String? { page.section("graphs")?.caption?.first?.text }

    // value: the picker as it is drawn: each window's words, whether it is
    // the page's, and its tap, which asks the app for that window.
    public static func picker(_ page: Page, handlers: PopoverHandlers) -> [(words: String, picked: Bool, tap: () -> Void)] {
        windows.map { w in (w.words, picked(page) == w.words, { handlers.window(w.flag) }) }
    }

    public var body: some View {
        let banner = page.section("machine")?.rows.first
        VStack(alignment: .leading, spacing: Token.Tile.section) {
            PageHeader(back: state.back ?? Page.ThisMacWord, title: "Activity", subtitle: Format.show(banner?.cells?.first ?? Value()), action: handlers.back)
            Segmented(options: Self.picker(page, handlers: handlers).map { ($0.words, $0.picked, $0.tap) })
            if let graphs = page.section("graphs") {
                VStack(alignment: .leading, spacing: Token.Gap.section) {
                    ForEach(Array(graphs.rows.enumerated()), id: \.offset) { i, r in measure(r, lead: i == 0) }
                }
                .stationTile()
                if !graphs.rows.contains(where: { if case .graph = Self.chart($0) { true } else { false } }) {
                    FootNote("The graphs fill in as \(banner?.label ?? Page.ThisMacWord) is sampled, every 2 s while this page is open.")
                }
            }
            if let top = page.section("processes") {
                let memory = Self.memory(top)
                VStack(alignment: .leading, spacing: Token.Space.gap) {
                    HStack(spacing: Token.Space.gap) {
                        SectionHeader(top.title ?? "")
                        Spacer(minLength: 0)
                        ForEach(["CPU", "Memory"], id: \.self) { key in
                            Text(key.uppercased()).font(.system(size: Token.Size.caption, weight: .semibold)).tracking(Token.Size.caption * Token.Size.tracking)
                                .foregroundStyle(Token.Ink.secondary).frame(width: Token.Measure.column, alignment: .trailing)
                        }
                    }
                    .padding(.trailing, Token.Tile.rowPadding.trailing)
                    VStack(alignment: .leading, spacing: 0) {
                        ForEach(Array(top.rows.prefix(5).enumerated()), id: \.offset) { i, r in
                            if i > 0 { RowSeparator() }
                            process(top, r, memory: memory)
                        }
                    }
                    .stationRows()
                }
            }
        }
    }

    // value: how a measure is drawn: its series as a graph, a share of a
    // whole (the disk's free space) as a meter, or, before it is sampled,
    // its graph's room waiting.
    public enum Chart: Equatable {
        case graph([Int])
        case meter(Value)
        case waiting
    }

    public static func chart(_ r: Row) -> Chart {
        let drawn = r.cells?.dropFirst().first
        if let points = drawn?.points, points.count > 1 { return .graph(points) }
        if let drawn, drawn.kind == "ratio", (drawn.of ?? 0) > 0 { return .meter(drawn) }
        return .waiting
    }

    static let shares: Set<String> = ["cpu", "memory", "swap"]

    static func line(_ points: [Int], unit: String?) -> PageChart {
        var series = Series()
        series.points = points.map(Double.init)
        series.unit = unit
        var chart = PageChart()
        chart.kind = ActivityCharts.Kind.line.rawValue
        chart.series = [series]
        return chart
    }

    // value: a series cut into `n` buckets, each its mean, oldest first.
    static func buckets(_ points: [Int], _ n: Int) -> [Int] {
        guard points.count > n else { return points }
        return (0..<n).map { i in
            let lo = i * points.count / n, hi = max((i + 1) * points.count / n, lo + 1)
            let slice = points[lo..<hi]
            return slice.reduce(0, +) / slice.count
        }
    }

    // invariant: a measure as the vision draws one: its name and its
    // words now, then its line over the window, the first one with the
    // window's start and Now under it.
    func measure(_ r: Row, lead: Bool) -> some View {
        let words = r.cells?.first.map { Format.show($0) } ?? ""
        let height = lead ? Token.Line.leadChart : Token.Line.chart
        return VStack(alignment: .leading, spacing: Token.Gap.small) {
            HStack(alignment: .firstTextBaseline) {
                Text(r.label ?? "").font(.subheadline.weight(.semibold)).foregroundStyle(Token.Ink.primary)
                Spacer(minLength: Token.Space.gap)
                Text(words).font(.subheadline.weight(.medium)).foregroundStyle(Token.Ink.primary).lineLimit(1).monospacedDigit()
                    .contentTransition(.numericText())
            }
            switch Self.chart(r) {
            case .graph(let points):
                ActivityCharts(Self.line(Self.buckets(points, 90), unit: Self.shares.contains(r.id ?? "") ? "%" : nil), scrub: state.scrub, onScrub: handlers.scrub)
            case .meter(let ratio):
                GeometryReader { geo in
                    let share = CGFloat(ratio.n ?? 0) / CGFloat(max(ratio.of ?? 1, 1))
                    ZStack(alignment: .leading) {
                        Capsule().fill(Token.Ink.track)
                        Capsule().fill(palette.accent).frame(width: geo.size.width * share)
                    }
                }
                .frame(height: Token.Space.meterHeight)
            case .waiting:
                ZStack(alignment: .bottom) {
                    Color.clear
                    Rectangle().fill(Token.Ink.track).frame(height: Token.Stroke.rule)
                }
                .frame(height: height)
                .accessibilityLabel("Not sampled yet")
            }
            if lead {
                HStack {
                    Text(Self.picked(page).map { "\($0) ago" } ?? "")
                    Spacer()
                    Text("Now")
                }
                .font(.caption).foregroundStyle(Token.Ink.secondary)
            }
        }
    }

    // value: why a process has no figure in a column: the engine ranks
    // the busiest by CPU and by memory apart, and names a figure only
    // where its list holds the process.
    static func missing(_ key: String) -> String {
        "Not among the busiest by \(key == "cpu" ? "CPU" : key)"
    }

    // value: the memory column's figures by process, each in the unit of
    // the largest.
    public static func memory(_ s: Section) -> [String: String] {
        let sized = s.rows.compactMap { r in s.cell(r, "memory").map { (r.id ?? "", $0.n ?? 0) } }
        return Dictionary(zip(sized.map(\.0), Format.column(sized.map(\.1))), uniquingKeysWith: { a, _ in a })
    }

    func process(_ s: Section, _ r: Row, memory: [String: String]) -> some View {
        HStack(spacing: Token.Space.gap) {
            Text(r.label ?? "").font(.system(size: Token.Size.text)).foregroundStyle(Token.Ink.primary).lineLimit(1)
            if let what = s.cell(r, "what") { Text(Format.show(what)).font(.subheadline).foregroundStyle(Token.Ink.secondary).lineLimit(1) }
            Spacer(minLength: 0)
            ForEach(["cpu", "memory"], id: \.self) { key in
                if let v = key == "memory" ? memory[r.id ?? ""] : s.cell(r, key).map({ Format.show($0) }) {
                    Text(v).font(.subheadline).foregroundStyle(Token.Ink.secondary).monospacedDigit().lineLimit(1).frame(width: Token.Measure.column, alignment: .trailing)
                } else {
                    Text(Token.Glyph.missing).font(.system(size: Token.Space.small)).foregroundStyle(Token.Ink.tertiary).frame(width: Token.Measure.column, alignment: .trailing)
                        .help(Self.missing(key))
                }
            }
        }
        .padding(Token.Tile.rowPadding)
        .frame(minHeight: Token.Line.activity)
    }
}
