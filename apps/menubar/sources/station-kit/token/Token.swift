import SwiftUI

public enum Token {
    public enum Glyph {
        public static let arrow = "→"
        public static let sep = " · "
        public static let ellipsis = "…"
        public static let link = " ›"
        public static let awake = "⏻"
        public static let charging = "⚡"
        public static let unknown = "?"
        public static let dash = "-"
        public static let missing = "—"
        public static let slash = "/"
        public static let command = "⌘"
        public static let restart = "↺"
        public static let caret = "|"
        public static let received = "↓"
        public static let secret = "••••••"
        public static let add = "+ "
        public static let sent = "↑"
        public static let unbroken = "\u{2011}"
    }

    // value: the keys a menu item says and answers (vision a, b).
    public enum Key {
        public static func shown(_ key: String) -> String {
            guard key.hasPrefix("cmd+") else { return key.uppercased() }
            return Glyph.command + key.dropFirst(4).uppercased()
        }
    }

    // value: the measures of the vision's panels (VISION-v3 .panel, .tile,
    // .list): 10 pt between blocks, a 16 pt tile padded 10 by 12, a list's
    // 40 pt row padded 7 by 12, its nested rows from 34 pt, one 24 pt icon
    // tile with a 13 pt glyph whose corner is concentric with the row's, in
    // the 26 pt slot of the vision's icon circle.
    public enum Tile {
        public static let row: CGFloat = 40
        public static let rowPadding = EdgeInsets(top: 7, leading: 12, bottom: 7, trailing: 12)
        public static let icon: CGFloat = 24
        public static let slot: CGFloat = 26
        public static let glyph: CGFloat = 13
        public static let corner: CGFloat = 6.5
        public static let headIcon: CGFloat = 24
        public static let hover: CGFloat = 6
        public static let group: CGFloat = 16
        public static let groupPadding = EdgeInsets(top: 10, leading: 12, bottom: 10, trailing: 12)
        public static let listPadding: CGFloat = 4
        public static let section: CGFloat = 10
        public static let gap: CGFloat = 10
        public static let pair: CGFloat = 8
        public static let indent: CGFloat = 34
        public static let figure: CGFloat = 17
        public static let spark: CGFloat = 26
        public static let chart: CGFloat = 54
        public static let measure: CGFloat = 22
        public static let dot: CGFloat = 7
        public static let chevron: CGFloat = 14
        public static let switchWidth: CGFloat = 38
        public static let switchHeight: CGFloat = 22
        public static let roundButton: CGFloat = 28
        public static let squareButton: CGFloat = 26
        public static let squareCorner: CGFloat = 8
        public static let buttonCorner: CGFloat = 12
        public static let fieldCorner: CGFloat = 10
        public static let qrBox: CGFloat = 172
        public static let qr: CGFloat = 148
        public static let qrCorner: CGFloat = 14
        public static let linkMeasure = 37
        public static let findInset: CGFloat = 3
    }

    // value: the vision's type scale: 13 pt text, 11.5 pt words under it,
    // 11 pt uppercase section titles, 10.5 pt group captions, a 19 pt
    // page title.
    public enum Size {
        public static let text: CGFloat = 13
        public static let sub: CGFloat = 11.5
        public static let label: CGFloat = 11
        public static let caption: CGFloat = 10.5
        public static let tag: CGFloat = 10
        public static let mini: CGFloat = 12
        public static let key: CGFloat = 12
        public static let segment: CGFloat = 12.5
        public static let title: CGFloat = 19
        // value: a health check's mark, the vision's thin 14 pt stroke.
        public static let check: CGFloat = 10
        public static let tracking: CGFloat = 0.04
        // value: half what the vision's 1.35 line height adds to a line
        // of SF's own.
        public static let leading: CGFloat = 0.8
        public static let labelLeading: CGFloat = 0.9
        public static let figureLeading: CGFloat = 1.3
        public static let titleLeading: CGFloat = 1.3
        public static let headLeading: CGFloat = 1.9
    }

    // value: the vision's fills on the panel's glass: a tile white at 8 %
    // in dark, 58 % in light; raised (a chosen segment, a small button)
    // 14 % and 80 %; the hairline between rows; the switch's green.
    public enum Fill {
        public static func tile(_ scheme: ColorScheme) -> Color { Color.white.opacity(scheme == .dark ? 0.08 : 0.58) }
        public static func raised(_ scheme: ColorScheme) -> Color { Color.white.opacity(scheme == .dark ? 0.14 : 0.8) }
        public static func picked(_ scheme: ColorScheme) -> Color { Color.white.opacity(scheme == .dark ? 0.32 : 0.8) }
        public static func faint(_ scheme: ColorScheme) -> Color {
            scheme == .dark ? Token.hex(0xEBEBF5).opacity(0.16) : Token.hex(0x3C3C43).opacity(0.18)
        }
        public static let icon = Token.hex(0x1C1C1E)
        public static let iconInk = Color.white
        public static func edge(_ scheme: ColorScheme) -> Color { Color.white.opacity(scheme == .dark ? 0.14 : 0.7) }
        public static let finding = 0.16
        public static let chip = 0.22
        public static let tag = 0.18
        public static let ring = 0.3
        public static let selected = 0.85
        public static let onTintSub = 0.85
    }

