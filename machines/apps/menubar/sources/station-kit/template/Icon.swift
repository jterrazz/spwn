import AppKit
import SwiftUI

public struct MenuBarIcon: Equatable, Sendable {
    public enum Mark: Equatable, Sendable {
        case plain, down, restarting, stopped, connecting
        case incompatible(EngineIncompatibility)
    }

    // value: how the mark is drawn, the six states of the item: the dots
    // plain, slashed, lighting in a wave, dimmed and slashed, the first
    // alone solid. The sixth, the menu open, is the plain mark on the
    // system's highlight, which the system paints.
    public enum Face: Equatable, Sendable {
        case plain, slashed, waving, away, connecting
    }

    public var mark: Mark
    public var busy: Bool
    public var said: String?

    public init(_ mark: Mark, busy: Bool = false, said: String? = nil) {
        self.mark = mark
        self.busy = busy
        self.said = said
    }

    // invariant: SPEC A3: the item draws the page's badge and never judges
    // it: the mark slashed while a server is down or a port exposed, its
    // dots lighting in a wave while a server restarts or a call runs, and
    // the words VoiceOver reads are the badge's own. The engine stopped or
    // not installed dims the slashed mark, so its silence is never mistaken
    // for all good; a connection being made keeps the first dot solid.
    public static func of(connection: Connection, page: Page?) -> MenuBarIcon {
        switch connection {
        case .connecting: return MenuBarIcon(.connecting)
        case .stopped, .notInstalled: return MenuBarIcon(.stopped)
        case .incompatible(let why): return MenuBarIcon(.incompatible(why))
        case .live: break
        }
        guard let badge = page?.badge else { return MenuBarIcon(.plain) }
        let mark: Mark = badge.down == true ? .down : badge.restarting == true ? .restarting : .plain
        return MenuBarIcon(mark, said: badge.label)
    }

    public var label: String {
        if let said { return said }
        switch mark {
        case .stopped: return "Spwn Menu: engine stopped"
        case .incompatible(let why): return "Spwn Menu: \(why.words)"
        case .connecting: return "Spwn Menu: connecting"
        case .plain, .down, .restarting: return "Spwn Menu"
        }
    }

    public var face: Face {
        switch mark {
        case .stopped, .incompatible: return .away
        case .down: return .slashed
        case .connecting: return .connecting
        case .restarting: return .waving
        case .plain: return busy ? .waving : .plain
        }
    }

    public var turning: Bool { face == .waving }
}

// value: the spwn mark, three dots on an arc (`Token.Icon`'s box and dots).
public enum SpwnMark {
    // value: a dot's opacity at a turn of the loop (0 to 1): it runs from
    // `Token.Icon.low` up to 1 and back, peaking `Token.Icon.peak` of the
    // way through its own run, eased in and out both ways; each dot's run
    // starts `Token.Icon.stagger` seconds after the one before.
    public static func light(dot: Int, turn: Double) -> Double {
        var phase = turn - Double(dot) * Token.Icon.stagger / Token.Busy.period
        phase -= phase.rounded(.down)
        let peak = Token.Icon.peak
        let rise = phase < peak ? phase / peak : 1 - (phase - peak) / (1 - peak)
        let eased = rise * rise * (3 - 2 * rise)
        return Token.Icon.low + (1 - Token.Icon.low) * eased
    }

    public static func opacity(dot: Int, face: MenuBarIcon.Face, turn: Double) -> Double {
        switch face {
        case .plain, .slashed: return 1
        case .away: return Token.Icon.dimmed
        case .waving: return light(dot: dot, turn: turn)
        case .connecting: return dot == 0 ? 1 : Token.Icon.resting
        }
    }
}

// invariant: the item has no colour of its own, in any state: macOS paints
// it white on a dark bar, black on a light one, and its states are told by
// the dots' opacity and a slash drawn across them alone.
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

    public var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.img, icon.label, kit: "StatusItem") }

    @ViewBuilder var drawing: some View {
        let scale = Token.Icon.height / Token.Icon.box.height
        let face = icon.face
        Canvas { context, _ in
            context.scaleBy(x: scale, y: scale)
            let slashed = face == .slashed || face == .away
            var line = Path()
            line.move(to: Token.Icon.slashFrom)
            line.addLine(to: Token.Icon.slashTo)
            context.drawLayer { layer in
                for (i, dot) in Token.Icon.dots.enumerated() {
                    let rect = CGRect(x: dot.x - dot.r, y: dot.y - dot.r, width: dot.r * 2, height: dot.r * 2)
                    layer.fill(Path(ellipseIn: rect), with: .color(ink.opacity(SpwnMark.opacity(dot: i, face: face, turn: turn))))
                }
                if slashed {
                    layer.blendMode = .destinationOut
                    layer.stroke(line, with: .color(ink), style: StrokeStyle(lineWidth: Token.Icon.cut, lineCap: .round))
                }
            }
            if slashed {
                let opacity = face == .away ? Token.Icon.dimmed : 1
                context.stroke(line, with: .color(ink.opacity(opacity)), style: StrokeStyle(lineWidth: Token.Icon.stroke, lineCap: .round))
            }
        }
        .frame(width: Token.Icon.box.width * scale, height: Token.Icon.box.height * scale)
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
    // ink whose alpha macOS repaints.
    @MainActor
    public func nsImage(turn: Double = 0) -> NSImage {
        let image = NSImage()
        for scale in [1, 2, 3] as [CGFloat] {
            guard let cg = self.image(scheme: .light, scale: scale, turn: turn) else { continue }
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

// invariant: the status item draws an icon at a turn of its spinner once:
// the same pair asked again draws nothing.
public struct StatusDrawing: Sendable {
    private var last: (icon: MenuBarIcon, turn: Double)?

    public init() {}

    // value: whether this pair is to be drawn, kept as the last drawn if so.
    public mutating func take(_ icon: MenuBarIcon, turn: Double) -> Bool {
        if let last, last.icon == icon, last.turn == turn { return false }
        last = (icon, turn)
        return true
    }
}
