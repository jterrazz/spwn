import SwiftUI

// invariant: SPEC A8 row by row: every movement drawn by its modifier at one
// moment of its run, or named with why it draws its end state.
public struct MotionSheet: View {
    public struct Cell: Identifiable {
        public let id: String
        public let title: String
        let draw: (Double, Bool) -> AnyView
    }

    let moment: Double
    let reduce: Bool
    @Environment(\.stationTokens) private var tokens

    public init(moment: Double = Token.Motion.Sheet.moment, reduce: Bool = false) {
        self.moment = moment
        self.reduce = reduce
    }

    public static let exempt: [String: String] = [
        "M13": "a number rolls by contentTransition, which draws its end state",
        "M14": "a symbol flips by contentTransition, which draws its end state",
        "M16": "Spinner turns on a TimelineView, still at a fixed angle",
        "M29": "StatusItem draws the item's frames at a progress",
    ]

    static func card(_ fill: Color, _ ink: Color) -> some View {
        let s = Token.Motion.Sheet.self
        return RoundedRectangle(cornerRadius: s.radius, style: .continuous)
            .fill(fill)
            .overlay(alignment: .topLeading) {
                VStack(alignment: .leading, spacing: s.gap) {
                    ForEach(s.lineWidths.indices, id: \.self) { i in
                        Capsule().fill(ink).frame(width: s.lineWidths[i], height: s.line)
                    }
                }
                .padding(s.inset)
            }
    }

    struct Sample: View {
        var lit = false
        @Environment(\.stationTokens) private var tokens

        var body: some View {
            MotionSheet.card(lit ? tokens.color.hero : tokens.color.card2, lit ? tokens.color.accent : tokens.color.t3)
        }
    }

    struct Line: View {
        @Environment(\.stationTokens) private var tokens

        var body: some View {
            Capsule().fill(tokens.color.t3).frame(height: Token.Motion.Sheet.line).padding(.horizontal, Token.Motion.Sheet.inset)
        }
    }

    struct Rows<Middle: View>: View {
        let middle: Middle

        init(@ViewBuilder middle: () -> Middle) { self.middle = middle() }

        var body: some View {
            VStack(spacing: Token.Motion.Sheet.gap) {
                Line()
                middle
                Line()
            }
            .frame(maxHeight: .infinity, alignment: .top)
            .padding(.top, Token.Motion.Sheet.inset)
        }
    }

    static func symbol(_ name: String, tint: Ink) -> some View {
        Image(systemName: name).font(Token.Role.heroTitle.font).foregroundStyle(tint)
    }

    static var thumbs: (CGRect, CGRect) {
        let s = Token.Motion.Sheet.self
        let w = (s.stage.width - s.inset * 2) / CGFloat(s.segments)
        let h = s.line * 2
        return (CGRect(x: s.inset, y: s.inset, width: w, height: h), CGRect(x: s.inset + w, y: s.inset, width: w, height: h))
    }