    // value: the vision's hues by scheme: a measure's line (CPU blue,
    // Memory indigo, Network and Disk teal) and the tags.
    public enum Hue {
        public static func blue(_ scheme: ColorScheme) -> Color { Token.hex(scheme == .dark ? 0x2B8CFF : 0x0A7AFF) }
        public static func indigo(_ scheme: ColorScheme) -> Color { Token.hex(scheme == .dark ? 0x6E6FF0 : 0x5B5CE6) }
        public static func teal(_ scheme: ColorScheme) -> Color { Token.hex(scheme == .dark ? 0x2BB8C9 : 0x14A6B8) }
        public static func green(_ scheme: ColorScheme) -> Color { Token.hex(scheme == .dark ? 0x32C75A : 0x2FB350) }

        // value: a measure's hue by its name.
        public static func measure(_ label: String, _ scheme: ColorScheme) -> Color {
            switch label {
            case "CPU": blue(scheme)
            case "Memory": indigo(scheme)
            default: teal(scheme)
            }
        }
    }

    // value: the menu bar icon: the Orbit filled, a custom SF Symbol drawn
    // as the status bar draws its own symbols, 13 pt medium (measured
    // against Spotlight's and a bare status item's on macOS 26), one symbol
    // per state, and the CPU line beside it.
    public enum Icon {
        public static let orbit = "station.orbit"
        public static let badged = "station.orbit.badge"
        public static let slashed = "station.orbit.slash"
        public static let working = "station.orbit.working"
        public static let size: CGFloat = 13
        public static let glyph: CGFloat = 15
        public static let weight = Font.Weight.medium
        public static let line: CGFloat = 1.25
        public static let sparkWidth: CGFloat = 22
        public static let sparkHeight: CGFloat = 12
        public static let gap: CGFloat = 4
        public static let dimmed = 0.4

        public static func needs(_ scheme: ColorScheme) -> Color { Token.hex(scheme == .dark ? 0xFFAD1F : 0xF29D0B) }
        public static func down(_ scheme: ColorScheme) -> Color { Token.hex(scheme == .dark ? 0xFF4F45 : 0xEC3B30) }
        public static func ink(_ scheme: ColorScheme) -> Color { Token.hex(scheme == .dark ? 0xFFFFFF : 0x000000) }
    }

    public enum Space {
        // value: the vision's panel padding, 12 pt inside its 0.5 pt rim.
        public static let margin: CGFloat = 12.5
        public static let indent: CGFloat = 12
        public static let gap: CGFloat = 8
        public static let row: CGFloat = 22
        public static let width: CGFloat = 340
        public static let cap: CGFloat = 560
        public static let qr: CGFloat = 160
        public static let sparkWidth: CGFloat = 48
        public static let sparkHeight: CGFloat = 12
        public static let meterWidth: CGFloat = 36
        public static let meterHeight: CGFloat = 5
        public static let mark: CGFloat = 10
        public static let text: CGFloat = 13
        public static let small: CGFloat = 11
        public static let title: CGFloat = 19
        public static let head: CGFloat = 15
    }

    // value: the stage the Catalog draws a story on: the panel widths its
    // knob offers, the menu bar strip a status item sits in, and the desk
    // left around a panel.
    public enum Stage {
        public static let widths: [CGFloat] = [Space.width, 320, 380]
        public static let scales: [CGFloat] = [1, 2]
        public static let strip = CGSize(width: 600, height: 28)
        public static let room: CGFloat = 24
    }

    // value: the panel an offscreen snapshot draws in place of the
    // app's tinted system glass, which the renderer cannot sample: a
    // translucent pane, its rim and its shadow over the vision's desk.
    public enum Glass {
        public static let window: CGFloat = 24

        // value: the live panel's glass tint, the plate's colour at the
        // lowest alpha of the capture rig's sweep (run cap3, 2026-11-11:
        // 0.5, 0.65, 0.8, 0.9) whose second tone holds 4.5:1 over M1's
        // three wallpapers in both schemes: 0.65, worst 4.90 dark over the
        // light wallpaper, where 0.5 read 4.42 (ADR-036).
        public static func tint(_ scheme: ColorScheme) -> Color { scheme == .dark ? Token.rgba(16, 16, 20, 0.65) : Token.rgba(250, 250, 252, 0.65) }

        public static func pane(_ scheme: ColorScheme) -> Color { scheme == .dark ? Token.hex(0x222328).opacity(0.58) : Token.hex(0xF6F7FB).opacity(0.62) }
        public static func sheen(_ scheme: ColorScheme) -> [Color] {
            [Color.white.opacity(scheme == .dark ? 0.06 : 0.35), Color.white.opacity(0)]
        }
        public static func edge(_ scheme: ColorScheme) -> Color { Fill.edge(scheme) }
        // value: the chosen segment of a segmented control, raised off its
        // track as macOS draws it.
        public static func knob(_ scheme: ColorScheme) -> Color { scheme == .dark ? Color.white.opacity(0.26) : Color.white }
        public static func shade(_ scheme: ColorScheme) -> Color { Color.black.opacity(scheme == .dark ? 0.5 : 0.22) }

