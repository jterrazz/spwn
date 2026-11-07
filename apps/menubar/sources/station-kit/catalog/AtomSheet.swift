import SwiftUI

public struct AtomSheet: View {
    public enum Family: String, CaseIterable, Sendable { case marks, text, controls }

    let family: Family
    @Environment(\.stationTokens) private var tokens

    public init(_ family: Family) { self.family = family }

    static let signals: [String?] = [nil, "active", "waits", "failing", "well", "faint", "done"]
    static let kinds = ["laptop", "desktop", "server", "workspace", "agent", "symbol:\(Token.Symbol.terminal)"]
    static let samples: [String: Double] = ["cpu:this-mac": 37, "cpu:mac-mini": 82, "cpu:linux-1": 34, "free:mac-mini": 19, "left:s2": 1350]

    static func glyph(_ kind: String, signal: String? = nil, dot: String? = nil, ring: String? = nil, spin: Bool? = nil, progress: Int? = nil) -> Glyph {
        var g = Glyph()
        g.kind = kind
        g.signal = signal
        g.dot = dot
        g.ring = ring
        g.spin = spin
        g.progress = progress
        return g
    }

    static func part(_ text: String, pri: Int? = nil, short: String? = nil, signal: String? = nil) -> Value {
        var v = Value(kind: "text", text: text)
        v.pri = pri
        v.short = short
        v.signal = signal
        return v
    }

    static let facts: Value = {
        var v = Value(kind: "list")
        v.items = [part("Mac mini"), part("M4 Pro", pri: 2), part("64 GB", pri: 3), part("up 21 days", pri: 1, short: "21 d")]
        return v
    }()

    static let holding: Value = {
        var v = Value(kind: "duration")
        v.n = 13920
        v.of = 14400
        return v
    }()

    static let ending: Value = {
        var v = Value(kind: "duration")
        v.n = 1380
        v.of = 14400
        return v
    }()

    static let cpu: Value = {
        var v = Value(kind: "percent")
        v.n = 120
        v.live = "cpu:linux-1"
        return v
    }()

    static let free: Value = {
        var v = Value(kind: "count")
        v.n = 0
        v.unit = "GB free"
        v.live = "free:mac-mini"
        return v
    }()

    func card<Content: View>(@ViewBuilder _ content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) { content() }
            .padding(Token.Metric.Card.padding)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(RoundedRectangle(cornerRadius: Token.Metric.Card.radius, style: .continuous).fill(tokens.color.card))
    }

    func line<Content: View>(@ViewBuilder _ content: () -> Content) -> some View {
        HStack(spacing: Token.Metric.Row.gap) { content() }.frame(minHeight: Token.Metric.Row.height)
    }

    var marks: some View {
        VStack(alignment: .leading, spacing: Token.Metric.Panel.blocks) {
            card {
                ForEach(Array(Self.kinds.enumerated()), id: \.offset) { _, kind in
                    line { ForEach(Array(Self.signals.enumerated()), id: \.offset) { _, s in GlyphView(Self.glyph(kind, signal: s)) } }
                }
            }
            card {
                line {
                    GlyphView(Self.glyph("agent", signal: "active", dot: "fill"))
                    GlyphView(Self.glyph("agent", signal: "waits", dot: "fill"))
                    GlyphView(Self.glyph("workspace", spin: true))
                    GlyphView(Self.glyph("workspace", progress: 400))
                    GlyphView(Self.glyph("server", signal: "failing", dot: "ping"))
                    GlyphView(Self.glyph("laptop", ring: "cpu:this-mac"))
                    GlyphView(Self.glyph("desktop", ring: "cpu:mac-mini"))
                    GlyphView(Self.glyph("laptop", signal: "asleep", ring: "cpu:air"))
                }
                line {
                    ForEach(Array(["well", "waits", "failing", "active", "faint"].enumerated()), id: \.offset) { _, s in StatusDot("fill", signal: s) }
                    StatusDot("ring")
                    StatusDot("dash")
                    StatusDot("square")
                    StatusDot("ping", signal: "failing")
                    ProgressRing(perMille: 0)
                    ProgressRing(perMille: 400)
                    ProgressRing(perMille: 1000)
                    Spinner()
                    Spinner(small: true)
                }
            }
            card {
                line { ForEach(Token.Object.accounts + ["gitlab"], id: \.self) { LogoChip(.account, $0) } }
                line { ForEach(Token.Object.ports + ["redis"], id: \.self) { LogoChip(.port, $0) } }
            }
        }
    }

    var text: some View {
        card {
            line {
                KeyCap("cmd+k")
                KeyCap("cmd+enter")
                KeyCap("opt+enter")
                KeyCap("cmd+0")
                KeyCap("esc")
                KeyCap("shift+cmd+p")
            }
            line {
                Pill("linux-1:5173", reach: .tailnet)
                Pill("api.jterrazz.com", reach: .open)
            }
            line {
                Pill("localhost:6006", reach: .unshared)
                Pill("linux-2:5432", reach: .off)
            }
            line {
                LiveText(Self.cpu)
                LiveText(Self.free)
            }
            line {
                TimeLeft(Self.holding)
                TimeLeft(Self.ending)
            }
            FactsLine(Self.facts)
            FactsLine(Self.facts).frame(width: Token.Metric.Alert.width / 2, alignment: .leading)
        }
    }

    var controls: some View {
        card {
            line {
                Switch(on: true)
                Switch(on: false)
                Switch(on: true).disabled(true)
            }
            Segmented(options: [("Overview", true, {}), ("Activity", false, {}), ("Ports", false, {})])
            Segmented(options: [("1 hr", false, {}), ("4 hr", true, {}), ("Until 18:00", false, {})])
            line {
                PanelButton("Restart", look: .tint)
                PanelButton("Copy", symbol: Token.Symbol.copy)
                PanelButton("Open", key: "cmd+o", look: .soft)
            }
            line {
                PanelButton("Restart", look: .redTint)
                PanelButton("Delete", look: .destructive)
                PanelButton("Create", key: "cmd+enter", look: .tint, big: true)
            }
            line {
                VerbButton("Review")
                VerbButton("Share")
                VerbButton("Waiting", busy: true)
            }
            line {
                RoundButton(Token.Symbol.terminal)
                RoundButton(Token.Symbol.plus, size: .small)
                RoundButton(Token.Symbol.search, size: .topBar)
                RoundButton(Token.Symbol.copy, size: .small, quiet: true)
                IconButton(Token.Symbol.gear)
            }
            PanelButton("Share a port", grow: true)
        }
    }

    public var body: some View {
        SwiftUI.Group {
            switch family {
            case .marks: marks
            case .text: text
            case .controls: controls
            }
        }
        .environment(\.stationSamples, Self.samples)
        .padding(Token.Metric.Panel.body)
        .frame(width: Token.Metric.Panel.width, alignment: .leading)
        .background(tokens.color.panelSolid)
    }
}
