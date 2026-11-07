import SwiftUI

public struct StatusDot: View {
    public enum Form: String, Sendable { case fill, ring, dash, square }

    let shape: Form
    let signal: String?
    let ping: Bool
    let size: CGFloat
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still
    @State private var pinged = false

    public init(_ dot: String?, signal: String? = nil, size: CGFloat = Token.Metric.Marks.dot) {
        shape = Form(rawValue: dot ?? "") ?? .fill
        self.signal = signal
        ping = dot == "ping"
        self.size = size
    }

    var ink: Color { shape == .fill ? tokens.signal(signal ?? "well") : tokens.color.t3 }

    public var body: some View {
        mark
            .frame(width: size, height: size)
            .overlay {
                if ping && !still {
                    Circle().fill(ink)
                        .scaleEffect(pinged ? Token.Metric.Marks.pingScale : 1)
                        .opacity(pinged ? 0 : Token.Metric.Marks.pingShare)
                        .onAppear { withAnimation(tokens.motion(Token.Motion.ping)) { pinged = true } }
                }
            }
    }

    @ViewBuilder var mark: some View {
        switch shape {
        case .fill: Circle().fill(ink)
        case .ring: Circle().strokeBorder(ink, lineWidth: Token.Metric.Marks.ringStroke)
        case .dash: Circle().strokeBorder(ink, style: StrokeStyle(lineWidth: Token.Metric.Marks.dashStroke, dash: Token.Metric.Marks.dash))
        case .square:
            RoundedRectangle(cornerRadius: Token.Metric.Marks.squareRadius, style: .continuous).strokeBorder(ink, lineWidth: Token.Metric.Marks.ringStroke)
        }
    }
}

struct Arc: InsettableShape {
    var share: Double
    var from: Angle = .degrees(-90)
    var inset: CGFloat = 0

    var animatableData: Double {
        get { share }
        set { share = newValue }
    }

    func path(in rect: CGRect) -> Path {
        var p = Path()
        let r = min(rect.width, rect.height) / 2 - inset
        p.addArc(center: CGPoint(x: rect.midX, y: rect.midY), radius: r, startAngle: from, endAngle: from + .degrees(360 * min(max(share, 0), 1)), clockwise: false)
        return p
    }

    func inset(by amount: CGFloat) -> Arc {
        var arc = self
        arc.inset += amount
        return arc
    }
}

public struct MacRing: View {
    let symbol: String
    let share: Double
    let asleep: Bool
    @Environment(\.stationTokens) private var tokens

    public init(lid: Bool, share: Double, asleep: Bool = false) {
        symbol = Token.Object.mac(lid: lid)
        self.share = share
        self.asleep = asleep
    }

    public var body: some View {
        let m = Token.Metric.Marks.self
        let inset = m.macRing / 2 - m.macRingRadius
        ZStack {
            Circle().inset(by: inset)
                .stroke(tokens.color.track, style: StrokeStyle(lineWidth: m.macRingStroke, dash: asleep ? m.asleepDash : []))
            if !asleep {
                Arc(share: share).inset(by: inset)
                    .stroke(share > Token.Hot.cpu ? tokens.color.orange : tokens.color.accent, style: StrokeStyle(lineWidth: m.macRingStroke, lineCap: .round))
                    .animation(tokens.motion(Token.Motion.progress), value: share)
            }
            Image(systemName: symbol).font(.system(size: m.macGlyph, weight: .medium)).foregroundStyle(tokens.color.t2)
        }
        .frame(width: m.macRing, height: m.macRing)
        .padding(.horizontal, m.ringMargin)
    }
}

public struct ProgressRing: View {
    let share: Double
    let size: CGFloat
    @Environment(\.stationTokens) private var tokens

    public init(perMille: Int, size: CGFloat = Token.Metric.Marks.progress) {
        share = max(Double(perMille) / Token.Metric.Marks.perMille, Token.Metric.Marks.progressLeast)
        self.size = size
    }

    public var body: some View {
        let stroke = Token.Metric.Marks.progressStroke
        ZStack {
            Circle().inset(by: stroke / 2).stroke(tokens.color.track, lineWidth: stroke)
            Arc(share: share).inset(by: stroke / 2)
                .stroke(tokens.color.accent, style: StrokeStyle(lineWidth: stroke, lineCap: .round))
                .animation(tokens.motion(Token.Motion.progress), value: share)
        }
        .frame(width: size, height: size)
    }
}

public struct Spinner: View {
    let small: Bool
    let tint: Color?
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still

    public init(small: Bool = false, tint: Color? = nil) {
        self.small = small
        self.tint = tint
    }

    public var body: some View {
        if still {
            ring(Token.Metric.Marks.spinnerStill)
        } else {
            TimelineView(.animation) { context in
                let period = Token.Metric.Marks.spinnerPeriod
                let turn = context.date.timeIntervalSinceReferenceDate.truncatingRemainder(dividingBy: period) / period
                ring(Token.Metric.Marks.spinnerStill + .degrees(360 * turn))
            }
        }
    }