        // value: the fixed wallpaper an offscreen snapshot draws the panel
        // over, so its glass reads in the PNG (the vision's desk): each
        // blob's colour, centre, and the share of the desk its ellipse
        // spans before it fades out.
        public static func wallpaper(_ scheme: ColorScheme) -> [(Color, UnitPoint, CGFloat)] {
            let dark = scheme == .dark
            return [
                (Token.hex(dark ? 0x9A3F2A : 0xFF9D6B), UnitPoint(x: 0.12, y: 0.18), 0.78),
                (Token.hex(dark ? 0x1E4FA8 : 0x5AA9FF), UnitPoint(x: 0.88, y: 0.12), 0.71),
                (Token.hex(dark ? 0x0F6E62 : 0x2FC6A8), UnitPoint(x: 0.7, y: 0.95), 0.84),
                (Token.hex(dark ? 0x7C5A12 : 0xF7C948), UnitPoint(x: 0.2, y: 0.9), 0.66),
            ]
        }
        public static func wallBase(_ scheme: ColorScheme) -> [Color] {
            scheme == .dark ? [Token.hex(0x142A5C), Token.hex(0x1B2F55), Token.hex(0x4A2A26)] : [Token.hex(0x3B6FE0), Token.hex(0x7FB6F2), Token.hex(0xF2A477)]
        }
        public static let wallStart = UnitPoint(x: 0.3, y: 0)
        public static let wallEnd = UnitPoint(x: 0.7, y: 1)
    }

    public enum Symbol {
        public static let ok = "circle.fill"
        public static let attention = "circle.lefthalf.filled"
        public static let absent = "circle"
        public static let broken = "xmark.circle.fill"
        public static let pending = "circle.dashed"
        public static let offHere = "nosign"
        public static let notForThisMachine = "minus"
        public static let notKnown = "questionmark"
        public static let error = "xmark.octagon"
        public static let stopped = "stop.circle"
        public static let awake = "sun.max"
        public static let needs = "exclamationmark"
        public static let gear = "ellipsis"
        public static let server = "server.rack"
        public static let accounts = "person.2"
        public static let more = "ellipsis.circle"
        public static let copy = "square.on.square"
        public static let back = "chevron.left"
        public static let down = "chevron.down"
        public static let forward = "chevron.right"
        public static let mac = "desktopcomputer"
        public static let workspace = "shippingbox"
        public static let plus = "plus"
        public static let list = "station"
        public static let prompt = "prompt"
        public static let terminal = "terminal"
        public static let stop = "stop"
        public static let search = "magnifyingglass"
        public static let check = "checkmark"
        public static let cpu = "cpu"
        public static let memory = "memorychip"
        public static let network = "arrow.up.arrow.down"
        public static let warn = "exclamationmark.triangle.fill"
        public static let share = "square.and.arrow.up"
        public static let up = "chevron.up"
        public static let health = "checkmark"
        public static let open = "arrow.up.right.square"
        public static let qr = "qrcode"
        public static let about = "info.circle"
        public static let globe = "globe"
        public static let tailnet = "point.3.connected.trianglepath.dotted"
        public static let lid = "rectangle.compress.vertical"
        public static let battery = "battery.75percent"
        public static let thermo = "thermometer.medium"
        public static let screen = "display"
        public static let ports = "network"
        public static let trash = "trash"
        public static let copyLink = "square.on.square"
        public static let key = "key.fill"
        public static let settings = "gearshape"
        public static let activity = "chart.bar"
        public static let restart = "arrow.counterclockwise"
        public static let unlock = "lock.open"

        // value: a machine's symbol by its kind: a Mac, a server, a
        // workspace.
        public static func machine(_ kind: String?) -> String {
            switch kind {
            case "Mac", "mac": mac
            case "Server", "server": server
            default: workspace
            }
        }

        // value: the tile a known line of a machine's page wears.
        public static func line(_ id: String) -> String? {
            switch id {
            case "apps": "square.grid.2x2"
            case "settings": "gearshape"
            case "disk": "internaldrive"
            case "ports": ports
            case "ssh": terminal
            case "github", "claude", "npm": accounts
            default: nil
            }
        }

        public static func of(_ mark: Mark) -> String {
            switch mark {
            case .ok: ok
            case .attention: attention
            case .absent: absent
            case .broken: broken
            case .pending: pending
            case .offHere: offHere
            case .notForThisMachine: notForThisMachine
            case .notKnown: notKnown
            }
        }
    }

    // value: the system's own colours: the label styles for ink, one
    // accent, the vision's green for a control's glyph (a chevron, a
    // footer line's symbol, a button's), and a state colour only where a
    // state means something (healthy, needs you, failed).
    public struct Palette: Sendable {
        public let text, muted, accent, glyph, good, warn, bad, onTint, qrDark, qrLight: Color

        public func of(_ mark: Mark) -> Color {
            switch mark {
            case .ok: good
            case .attention: warn
            case .broken: bad
            case .absent, .pending, .offHere, .notKnown, .notForThisMachine: muted
            }
        }
    }

    static func hex(_ value: UInt32) -> Color {
        Color(.sRGB, red: Double((value >> 16) & 0xFF) / 255, green: Double((value >> 8) & 0xFF) / 255, blue: Double(value & 0xFF) / 255, opacity: 1)
    }

    static func rgba(_ red: Int, _ green: Int, _ blue: Int, _ alpha: Double) -> Color {
        Color(.sRGB, red: Double(red) / 255, green: Double(green) / 255, blue: Double(blue) / 255, opacity: alpha)
    }

    public static func palette(_ scheme: ColorScheme, accent: Color) -> Palette {
        Palette(
            text: Color.primary, muted: Color.secondary,
            accent: accent, glyph: Hue.green(scheme), good: Color.green, warn: Color.orange, bad: Color.red,
            onTint: Color.white, qrDark: hex(0x000000), qrLight: hex(0xFFFFFF))
    }
}

