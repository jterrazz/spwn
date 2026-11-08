import AppKit
import SwiftUI

public struct MenuBarIcon: Equatable, Sendable {
    public enum Mark: Equatable, Sendable {
        case plain, red, restarting, stopped, connecting
    }

    public var mark: Mark
    public var count: Int
    public var spark: [Int]?
    public var busy: Bool
    public var said: String?

    public init(_ mark: Mark, count: Int = 0, spark: [Int]? = nil, busy: Bool = false, said: String? = nil) {
        self.mark = mark
        self.count = count
        self.spark = spark
        self.busy = busy
        self.said = said
    }

    // invariant: SPEC A3: the item draws the page's badge and never judges
    // it: the count of what needs you beside the Orbit, the Orbit red while a
    // server is down or a port exposed, a turning mark in place of the red
    // while a server restarts, and the words VoiceOver reads are the
    // badge's own. The engine stopped dims the slashed Orbit, so its silence
    // is never mistaken for all good. The CPU line is the owner's option,
    // drawn only while the engine streams.
    public static func of(connection: Connection, page: Page?, sparkline: Bool = false, live: LiveStore? = nil) -> MenuBarIcon {
        switch connection {
        case .connecting: return MenuBarIcon(.connecting)
        case .stopped, .notInstalled: return MenuBarIcon(.stopped)
        case .live: break
        }
        let cpu = page?.strip.flatMap { live?.points($0.cpu, else: $0.spark) ?? $0.spark }
        let spark = sparkline ? cpu.flatMap { $0.isEmpty ? nil : $0 } : nil
        guard let badge = page?.badge else { return MenuBarIcon(.plain, spark: spark) }
        let mark: Mark = badge.down == true ? .red : badge.restarting == true ? .restarting : .plain
        return MenuBarIcon(mark, count: max(badge.needs ?? 0, 0), spark: spark, said: badge.label)
    }

    public var label: String {
        if let said { return said }
        switch mark {
        case .stopped: return "station: engine stopped"
        case .connecting: return "station: connecting"
        case .plain, .red, .restarting: return "station"
        }
    }

    public var turning: Bool { busy || mark == .restarting }

    public var template: Bool { mark != .red }

    public var symbol: String { mark == .stopped ? Token.Icon.slashed : Token.Icon.orbit }
}

struct IconSpark: Shape {
    let points: [Int]

    func path(in rect: CGRect) -> Path {
        var p = Path()
        guard points.count > 1 else { return p }
        let step = rect.width / CGFloat(points.count - 1)
        for (i, v) in points.enumerated() {
            let share = CGFloat(min(max(v, 0), 100)) / 100
            let pt = CGPoint(x: CGFloat(i) * step, y: rect.height - rect.height * share)
            if i == 0 { p.move(to: pt) } else { p.addLine(to: pt) }
        }
        return p
    }
}

// invariant: the item has no colour of its own but the red of a server
// down: drawn at the status bar's symbol size and weight, macOS paints it
// white on a dark bar, black on a light one; red, it is drawn in the bar's
// ink for the appearance it is drawn for. A connection being made dims the
// Mac and its ring, the machines stay lit.
public struct StatusItem: View {
    let icon: MenuBarIcon
    let scheme: ColorScheme
    let turn: Double

    public init(_ icon: MenuBarIcon, scheme: ColorScheme, turn: Double = 0) {
        self.icon = icon
        self.scheme = scheme
        self.turn = turn
    }

    var ink: Color { Token.Icon.ink(scheme) }

    public var body: some View {
        HStack(spacing: Token.Icon.gap) {
            HStack(spacing: Token.Item.countGap) {
                Image(icon.symbol, bundle: .module)
                    .font(.system(size: Token.Icon.size, weight: Token.Icon.weight))
                    .imageScale(.medium)
                    .symbolRenderingMode(icon.mark == .connecting ? .hierarchical : .monochrome)
                    .foregroundStyle(icon.mark == .red ? Token.Item.red(scheme) : ink)
                    .opacity(icon.mark == .stopped ? Token.Icon.dimmed : 1)
                if icon.turning {
                    BusyMark(turn: turn, ink: ink)
                }
                if icon.count > 0 {
                    Text(String(icon.count))
                        .font(.system(size: Token.Icon.size, weight: Token.Icon.weight))
                        .monospacedDigit()
                        .foregroundStyle(ink)
                }
            }
            if let spark = icon.spark {
                IconSpark(points: spark)
                    .stroke(ink, style: StrokeStyle(lineWidth: Token.Icon.line, lineCap: .round, lineJoin: .round))
                    .frame(width: Token.Icon.sparkWidth, height: Token.Icon.sparkHeight)
            }
        }
    }
}

struct BusyMark: View {
    let turn: Double
    let ink: Color

    var body: some View {
        Circle()
            .trim(from: Token.Busy.open, to: 1)
            .stroke(ink, style: StrokeStyle(lineWidth: Token.Busy.stroke, lineCap: .round))
            .rotationEffect(Token.Busy.start + .radians(turn * .pi * 2))
            .frame(width: Token.Busy.size, height: Token.Busy.size)
    }
}

extension MenuBarIcon {
    // value: the icon as the menu bar's label draws it: rendered offscreen,
    // at the screen's scale, for the menu bar's appearance.
    @MainActor
    public func image(scheme: ColorScheme, scale: CGFloat = 2, turn: Double = 0) -> CGImage? {
        let renderer = ImageRenderer(content: StatusItem(self, scheme: scheme, turn: turn))
        renderer.scale = scale
        return renderer.cgImage
    }

    // value: the label's image with a bitmap per screen scale, so the mark
    // lands on whole pixels at 1x, 2x and 3x alike; a template in the one
    // ink whose alpha macOS repaints, the red in the bar's own.
    @MainActor
    public func nsImage(scheme: ColorScheme, turn: Double = 0) -> NSImage {
        let image = NSImage()
        let drawn = template ? ColorScheme.light : scheme
        for scale in [1, 2, 3] as [CGFloat] {
            guard let cg = self.image(scheme: drawn, scale: scale, turn: turn) else { continue }
            let rep = NSBitmapImageRep(cgImage: cg)
            rep.size = NSSize(width: CGFloat(cg.width) / scale, height: CGFloat(cg.height) / scale)
            image.addRepresentation(rep)
            image.size = rep.size
        }
        image.isTemplate = template
        image.accessibilityDescription = label
        return image
    }
}