    func ring(_ angle: Angle) -> some View {
        let m = Token.Metric.Marks.self
        let size = small ? m.spinnerSmall : m.spinner
        let stroke = small ? m.spinnerSmallStroke : m.spinnerStroke
        return ZStack {
            Circle().inset(by: stroke / 2).stroke(tokens.color.track, lineWidth: stroke)
            Arc(share: m.spinnerArc, from: angle).inset(by: stroke / 2).stroke(tint ?? tokens.color.accent, lineWidth: stroke)
        }
        .frame(width: size, height: size)
    }
}

public struct LogoChip: View {
    public enum Kind: Sendable { case account, port }

    let kind: Kind
    let brand: String
    @Environment(\.stationTokens) private var tokens

    public init(_ kind: Kind, _ brand: String) {
        self.kind = kind
        self.brand = brand
    }

    var known: String? { Token.Object.brand(brand, among: kind == .account ? Token.Object.accounts : Token.Object.ports) }

    public var body: some View {
        let m = Token.Metric.Marks.self
        let chip = tokens.chip(known)
        let shape = RoundedRectangle(cornerRadius: m.logoRadius, style: .continuous)
        ZStack {
            shape.fill(LinearGradient(colors: chip.fill, startPoint: .topLeading, endPoint: .bottomTrailing))
            shape.strokeBorder(tokens.chipEdge, lineWidth: m.chipEdge)
            if kind == .account && known == nil {
                Image(systemName: Token.Object.account).font(.system(size: m.logoKey, weight: .semibold)).foregroundStyle(chip.mark)
            } else {
                Image(Token.Object.mark(known ?? Token.Object.port), bundle: .module)
                    .renderingMode(.template).resizable().scaledToFit()
                    .frame(width: m.logoMark, height: m.logoMark)
                    .foregroundStyle(chip.mark)
            }
        }
        .frame(width: m.logoChip, height: m.logoChip)
    }
}

public struct GlyphView: View {
    public enum Drawn: Equatable, Sendable {
        case symbol(String)
        case custom(String)
        case chip(LogoChip.Kind, String)
        case mac(lid: Bool)
    }

    let glyph: Glyph
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationSamples) private var samples

    public init(_ glyph: Glyph) { self.glyph = glyph }

    public static func drawn(_ glyph: Glyph) -> Drawn {
        let kind = glyph.kind ?? ""
        func after(_ prefix: String) -> String? { kind.hasPrefix(prefix) ? String(kind.dropFirst(prefix.count)) : nil }
        if let brand = after("port:") { return .chip(.port, brand) }
        if let brand = after("account:") { return .chip(.account, brand) }
        if let name = after("symbol:") { return .symbol(name) }
        switch kind {
        case "laptop", "desktop": return glyph.ring != nil ? .mac(lid: kind == "laptop") : .symbol(Token.Object.mac(lid: kind == "laptop"))
        case "server": return .symbol(Token.Object.server)
        case "workspace": return .symbol(Token.Object.workspace)
        case "agent": return .custom(Token.Object.agent)
        default: return .symbol(Token.Object.unknown)
        }
    }

    public var body: some View {
        switch Self.drawn(glyph) {
        case let .chip(kind, brand): LogoChip(kind, brand)
        case let .mac(lid):
            MacRing(lid: lid, share: (samples[glyph.ring ?? ""] ?? 0) / Token.Metric.Marks.percent, asleep: glyph.signal == "asleep")
        case let .symbol(name): slot(Image(systemName: name))
        case let .custom(name): slot(Image(name, bundle: .module))
        }
    }

    func slot(_ image: Image) -> some View {
        let m = Token.Metric.Marks.self
        return image
            .font(.system(size: m.glyph, weight: .medium))
            .foregroundStyle(tokens.signal(glyph.signal))
            .opacity(glyph.signal == "done" ? m.doneShare : 1)
            .frame(width: m.glyphSlot, height: m.glyphSlot)
            .overlay(alignment: .bottomTrailing) { badge.offset(y: m.badgeDrop) }
    }

    @ViewBuilder var badge: some View {
        let m = Token.Metric.Marks.self
        if let progress = glyph.progress {
            ProgressRing(perMille: progress, size: m.badgeProgress)
                .background(Circle().fill(tokens.color.ring))
        } else if glyph.spin == true {
            Spinner(small: true).padding(m.badgeRing / 2).background(Circle().fill(tokens.color.ring))
        } else if let dot = glyph.dot {
            StatusDot(dot, signal: glyph.signal, size: m.badgeDot)
                .padding(m.badgeRing).background(Circle().fill(tokens.color.ring))
                .padding(-m.badgeRing)
        }
    }
}