private struct AccentKey: EnvironmentKey {
    static let defaultValue: Color? = nil
}

private struct HintsKey: EnvironmentKey {
    static let defaultValue = true
}

private struct StillKey: EnvironmentKey {
    static let defaultValue = false
}

private struct NowKey: EnvironmentKey {
    static let defaultValue: Date? = nil
}

extension EnvironmentValues {
    public var stationAccent: Color? {
        get { self[AccentKey.self] }
        set { self[AccentKey.self] = newValue }
    }

    public var stationTokens: Token.Resolved {
        Token.resolved(scheme: colorScheme, contrast: stationContrast, reduceTransparency: stationReduceTransparency, reduceMotion: stationReduceMotion, accent: stationAccent)
    }

    public var stationPalette: Token.Palette { stationTokens.palette }

    public var stationHints: Bool {
        get { self[HintsKey.self] }
        set { self[HintsKey.self] = newValue }
    }

    public var stationStill: Bool {
        get { self[StillKey.self] }
        set { self[StillKey.self] = newValue }
    }

    public var stationNow: Date? {
        get { self[NowKey.self] }
        set { self[NowKey.self] = newValue }
    }
}

extension View {
    public func stationTheme(_ scheme: ColorScheme) -> some View {
        environment(\.colorScheme, scheme)
    }

    // value: the appearance macOS shows, light or dark, and the owner's
    // accent in place of the pinned one.
    public func stationSystemTheme() -> some View { environment(\.stationAccent, Color.accentColor) }
}

extension Token {
    // value: the v4 colour table (SPEC A1), by scheme and contrast, drawn
    // the M1 way: one plate under the content (80 %, M1's 82 % in light
    // less the 2 % a text-dense frame needs to stay glass), which still
    // renders draw in place of the live glass and its tint (Glass.tint),
    // two text tones, cards and Find edged by a faint line, red only as a
    // solid fill. A light card is white at 30 %, not M1's 75 %: over the
    // 80 % plate that would show 5 % of the desk, opaque by the stories'
    // glass rule, and its edge must stay glass under an alert's scrim too;
    // the scrims dim a plate that already dims, so 30 % and 12 % are enough.
    public struct Colors: Sendable {
        public let panel, panelSolid, panelEdge, sheetSolid, menu: Color
        public let plate, plateSolid: Color
        public let card, hero, card2, edge, hair: Color
        public let t1, t2, t3, t4: Color
        public let q, qq, key, track: Color
        public let accent, accentText, green, greenText, orange, orangeText, red, redText, redFill, indigo: Color
        public let scrim, scrimFind, ring, hud: Color

        public func of(_ tone: Tone) -> Color {
            switch tone {
            case .t1: t1
            case .t2: t2
            case .t3: t3
            case .t4: t4
            case .section: t2
            case .hud: Color.white
            }
        }
    }

    public static let accentDark = hex(0x0A84FF)
    public static let accentLight = hex(0x007AFF)

    public static func colors(_ scheme: ColorScheme, contrast: ColorSchemeContrast, accent: Color? = nil) -> Colors {
        let dark = scheme == .dark
        let ic = contrast == .increased
        let panelSolid = dark ? hex(0x2A2A2C) : hex(0xF6F6F8)
        let menu = dark ? rgba(44, 46, 56, ic ? 1 : 0.94) : rgba(246, 246, 248, ic ? 1 : 0.96)
        let card: Color, card2: Color, edge: Color, hair: Color, t2: Color, q: Color, qq: Color
        switch (dark, ic) {
        case (true, false):
            card = rgba(255, 255, 255, 0.07); card2 = rgba(255, 255, 255, 0.11); edge = rgba(255, 255, 255, 0.11)
            hair = rgba(255, 255, 255, 0.12); t2 = rgba(255, 255, 255, 0.72)
            q = rgba(255, 255, 255, 0.14); qq = rgba(255, 255, 255, 0.20)
        case (true, true):
            card = hex(0x2C2D33); card2 = hex(0x3A3B41); edge = rgba(255, 255, 255, 0.30)
            hair = rgba(255, 255, 255, 0.30); t2 = rgba(255, 255, 255, 0.82)
            q = rgba(255, 255, 255, 0.18); qq = rgba(255, 255, 255, 0.26)
        case (false, false):
            card = rgba(255, 255, 255, 0.30); card2 = rgba(255, 255, 255, 0.34); edge = rgba(0, 0, 0, 0.08)
            hair = rgba(0, 0, 0, 0.10); t2 = rgba(0, 0, 0, 0.62)
            q = rgba(0, 0, 0, 0.065); qq = rgba(0, 0, 0, 0.10)
        case (false, true):
            card = hex(0xFFFFFF); card2 = hex(0xF2F2F5); edge = rgba(0, 0, 0, 0.28)
            hair = rgba(0, 0, 0, 0.28); t2 = rgba(0, 0, 0, 0.74)
            q = rgba(0, 0, 0, 0.08); qq = rgba(0, 0, 0, 0.14)
        }
        let pinned = dark ? accentDark : accentLight
        return Colors(
            panel: ic ? panelSolid : (dark ? rgba(30, 31, 38, 0.52) : rgba(236, 236, 242, 0.60)),
            panelSolid: panelSolid,
            panelEdge: dark ? rgba(255, 255, 255, 0.20) : rgba(255, 255, 255, 0.75),
            sheetSolid: dark ? hex(0x272A35) : hex(0xF9F9FB),
            menu: menu,
            plate: dark ? rgba(16, 16, 20, 0.80) : rgba(250, 250, 252, 0.80),
            plateSolid: dark ? hex(0x101014) : hex(0xFAFAFC),
            card: card, hero: card, card2: card2, edge: edge, hair: hair,
            t1: dark ? hex(0xFFFFFF) : hex(0x000000), t2: t2, t3: t2,
            t4: dark ? rgba(235, 235, 245, 0.16) : rgba(60, 60, 67, 0.18),
            q: q, qq: qq,
            key: dark ? rgba(255, 255, 255, 0.14) : rgba(0, 0, 0, 0.07),
            track: dark ? rgba(255, 255, 255, 0.08) : rgba(0, 0, 0, 0.07),
            accent: accent ?? pinned, accentText: accent ?? (dark ? hex(0x5AABFF) : hex(0x0062CC)),
            green: dark ? hex(0x32D74B) : hex(0x28B14C), greenText: dark ? hex(0x32D74B) : hex(0x1A7F37),
            orange: dark ? hex(0xFF9F0A) : hex(0xFF9500), orangeText: dark ? hex(0xFFB340) : hex(0xA85400),
            red: dark ? hex(0xFF453A) : hex(0xFF3B30), redText: dark ? hex(0xFF6961) : hex(0xD70015), redFill: hex(0xD42A20),
            indigo: dark ? hex(0x7D7AFF) : hex(0x5856D6),
            scrim: dark ? rgba(0, 0, 0, 0.30) : rgba(20, 20, 30, 0.12),
            scrimFind: dark ? rgba(0, 0, 0, 0.32) : rgba(20, 20, 30, 0.14),
            ring: dark ? rgba(46, 48, 58, 1) : rgba(246, 247, 251, 1),
            hud: Color.black)
    }

