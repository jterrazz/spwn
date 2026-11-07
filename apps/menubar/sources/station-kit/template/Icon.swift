import AppKit
import SwiftUI

public struct MenuBarIcon: Equatable, Sendable {
    public enum Mark: Equatable, Sendable {
        case plain, needs(Int), down, stopped, connecting
    }

    public var mark: Mark
    public var spark: [Int]?

    public init(_ mark: Mark, spark: [Int]? = nil) {
        self.mark = mark
        self.spark = spark
    }

    // invariant: SPEC §10.1: the icon draws the page's badge and never
    // judges it: a server down wins over any count, something needing you
    // is a dot cut into the Orbit's corner, and nothing needing you is the
    // Orbit alone; the engine stopped dims the slashed Orbit, so its silence
    // is never mistaken for all good. The CPU line is the owner's option,
    // drawn only while the engine streams.
    public static func of(connection: Connection, page: Page?, sparkline: Bool = false) -> MenuBarIcon {
        switch connection {
        case .connecting: return MenuBarIcon(.connecting)
        case .stopped, .notInstalled: return MenuBarIcon(.stopped)
        case .live: break
        }
        let spark = sparkline ? page?.strip?.spark.flatMap { $0.isEmpty ? nil : $0 } : nil
        guard let badge = page?.badge else { return MenuBarIcon(.plain, spark: spark) }
        if badge.down == true { return MenuBarIcon(.down, spark: spark) }
        if let n = badge.needs, n > 0 { return MenuBarIcon(.needs(n), spark: spark) }
        return MenuBarIcon(.plain, spark: spark)
    }

    public var label: String {
        switch mark {
        case .plain: "station: nothing needs you"
        case .needs(1): "station: 1 thing needs you"
        case .needs(let n): "station: \(n) things need you"
        case .down: "station: a server is down"
        case .stopped: "station: engine stopped"
        case .connecting: "station: connecting"
        }
    }

    // value: the custom symbol each state draws.
    public var symbol: String {
        switch mark {
        case .plain, .connecting: Token.Icon.orbit
        case .needs: Token.Icon.badged
        case .down, .stopped: Token.Icon.slashed
        }
    }
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

// invariant: the Orbit is a symbol with no colour of its own, drawn at the
// status bar's symbol size and weight, so macOS paints it white on a dark
// bar, black on a light one, and tints it while the panel is open; a
// connection being made dims the Mac and its ring, the machines stay lit.
public struct MenuBarIconView: View {
    let icon: MenuBarIcon
    let scheme: ColorScheme

    public init(_ icon: MenuBarIcon, scheme: ColorScheme) {
        self.icon = icon
        self.scheme = scheme
    }

    var ink: Color { Token.Icon.ink(scheme) }

    public var body: some View {
        HStack(spacing: Token.Icon.gap) {
            Image(icon.symbol, bundle: .module)
                .font(.system(size: Token.Icon.size, weight: Token.Icon.weight))
                .imageScale(.medium)
                .symbolRenderingMode(icon.mark == .connecting ? .hierarchical : .monochrome)
                .foregroundStyle(ink)
                .opacity(icon.mark == .stopped ? Token.Icon.dimmed : 1)
            if let spark = icon.spark {
                IconSpark(points: spark)
                    .stroke(ink, style: StrokeStyle(lineWidth: Token.Icon.line, lineCap: .round, lineJoin: .round))
                    .frame(width: Token.Icon.sparkWidth, height: Token.Icon.sparkHeight)
            }
        }
    }
}

extension MenuBarIcon {
    // value: the icon as the menu bar's label draws it: rendered offscreen,
    // at the screen's scale, for the menu bar's appearance.
    @MainActor
    public func image(scheme: ColorScheme, scale: CGFloat = 2) -> CGImage? {
        let renderer = ImageRenderer(content: MenuBarIconView(self, scheme: scheme))
        renderer.scale = scale
        return renderer.cgImage
    }

    // value: the label's image with a bitmap per screen scale, so the mark
    // lands on whole pixels at 1x, 2x and 3x alike.
    @MainActor
    public func nsImage(scheme: ColorScheme) -> NSImage {
        let image = NSImage()
        for scale in [1, 2, 3] as [CGFloat] {
            guard let cg = self.image(scheme: scheme, scale: scale) else { continue }
            let rep = NSBitmapImageRep(cgImage: cg)
            rep.size = NSSize(width: CGFloat(cg.width) / scale, height: CGFloat(cg.height) / scale)
            image.addRepresentation(rep)
            image.size = rep.size
        }
        image.isTemplate = true
        image.accessibilityDescription = label
        return image
    }
}
