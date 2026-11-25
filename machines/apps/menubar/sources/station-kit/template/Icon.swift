import AppKit
import SwiftUI

public struct MenuBarIcon: Equatable, Sendable {
    public enum Mark: Equatable, Sendable {
        case plain, down, restarting, stopped, connecting
        case incompatible(EngineIncompatibility)
    }

    // value: how the mark is drawn, the six states of the item: the
    // dots behind you plain, faded under a grown front dot, lighting in
    // turn, dimmed and slashed, the nearest one alone solid. The sixth, the menu open, is the
    // plain mark on the system's highlight, which the system paints.
    public enum Face: Equatable, Sendable {
        case plain, needs, waving, away, connecting
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
    // it: the dots behind faded under a grown front dot while a server is
    // down or a port exposed, lighting in turn while a server restarts or a
    // call runs, and the words VoiceOver reads are the badge's own. The
    // engine stopped or not installed dims and slashes the mark, so its
    // silence is never mistaken for all good; a connection being made keeps
    // the nearest dot behind alone solid.
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
        case .down: return .needs
        case .connecting: return .connecting
        case .restarting: return .waving
        case .plain: return busy ? .waving : .plain
        }
    }

    public var turning: Bool { face == .waving }
}

// value: the spwn mark, three dots one behind the other, the front one you (`Token.Icon`).
public enum SpwnMark {
    // value: a dot behind's opacity at a turn of the loop (0 to 1): it runs
    // from `Token.Icon.low` up to 1 and back, peaking `Token.Icon.peak` of
    // the way through its own run, eased in and out both ways; each dot's
    // run starts `Token.Icon.stagger` seconds after the one nearer you.
    public static func light(behind: Int, turn: Double) -> Double {
        var phase = turn - Double(behind) * Token.Icon.stagger / Token.Busy.period
        phase -= phase.rounded(.down)
        let peak = Token.Icon.peak
        let rise = phase < peak ? phase / peak : 1 - (phase - peak) / (1 - peak)
        let eased = rise * rise * (3 - 2 * rise)
        return Token.Icon.low + (1 - Token.Icon.low) * eased
    }

    // value: a dot behind's opacity in a face, 0 the one nearest you.
    public static func opacity(behind: Int, face: MenuBarIcon.Face, turn: Double) -> Double {
        switch face {
        case .plain: return 1
        case .needs: return Token.Icon.faded
        case .away: return Token.Icon.dimmed
        case .waving: return light(behind: behind, turn: turn)
        case .connecting: return behind == 0 ? 1 : Token.Icon.resting
        }
    }

    public static func dotOpacity(_ face: MenuBarIcon.Face) -> Double {
        switch face {
        case .plain, .needs, .waving: return 1
        case .away: return Token.Icon.dimmed
        case .connecting: return Token.Icon.resting
        }
    }

    public static func dot(_ face: MenuBarIcon.Face) -> (x: CGFloat, y: CGFloat, r: CGFloat) {
        face == .needs ? Token.Icon.needsDot : Token.Icon.dot
    }

    public static func circle(_ dot: (x: CGFloat, y: CGFloat, r: CGFloat), grown: CGFloat = 0) -> Path {
        let r = dot.r + grown
        return Path(ellipseIn: CGRect(x: dot.x - r, y: dot.y - r, width: r * 2, height: r * 2))
    }
}

// invariant: the item has no colour of its own, in any state: macOS paints
// it white on a dark bar, black on a light one, and its states are told by
// the opacity of its dots, the front dot's size and a slash alone.
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
        let face = icon.face
        let dot = SpwnMark.dot(face)
        Canvas { context, _ in
            var line = Path()
            line.move(to: Token.Icon.slashFrom)
            line.addLine(to: Token.Icon.slashTo)
            context.drawLayer { layer in
                let behind = Token.Icon.behind
                for (i, back) in behind.enumerated() {
                    layer.drawLayer { crescent in
                        crescent.fill(SpwnMark.circle(back), with: .color(ink.opacity(SpwnMark.opacity(behind: i, face: face, turn: turn))))
                        crescent.blendMode = .destinationOut
                        crescent.fill(SpwnMark.circle(i == 0 ? dot : behind[i - 1], grown: Token.Icon.gap), with: .color(ink))
                    }
                }
                layer.fill(SpwnMark.circle(dot), with: .color(ink.opacity(SpwnMark.dotOpacity(face))))
                if face == .away {
                    layer.blendMode = .destinationOut
                    layer.stroke(line, with: .color(ink), style: StrokeStyle(lineWidth: Token.Icon.cut, lineCap: .round))
                }
            }
            if face == .away {
                context.stroke(line, with: .color(ink), style: StrokeStyle(lineWidth: Token.Icon.stroke, lineCap: .round))
            }
        }
        .frame(width: Token.Icon.box.width, height: Token.Icon.box.height)
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