    // value: a brand's chip, its colour in every mode: one fill or a
    // gradient's two, and its mark.
    public struct Chip: Sendable {
        public let fill: [Color]
        public let mark: Color
    }

    public enum Brand: String, CaseIterable, Sendable {
        case github, claude, npm, tailscale, cloudflare, docker, vite, astro, postgres, traefik, kubernetes, node, storybook

        public func chip(_ scheme: ColorScheme) -> Chip {
            let dark = scheme == .dark
            let white = Color.white
            return switch self {
            case .github: dark ? Chip(fill: [hex(0xF0F3F6)], mark: hex(0x1F2328)) : Chip(fill: [hex(0x24292F)], mark: white)
            case .claude: Chip(fill: [hex(0xD97757)], mark: white)
            case .npm: Chip(fill: [hex(0xCB3837)], mark: white)
            case .tailscale: Chip(fill: [hex(dark ? 0x3A3A3C : 0x242424)], mark: white)
            case .cloudflare: Chip(fill: [hex(0xF38020)], mark: white)
            case .docker: Chip(fill: [hex(0x1D63ED)], mark: white)
            case .vite: Chip(fill: [hex(0x41A8FF), hex(0xA245FC)], mark: hex(0xFFD62E))
            case .astro: Chip(fill: [hex(0x17191E)], mark: white)
            case .postgres: Chip(fill: [hex(0x336791)], mark: white)
            case .traefik: Chip(fill: [hex(0x24A1C1)], mark: white)
            case .kubernetes: Chip(fill: [hex(0x326CE5)], mark: white)
            case .node: Chip(fill: [hex(0x3F873F)], mark: white)
            case .storybook: Chip(fill: [hex(0xFF4785)], mark: white)
            }
        }
    }

    // value: the four text tones, of which the panel reads two (t3 is t2,
    // the section label too), and the HUD's white.
    public enum Tone: Sendable { case t1, t2, t3, t4, section, hud }

    // value: a type role of SPEC A1: SF Pro, Rounded or Mono at a size and
    // a weight, its tracking in points, its tone; every value tabular.
    public struct Role: Sendable {
        public let size: CGFloat
        public let weight: Font.Weight
        public let design: Font.Design
        public let tracking: CGFloat
        public let tone: Tone

        init(_ size: CGFloat, _ weight: Font.Weight, _ tone: Tone, design: Font.Design = .default, tracking: CGFloat = 0) {
            self.size = size
            self.weight = weight
            self.design = design
            self.tracking = tracking
            self.tone = tone
        }

        public var font: Font { .system(size: size, weight: weight, design: design).monospacedDigit() }

        public static let heroTitle = Role(17, .bold, .t1, tracking: -0.17)
        public static let pageTitle = Role(17, .bold, .t1)
        public static let homeTitle = Role(13, .bold, .t1)
        public static let sectionLabel = Role(12, .semibold, .section)
        public static let sectionValue = Role(12, .medium, .t3)
        public static let rowName = Role(13, .medium, .t1)
        public static let rowSuffix = Role(11, .regular, .t3)
        public static let rowSub = Role(11, .regular, .t2)
        public static let rowValue = Role(12, .regular, .t2)
        public static let facts = Role(12, .regular, .t2)
        public static let tileCaption = Role(11, .medium, .t2)
        public static let tileNumber = Role(17, .semibold, .t1, design: .rounded)
        public static let tileUnit = Role(11, .regular, .t2)
        public static let gaugeTitle = Role(11, .medium, .t2)
        public static let gaugeValue = Role(15, .semibold, .t1, design: .rounded)
        public static let gaugeUnit = Role(11, .medium, .t2)
        public static let button = Role(12, .semibold, .t1)
        public static let verb = Role(11, .semibold, .t1)
        public static let address = Role(11, .regular, .t1, design: .monospaced)
        public static let findGroup = Role(11, .semibold, .t3)
        public static let findResult = Role(13, .regular, .t1)
        public static let sheetTitle = Role(15, .semibold, .t1)
        public static let sheetBody = Role(12, .regular, .t2)
        public static let alertTitle = Role(13, .bold, .t1)
        public static let alertBody = Role(11, .regular, .t2)
        public static let keyCap = Role(11, .medium, .t2)
        public static let hud = Role(12, .medium, .hud)
        public static let tooltip = Role(11, .regular, .t1)