    public static let cells: [Cell] = [
        Cell(id: "M1", title: "panel open") { t, r in
            AnyView(Sample(lit: true).modifier(PanelAppear(progress: PanelAppear.frozen(t, closing: false, reduce: r))))
        },
        Cell(id: "M2", title: "push") { t, r in
            AnyView(ZStack {
                Sample().modifier(PushTransition(progress: PushTransition.frozen(t, role: .under, way: .push, reduce: r), role: .under))
                Sample(lit: true).modifier(PushTransition(progress: PushTransition.frozen(t, role: .top, way: .push, reduce: r), role: .top))
            }.clipped())
        },
        Cell(id: "M3", title: "pop") { t, r in
            AnyView(ZStack {
                Sample().modifier(PushTransition(progress: PushTransition.frozen(t, role: .under, way: .pop, reduce: r), role: .under, way: .pop))
                Sample(lit: true).modifier(PushTransition(progress: PushTransition.frozen(t, role: .top, way: .pop, reduce: r), role: .top, way: .pop))
            }.clipped())
        },
        Cell(id: "M4", title: "jump") { t, _ in
            AnyView(ZStack {
                Sample().modifier(JumpTransition(progress: JumpTransition.frozen(t, role: .leaving), role: .leaving))
                Sample(lit: true).modifier(JumpTransition(progress: JumpTransition.frozen(t, role: .arriving), role: .arriving))
            })
        },
        Cell(id: "M5", title: "segment") { t, r in
            let (from, to) = thumbs
            return AnyView(ZStack(alignment: .topLeading) {
                Sample()
                Capsule().fill(Ink.qq)
                    .modifier(SegmentThumb(progress: SegmentThumb.frozen(t, reduce: r), from: from, to: to))
                Rows { Line() }.padding(.top, Token.Motion.Sheet.line * 2)
                    .modifier(SegmentSlide(progress: SegmentSlide.frozen(t, reduce: r)))
            })
        },
        Cell(id: "M6", title: "height") { t, r in
            AnyView(Sample(lit: true)
                .modifier(HeightMorph(progress: HeightMorph.frozen(t, reduce: r), from: Token.Motion.Sheet.stage.height * Token.Motion.Sheet.morphFrom, to: Token.Motion.Sheet.stage.height))
                .frame(maxHeight: .infinity, alignment: .top))
        },
        Cell(id: "M7", title: "fold") { t, r in
            AnyView(VStack(spacing: Token.Motion.Sheet.gap) {
                symbol(Token.Symbol.down, tint: .t2).modifier(FoldChevron(progress: Fold.frozen(t, reduce: r)))
                VStack(spacing: Token.Motion.Sheet.gap) { Line(); Line() }.modifier(Fold(progress: Fold.frozen(t, reduce: r)))
            }.frame(maxHeight: .infinity, alignment: .top))
        },
        Cell(id: "M8", title: "hero morph") { t, r in
            AnyView(VStack(spacing: Token.Motion.Sheet.gap) {
                symbol(Token.Symbol.check, tint: .red).modifier(HeroMorph(progress: HeroMorph.frozen(t, part: .glyph, reduce: r), part: .glyph))
                Line().modifier(HeroMorph(progress: HeroMorph.frozen(t, part: .top, reduce: r), part: .top))
                Line().modifier(HeroMorph(progress: HeroMorph.frozen(t, part: .rows, reduce: r), part: .rows))
            }.frame(maxHeight: .infinity, alignment: .top))
        },
        Cell(id: "M9", title: "row leaves") { t, r in
            AnyView(Rows { Line().modifier(RowLeave(progress: RowLeave.frozen(t, reduce: r))) })
        },
        Cell(id: "M10", title: "row lands") { t, r in
            AnyView(Rows { Line().modifier(RowLand(progress: RowLand.frozen(t, reduce: r))) })
        },
        Cell(id: "M11", title: "fresh row") { t, _ in
            AnyView(Rows { Line().padding(.vertical, Token.Motion.Sheet.gap).modifier(FreshTint(progress: FreshTint.frozen(t))) })
        },
        Cell(id: "M12", title: "success") { t, r in
            AnyView(symbol(Token.Symbol.check, tint: .green).modifier(Born(progress: Born.frozen(t, reduce: r))))
        },
        Cell(id: "M15", title: "ping") { t, r in
            AnyView(PingDot(progress: Ping.frozen(t, reduce: r)))
        },
        Cell(id: "M17", title: "skeleton") { t, _ in
            AnyView(Sample().modifier(Skeleton(progress: Skeleton.frozen(t))))
        },
        Cell(id: "M18", title: "stale reveal") { t, r in
            AnyView(Sample(lit: true).modifier(Reveal(progress: Reveal.frozen(t, reduce: r))))
        },
        Cell(id: "M19", title: "glyph draws") { t, r in
            AnyView(symbol(Token.Symbol.awake, tint: .t2).modifier(GlyphDraw(progress: GlyphDraw.frozen(t, reduce: r))))
        },
        Cell(id: "M20", title: "sheet") { t, r in
            AnyView(ZStack(alignment: .top) {
                Sample()
                RoundedRectangle(cornerRadius: Token.Motion.Sheet.radius, style: .continuous).fill(Ink.scrim).modifier(ScrimFade(progress: ScrimFade.frozen(t, reduce: r)))
                Sample(lit: true).padding(.horizontal, Token.Motion.Sheet.inset).modifier(SheetReveal(progress: SheetReveal.frozen(t, reduce: r)))
            })
        },
        Cell(id: "M21", title: "alert") { t, r in
            AnyView(Sample(lit: true).padding(Token.Motion.Sheet.inset * 2).modifier(PopAppear(progress: PopAppear.frozen(t, kind: .alert, reduce: r), kind: .alert)))
        },
        Cell(id: "M22", title: "find") { t, r in
            AnyView(ZStack(alignment: .top) {
                Sample().modifier(FindAppear(progress: FindAppear.frozen(t, field: false, reduce: r)))
                Line().padding(.top, Token.Motion.Sheet.inset).modifier(FindAppear(progress: FindAppear.frozen(t, field: true, reduce: r), field: true))
            })
        },
        Cell(id: "M23", title: "menu") { t, r in
            AnyView(Sample(lit: true).padding(Token.Motion.Sheet.inset).modifier(PopAppear(progress: PopAppear.frozen(t, kind: .menu, reduce: r), kind: .menu, anchor: .topLeading)))
        },
        Cell(id: "M24", title: "popover") { t, r in
            AnyView(Sample(lit: true).padding(Token.Motion.Sheet.inset).modifier(PopAppear(progress: PopAppear.frozen(t, kind: .popover, reduce: r), kind: .popover, anchor: .top)))
        },
        Cell(id: "M25", title: "hover") { t, r in
            AnyView(HStack {
                Line()
                symbol(Token.Symbol.copy, tint: .t2).modifier(HoverReveal(progress: HoverReveal.frozen(t, reduce: r)))
            }.padding(.horizontal, Token.Motion.Sheet.inset))
        },
        Cell(id: "M26", title: "tile plus one") { t, r in
            AnyView(Sample().overlay { Text(verbatim: "+1").font(Token.Role.tileNumber.font).modifier(PlusOne(progress: PlusOne.frozen(t, reduce: r))) })
        },
        Cell(id: "M27", title: "chart regrows") { t, r in
            AnyView(HStack(alignment: .bottom, spacing: Token.Motion.Sheet.gap) {
                ForEach(Array(Token.Motion.Sheet.bars.enumerated()), id: \.offset) { _, share in
                    Capsule().fill(Ink.accent).frame(height: Token.Motion.Sheet.stage.height * CGFloat(share))
                }
            }.modifier(ChartRegrow(progress: ChartRegrow.frozen(t, reduce: r))))
        },
        Cell(id: "M28", title: "footer rises") { t, r in
            AnyView(Line().frame(maxHeight: .infinity, alignment: .bottom).padding(.bottom, Token.Motion.Sheet.inset).modifier(FooterRise(progress: FooterRise.frozen(t, reduce: r))))
        },
        Cell(id: "M30", title: "HUD") { t, _ in
            AnyView(HUDCapsule(Value(kind: "text", text: "Done"), symbol: Token.Symbol.check)
                .modifier(HUDReveal(progress: Token.Motion.hudOpen * t)))
        },
        Cell(id: "M31", title: "progress") { t, r in
            AnyView(Capsule().fill(Ink.accent).frame(height: Token.Motion.Sheet.line)
                .modifier(ProgressEase(progress: ProgressEase.frozen(t, reduce: r), from: Token.Motion.Sheet.fillFrom, to: Token.Motion.Sheet.fillTo))
                .padding(.horizontal, Token.Motion.Sheet.inset))
        },
    ]

