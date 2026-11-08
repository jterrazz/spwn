import SwiftUI

// value: the window a machine's Activity shows: the one last picked for
// that machine, else the one its page opens on.
public enum ActivityWindow {
    public static let segment = "activity"

    public static func flag(_ action: Action) -> String { String((action.id ?? "").dropFirst(MachineActivity.windowPrefix.count)) }

    public static func opens(_ page: Page) -> String? {
        let picker = (page.sections ?? []).first { MachineActivity.windowed($0) }?.picker ?? []
        return picker.first { $0.picked == true }.map(flag)
    }

    public static func remembered(_ state: PanelState, _ page: Page) -> String {
        state.windows[page.context?.machine ?? ""] ?? opens(page) ?? ""
    }
}

// invariant: SPEC A4.1: a machine's Activity: the window picked, its CPU
// and its network over it, the memory now, the busiest processes, then on
// a VM its upkeep and where its figures come from. Every window is on the
// page: picking one redraws it and asks the engine nothing.
struct MachineActivity: View {
    let page: Page
    let sections: [Section]
    let window: String
    let pick: (String) -> Void
    let scrub: Int?
    let onScrub: (Int?) -> Void
    let run: (Action, Row?) -> Void
    let open: (String) -> Void
    @Environment(\.stationTokens) private var tokens

    static let windowPrefix = "window-"

    // value: the section of the window picked, else the first one the page
    // carries.
    static func shown(_ sections: [Section], window: String) -> Section? {
        sections.first { $0.id == windowPrefix + window } ?? sections.first { $0.id?.hasPrefix(windowPrefix) == true }
    }

    static func windowed(_ s: Section) -> Bool { s.id?.hasPrefix(windowPrefix) == true }