        public static let all: [(String, Role)] = [
            ("hero title", heroTitle), ("page title", pageTitle), ("home title", homeTitle), ("section label", sectionLabel),
            ("section value", sectionValue), ("row name", rowName), ("row suffix", rowSuffix), ("row sub", rowSub),
            ("row value", rowValue), ("facts", facts), ("tile caption", tileCaption), ("tile number", tileNumber),
            ("tile unit", tileUnit), ("gauge title", gaugeTitle), ("gauge value", gaugeValue), ("gauge unit", gaugeUnit),
            ("button", button), ("verb", verb), ("address", address), ("find group", findGroup), ("find result", findResult),
            ("sheet title", sheetTitle), ("sheet body", sheetBody), ("alert title", alertTitle), ("alert body", alertBody),
            ("key cap", keyCap), ("hud", hud), ("tooltip", tooltip),
        ]
    }

    // value: the v4 metrics (SPEC A1), in points.
    public enum Metric {
        public enum Panel {
            public static let width: CGFloat = 340
            public static let radius: CGFloat = 18
            public static let header = EdgeInsets(top: 10, leading: 14, bottom: 8, trailing: 14)
            public static let body = EdgeInsets(top: 2, leading: 12, bottom: 12, trailing: 12)
            public static let blocks: CGFloat = 8
            public static let label: CGFloat = 4
            public static let footer = EdgeInsets(top: 6, leading: 12, bottom: 12, trailing: 12)
            public static let cap: CGFloat = 700
            public static let screenRoom: CGFloat = 48
            public static let fade: CGFloat = 14
            public static let rim: CGFloat = 1
        }

        public enum Card {
            public static let radius: CGFloat = 12
            public static let padding = EdgeInsets(top: 10, leading: 12, bottom: 10, trailing: 12)
            public static let flush: CGFloat = 4
            public static let highlight: CGFloat = 0.5
            public static let contrastStroke: CGFloat = 1
            public static let edge: CGFloat = 1
        }

        public enum Row {
            public static let height: CGFloat = 34
            public static let twoLines: CGFloat = 38
            public static let padding: CGFloat = 8
            public static let gap: CGFloat = 8
            public static let radius: CGFloat = 8
            public static let accessory: CGFloat = 24
            public static let hairline: CGFloat = 0.5
            public static let inset: CGFloat = 36
            public static let insetGlyph: CGFloat = 40
            public static let insetRing: CGFloat = 44
            public static let insetEnd: CGFloat = 8
        }

        public enum Hero {
            public static let radius: CGFloat = 12
            public static let padding = EdgeInsets(top: 9, leading: 12, bottom: 8, trailing: 12)
            public static let top: CGFloat = 44
            public static let topTitleOnly: CGFloat = 28
            public static let glyph: CGFloat = 26
            public static let slot = CGSize(width: 24, height: 28)
            public static let row: CGFloat = 32
            public static let more: CGFloat = 28
        }

        public enum Tiles {
            public static let columns = 3
            public static let gap: CGFloat = 6
            public static let height: CGFloat = 56
            public static let padding = EdgeInsets(top: 7, leading: 10, bottom: 8, trailing: 10)
            public static let radius: CGFloat = 12
        }

        public enum Gauges {
            public static let columns = 3
            public static let gap: CGFloat = 6
            public static let height: CGFloat = 62
            public static let padding = EdgeInsets(top: 7, leading: 9, bottom: 7, trailing: 9)
            public static let visual: CGFloat = 14
        }

        public struct Control: Sendable {
            public let height: CGFloat
            public let radius: CGFloat
            public let padding: CGFloat

            public static let button = Control(height: 24, radius: 12, padding: 10)
            public static let big = Control(height: 28, radius: 14, padding: 14)
            public static let verb = Control(height: 22, radius: 11, padding: 0)
            public static let verbMinWidth: CGFloat = 64
            public static let round: CGFloat = 24
            public static let roundSmall: CGFloat = 20
            public static let roundTopBar: CGFloat = 26
            public static let icon: CGFloat = 24
            public static let switchSize = CGSize(width: 30, height: 18)
            public static let knob: CGFloat = 14
            public static let segmented = Control(height: 24, radius: 7, padding: 2)
            public static let thumbRadius: CGFloat = 5
            public static let windowPicker: CGFloat = 18
            public static let pill = Control(height: 18, radius: 9, padding: 0)
            public static let keyCap: CGFloat = 16
            public static let keyCapRadius: CGFloat = 4
            public static let keyCapPadding = EdgeInsets(top: 1, leading: 5, bottom: 1, trailing: 5)
            public static let contrastRim: CGFloat = 1
            public static let contrastRimShare = 0.35
        }