    enum Ink: ShapeStyle {
        case t2, t3, accent, red, green, qq, scrim

        func resolve(in environment: EnvironmentValues) -> Color.Resolved {
            let c = environment.stationTokens.color
            let color: Color = switch self {
            case .t2: c.t2
            case .t3: c.t3
            case .accent: c.accent
            case .red: c.red
            case .green: c.green
            case .qq: c.qq
            case .scrim: c.scrim
            }
            return color.resolve(in: environment)
        }
    }

    struct PingDot: View {
        let progress: Double
        @Environment(\.stationTokens) private var tokens

        var body: some View {
            StatusDot("fill", signal: "failing").modifier(Ping(progress: progress, ink: tokens.color.red))
        }
    }

    public var body: some View {
        let s = Token.Motion.Sheet.self
        let columns = Array(repeating: GridItem(.fixed(s.stage.width), spacing: Token.Metric.Tiles.gap, alignment: .top), count: s.columns)
        LazyVGrid(columns: columns, alignment: .leading, spacing: Token.Metric.Tiles.gap) {
            ForEach(Self.cells) { cell in
                VStack(alignment: .leading, spacing: s.gap) {
                    cell.draw(moment, reduce)
                        .frame(width: s.stage.width, height: s.stage.height)
                        .background(RoundedRectangle(cornerRadius: s.radius, style: .continuous).strokeBorder(tokens.color.hair, lineWidth: Token.Metric.Card.contrastStroke))
                    Text(verbatim: "\(cell.id) \(cell.title)").font(Token.Role.address.font).foregroundStyle(tokens.color.t2)
                }
            }
        }
        .padding(Token.Metric.Panel.body)
        .frame(width: Token.Metric.Panel.width, alignment: .topLeading)
        .background(tokens.panel)
        .environment(\.stationReduceMotion, reduce)
    }
}