    var body: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
            if let w = Self.shown(sections, window: window) { windowed(w) }
            ForEach(Array(sections.filter { !Self.windowed($0) }.enumerated()), id: \.offset) { _, s in
                block(s)
            }
        }
    }

    func windowed(_ s: Section) -> some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
            HStack(alignment: .center) {
                Text(s.title ?? "").font(Token.Role.sectionLabel.font).foregroundStyle(tokens.tone(Token.Role.sectionLabel.tone))
                Spacer(minLength: 0)
                Segmented(options: (s.picker ?? []).map { a in
                    let flag = ActivityWindow.flag(a)
                    return (label: a.label ?? "", on: flag == window, tap: { pick(flag) })
                })
                .frame(width: Token.Metric.Activity.picker)
            }
            .padding(.horizontal, Token.Metric.CardLabel.inset)
            VStack(spacing: Token.Metric.Panel.blocks) {
                ForEach(Array(Self.cards(s.items ?? []).enumerated()), id: \.offset) { i, card in
                    chartCard(card.row, card.chart, scrubbed: i == 0)
                }
            }
            .id(window)
            .transition(.opacity)
        }
    }

    // value: a rate received and sent, each after its arrow.
    static func directed(_ v: Value) -> String {
        (v.items ?? []).map { part in
            let arrow = part.label == "received" ? Token.Glyph.received : part.label == "sent" ? Token.Glyph.sent : ""
            return [arrow, Format.show(part)].filter { !$0.isEmpty }.joined(separator: Token.Glyph.space)
        }.joined(separator: Token.Glyph.sep)
    }

    // value: a window's cards: each row titles the chart under it.
    static func cards(_ items: [Item]) -> [(row: Row, chart: PageChart?)] {
        var out: [(row: Row, chart: PageChart?)] = []
        for item in items {
            if let row = item.row {
                out.append((row, nil))
            } else if let chart = item.chart, !out.isEmpty, out[out.count - 1].chart == nil {
                out[out.count - 1].chart = chart
            }
        }
        return out
    }

    func chartCard(_ row: Row, _ chart: PageChart?, scrubbed: Bool) -> some View {
        Card(flush: false) {
            VStack(alignment: .leading, spacing: Token.Metric.Activity.cardGap) {
                HStack(alignment: .firstTextBaseline, spacing: 0) {
                    Text(row.label ?? "").font(Token.Role.chartTitle.font).foregroundStyle(tokens.tone(Token.Role.chartTitle.tone))
                    Spacer(minLength: Token.Metric.Row.gap)
                    if let now = row.cells?.first, now.kind == "list" {
                        Text(Self.directed(now)).font(Token.Role.chartValue.font).foregroundStyle(tokens.tone(Token.Role.chartValue.tone))
                    } else if let now = row.cells?.first {
                        LiveText(now, role: .chartValue)
                    }
                    if let word = row.detail, !Format.show(word).isEmpty {
                        Text(Token.Glyph.space + Format.show(word)).font(Token.Role.chartValue.font).foregroundStyle(tokens.tone(Token.Role.chartValue.tone))
                    }
                }
                .lineLimit(1)
                if let sentence = chart?.anomaly?.sentence, !sentence.isEmpty {
                    Text(sentence).font(Token.Role.chartNote.font).foregroundStyle(tokens.color.orangeText).lineLimit(1)
                }
                if let chart {
                    ActivityCharts(chart, scrub: scrubbed ? scrub : nil, onScrub: scrubbed ? onScrub : { _ in })
                    if chart.kind == ActivityCharts.Kind.mirror.rawValue { legend(chart) }
                }
            }
        }
    }

    // value: a mirrored chart's key: received in accent, sent in indigo,
    // in the order the card says them.
    func legend(_ chart: PageChart) -> some View {
        let keys = (chart.series ?? []).compactMap(\.key)
        return HStack(spacing: Token.Metric.Activity.legendGap) {
            ForEach(Array(keys.reversed().enumerated()), id: \.offset) { i, key in
                dot(key, ink: i == 0 ? tokens.color.accent : tokens.color.indigo)
            }
        }
    }

    func dot(_ words: String, ink: Color, hollow: Bool = false) -> some View {
        let m = Token.Metric.Activity.self
        return HStack(spacing: m.legendDotGap) {
            if hollow {
                Circle().strokeBorder(ink, lineWidth: Token.Metric.Chart.rule).frame(width: m.legendDot, height: m.legendDot)
            } else {
                Circle().fill(ink).frame(width: m.legendDot, height: m.legendDot)
            }
            Text(words).font(Token.Role.legend.font).foregroundStyle(tokens.tone(Token.Role.legend.tone))
        }
    }

    @ViewBuilder
    func block(_ s: Section) -> some View {
        switch s.id {
        case "memory":
            if let gauge = s.items?.compactMap(\.gauge).first { memory(s, gauge) }
        case "processes":
            if !s.rows.isEmpty {
                VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
                    CardLabel(s.title ?? "", value: s.caption ?? [])
                    TopProcesses(s, open: open)
                }
            }
        case "guest-upkeep":
            VStack(alignment: .leading, spacing: Token.Metric.Panel.label) {
                CardLabel(s.title ?? "", value: s.caption ?? [])
                Card { RowStack(s.rows, columns: s.columns ?? [], actions: page.actions ?? []) { run($0, nil) } }
            }
        default:
            ForEach(Array(s.notes.enumerated()), id: \.offset) { _, note in
                Text(note).font(Token.Role.footnote.font).foregroundStyle(tokens.tone(Token.Role.footnote.tone))
                    .lineSpacing(Token.Metric.Footnote.lineSpacing)
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(Token.Metric.Footnote.padding)
            }
        }
    }

    // value: the memory bar's parts by their label: macOS and each VM on a
    // Mac, used and free inside a VM, over the total.
    static func part(_ bar: Value, _ label: String) -> Value? { bar.items?.first { $0.label == label } }

    static func bytes(_ v: Value?) -> Double { Double(v?.n ?? 0) }

    static func kind(_ bar: Value) -> Bar.Kind {
        let total = bytes(part(bar, "of"))
        if let os = part(bar, "macOS") {
            let vms = (part(bar, "VMs")?.items ?? []).map(bytes)
            return .capacity(os: bytes(os), vms: vms, total: total)
        }
        return .memory(total > 0 ? bytes(part(bar, "used")) / total : 0)
    }

    func memory(_ s: Section, _ gauge: PageGauge) -> some View {
        Card(flush: false) {
            VStack(alignment: .leading, spacing: Token.Metric.Activity.cardGap) {
                HStack(alignment: .firstTextBaseline, spacing: 0) {
                    Text(gauge.title ?? "").font(Token.Role.chartTitle.font).foregroundStyle(tokens.tone(Token.Role.chartTitle.tone))
                    if let caption = s.caption, !caption.isEmpty {
                        Text(Token.Glyph.sep + FactsLine.words(caption)).font(Token.Role.chartValue.font).foregroundStyle(tokens.color.t3)
                    }
                    Spacer(minLength: Token.Metric.Row.gap)
                    if let value = gauge.value { LiveText(value, role: .chartValue) }
                }
                .lineLimit(1)
                if let bar = gauge.bar {
                    Bar(Self.kind(bar))
                    legend(bar)
                }
            }
        }
    }

    func legend(_ bar: Value) -> some View {
        HStack(spacing: Token.Metric.Activity.legendGap) {
            ForEach(Array((bar.items ?? []).enumerated()), id: \.offset) { _, part in
                let label = part.label ?? ""
                let words = [label, part.text ?? Format.show(part)].joined(separator: Token.Glyph.space)
                if label == "of" {
                    Text(words).font(Token.Role.legend.font).foregroundStyle(tokens.tone(Token.Role.legend.tone))
                } else {
                    dot(words, ink: Self.ink(label, tokens), hollow: label == "free")
                }
            }
        }
        .lineLimit(1)
    }

    static func ink(_ label: String, _ tokens: Token.Resolved) -> Color {
        switch label {
        case "macOS": tokens.color.t3
        case "free": tokens.color.t3
        default: tokens.color.accent
        }
    }
}