        public enum Marks {
            public static let dot: CGFloat = 7
            public static let ringStroke: CGFloat = 1.3
            public static let glyph: CGFloat = 14
            public static let glyphSlot: CGFloat = 24
            public static let macRing: CGFloat = 28
            public static let macRingStroke: CGFloat = 2.6
            public static let macRingRadius: CGFloat = 11.2
            public static let progress: CGFloat = 14
            public static let progressStroke: CGFloat = 2.2
            public static let spinner: CGFloat = 13
            public static let spinnerSmall: CGFloat = 10
            public static let spinnerStroke: CGFloat = 1.6
            public static let logoChip: CGFloat = 22
            public static let logoRadius: CGFloat = 6
            public static let logoMark: CGFloat = 13
            public static let check: CGFloat = 16
        }

        public enum Chart {
            public static let cpu: CGFloat = 80
            public static let chips: CGFloat = 18
            public static let network: CGFloat = 64
            public static let spark = CGSize(width: 60, height: 14)
            public static let uptimeCells = 30
            public static let uptimeHeight: CGFloat = 16
            public static let uptimeGap: CGFloat = 2
            public static let uptimeRadius: CGFloat = 2
        }

        public enum Find {
            public static let field: CGFloat = 30
            public static let fieldRadius: CGFloat = 15
            public static let margin = EdgeInsets(top: 12, leading: 12, bottom: 4, trailing: 12)
            public static let focusShare = 0.55
            public static let result: CGFloat = 30
            public static let resultRadius: CGFloat = 8
            public static let results: CGFloat = 440
            public static let footer: CGFloat = 30
        }

        public enum Sheet {
            public static let inset: CGFloat = 8
            public static let radius: CGFloat = 14
            public static let header = EdgeInsets(top: 12, leading: 16, bottom: 4, trailing: 16)
            public static let body = EdgeInsets(top: 8, leading: 16, bottom: 10, trailing: 16)
            public static let footer = EdgeInsets(top: 10, leading: 12, bottom: 12, trailing: 12)
            public static let labelColumn: CGFloat = 58
            public static let formGap: CGFloat = 10
        }

        public enum Alert {
            public static let width: CGFloat = 268
            public static let padding = EdgeInsets(top: 18, leading: 16, bottom: 14, trailing: 16)
            public static let radius: CGFloat = 16
            public static let icon: CGFloat = 34
            public static let buttonsAbove: CGFloat = 14
        }

        public enum Menu {
            public static let minWidth: CGFloat = 200
            public static let padding: CGFloat = 5
            public static let radius: CGFloat = 11
            public static let item: CGFloat = 24
            public static let itemRadius: CGFloat = 6
            public static let icon: CGFloat = 16
        }

        public enum Popover {
            public static let width: CGFloat = 236
            public static let widthQR: CGFloat = 304
            public static let padding: CGFloat = 12
            public static let paddingQR: CGFloat = 10
            public static let radius: CGFloat = 14
            public static let qr: CGFloat = 112
            public static let qrAlone: CGFloat = 132
        }

        public enum HUD {
            public static let height: CGFloat = 34
            public static let corner: CGFloat = 18
            public static let padding = EdgeInsets(top: 0, leading: 12, bottom: 0, trailing: 16)
            public static let icon: CGFloat = 16
        }

        public enum Tooltip {
            public static let maxWidth: CGFloat = 280
            public static let padding = EdgeInsets(top: 4, leading: 8, bottom: 4, trailing: 8)
            public static let radius: CGFloat = 6
            public static let size: CGFloat = 11
            public static let wait: Double = 1
        }
    }

    // value: the prototype's three curves, its tint, and what each becomes
    // under Reduce Motion.
    public enum Motion {
        public static let snappy = Animation.timingCurve(snappyCurve, duration: 0.20)
        public static let smooth = Animation.timingCurve(smoothCurve, duration: smoothDuration)
        public static let bouncy = Animation.spring(bouncySpring)
        public static let tint = Animation.easeOut(duration: 0.12)
        public static let sample = Animation.linear(duration: 0.2)

        public static let snappyCurve = UnitCurve.bezier(startControlPoint: UnitPoint(x: 0.30, y: 0.90), endControlPoint: UnitPoint(x: 0.30, y: 1))
        public static let smoothCurve = UnitCurve.bezier(startControlPoint: UnitPoint(x: smoothPoints[0], y: smoothPoints[1]), endControlPoint: UnitPoint(x: smoothPoints[2], y: smoothPoints[3]))
        // value: the smooth curve's control points, which the app's window
        // also moves on, since AppKit animates it, not SwiftUI.
        public static let smoothPoints: [Double] = [0.32, 0.72, 0, 1]
        public static let bouncySpring = Spring(duration: 0.225, bounce: 0.42)
        public static let bouncyDuration = 0.46
        public static let bouncyStops: [(at: Double, value: Double)] = [
            (0, 0), (0.05, 0.157), (0.11, 0.53), (0.18, 0.896), (0.25, 1.077), (0.30, 1.115),
            (0.36, 1.093), (0.46, 1.019), (0.56, 0.991), (0.70, 0.995), (1, 1),
        ]

        public static func bouncy(at progress: Double) -> Double {
            bouncySpring.value(target: 1.0, time: min(max(progress, 0), 1) * bouncyDuration)
        }

        public static func resolved(_ animation: Animation, reduce: Bool) -> Animation { reduce ? .easeInOut(duration: 0.15) : animation }
        public static let reducedHeight = Animation.easeInOut(duration: 0.20)
        public static let reducedPage = Animation.easeInOut(duration: 0.24)
        public static let reducedPageGone = 0.45

        // value: when the panel opened is made key, in seconds.
        public static let keyTries: [Double] = [0, 0.05, 0.15, 0.4, 0.8]
    }

    // value: the tokens a view reads, resolved once from the scheme, the
    // kit's contrast, transparency and motion switches, and the accent:
    // the pinned blue unless the app hands it the owner's.
    public struct Resolved: Sendable {
        public let scheme: ColorScheme
        public let increased: Bool
        public let solid: Bool
        public let reduceMotion: Bool
        public let color: Colors
        public let palette: Palette

        public var panel: Color { solid ? color.panelSolid : color.panel }
        public var plate: Color { solid ? color.plateSolid : color.plate }
        public var cardEdge: Color { increased ? color.hair : color.edge }

        public func tone(_ tone: Tone) -> Color { color.of(tone) }

        public func chip(_ id: String?) -> Chip {
            Brand(rawValue: id ?? "")?.chip(scheme) ?? Chip(fill: [color.q], mark: color.t2)
        }

        public func motion(_ animation: Animation) -> Animation { Motion.resolved(animation, reduce: reduceMotion) }
    }

    public static func resolved(scheme: ColorScheme, contrast: ColorSchemeContrast = .standard, reduceTransparency: Bool = false, reduceMotion: Bool = false, accent: Color? = nil) -> Resolved {
        let color = colors(scheme, contrast: contrast, accent: accent)
        return Resolved(
            scheme: scheme, increased: contrast == .increased, solid: contrast == .increased || reduceTransparency, reduceMotion: reduceMotion,
            color: color, palette: palette(scheme, accent: color.accent))
    }
}

extension Token {
    // value: the steps today's views stack and inset by.
    public enum Gap {
        public static let line: CGFloat = 1
        public static let tight: CGFloat = 2
        public static let inner: CGFloat = 3
        public static let small: CGFloat = 4
        public static let flank: CGFloat = 5
        public static let item: CGFloat = 6
        public static let row: CGFloat = 7
        public static let pair: CGFloat = 8
        public static let block: CGFloat = 10
        public static let edge: CGFloat = 12
        public static let section: CGFloat = 14
    }

    // value: the line widths today's views stroke with.
    public enum Stroke {
        public static let hair: CGFloat = 0.5
        public static let rim: CGFloat = 0.6
        public static let ringed: CGFloat = 0.8
        public static let rule: CGFloat = 1
        public static let graph: CGFloat = 1.5
        public static let caret: CGFloat = 1.5
        public static let focus: CGFloat = 2
        public static let open: CGFloat = 3
    }

    // value: the shares a dimmed or fading thing keeps.
    public enum Fade {
        public static let connecting = 0.5
        public static let disabled = 0.4
        public static let off = 0.45
        public static let quiet = 0.75
    }

    // value: a knob's or a raised segment's drop shadow.
    public struct Shadow: Sendable {
        public let color: Color
        public let radius: CGFloat
        public let y: CGFloat

        public static let knob = Shadow(color: Color.black.opacity(0.25), radius: 1.5, y: 1)
        public static let toggle = Shadow(color: Color.black.opacity(0.3), radius: 1.5, y: 1)
        public static let thumb = Shadow(color: Color.black.opacity(0.12), radius: 1.5, y: 1)
        public static let panelRadius: CGFloat = 18
        public static let panelY: CGFloat = 10
    }

    // value: the system's own label and fill styles today's views ink with.
    public enum Ink {
        public static let primary = HierarchicalShapeStyle.primary
        public static let secondary = HierarchicalShapeStyle.secondary
        public static let tertiary = HierarchicalShapeStyle.tertiary
        public static let separator = SeparatorShapeStyle()
        public static let track = AnyShapeStyle(.fill.tertiary)
        public static let idle = AnyShapeStyle(.fill.secondary)
    }

    // value: today's glyph sizes beside their text, and the tracking a
    // big number or a page title is drawn with.
    public enum Detail {
        public static let chevron: CGFloat = 12
        public static let back: CGFloat = 11
        public static let menu: CGFloat = 13
        public static let menuStroke: CGFloat = 14
        public static let check: CGFloat = 12
        public static let knob: CGFloat = 18
        public static let caretHeight: CGFloat = 14
        public static let anchor: CGFloat = 1
        public static let figureTracking: CGFloat = -0.01
        public static let titleTracking: CGFloat = -0.015
        public static let codeCorner: CGFloat = 5
        public static let thumbCorner: CGFloat = 8
        public static let stroke16Box: CGFloat = 11.6
        public static let stroke16Corner: CGFloat = 3.2
        public static let stroke16Inset: CGFloat = 2.2
        public static let roundGlyph: CGFloat = 13
        public static let squareGlyph: CGFloat = 11
        public static let figureShrink = 0.8
    }

    // value: what a line of text is measured by: a character's width and
    // a row's mark in a laid out row, the Activity page's CPU and Memory
    // columns, and what the pixel grid may take off a width it is given.
    public enum Measure {
        public static let char: CGFloat = 6.6
        public static let mark: CGFloat = 12
        public static let column: CGFloat = 56
        public static let grid: CGFloat = 0.5
    }

    // value: the least height a line of today's pages takes.
    public enum Line {
        public static let activity: CGFloat = 36
        public static let machine: CGFloat = 24
        public static let card: CGFloat = 26
        public static let leadChart: CGFloat = 56
        public static let chart: CGFloat = 30
    }
}
