import SwiftUI

// invariant: the MenuBarExtra's window is the only glass. Live the panel
// paints nothing; drawn still (an offscreen snapshot, which cannot sample
// what lies behind) it stands in for that window. Its corner is the
// container shape every grouping inside is concentric with.
struct PanelGlass: ViewModifier {
    @Environment(\.stationStill) private var still
    @Environment(\.colorScheme) private var scheme

    func body(content: Content) -> some View {
        let shape = RoundedRectangle(cornerRadius: Token.Glass.window, style: .continuous)
        if still {
            content
                .containerShape(shape)
                .background {
                    ZStack {
                        shape.fill(Token.Glass.pane(scheme))
                        shape.fill(LinearGradient(colors: Token.Glass.sheen(scheme), startPoint: .top, endPoint: .center))
                        shape.strokeBorder(Token.Glass.edge(scheme), lineWidth: Token.Stroke.rim)
                    }
                    .shadow(color: Token.Glass.shade(scheme), radius: Token.Shadow.panelRadius, y: Token.Shadow.panelY)
                }
        } else {
            content.containerShape(shape)
        }
    }
}

// value: a tile of the vision on the panel's glass: a translucent fill
// 16 pt round (white at 8 % in dark, 58 % in light), never glass, padded
// 10 by 12; a finding mixes its colour in. With Reduce Transparency or
// Increase Contrast, a separator stroke in its place.
struct TileGround: ViewModifier {
    let padding: EdgeInsets
    let tint: Color?
    @Environment(\.stationTokens) private var tokens
    @Environment(\.colorScheme) private var scheme

    func body(content: Content) -> some View {
        let shape = RoundedRectangle(cornerRadius: Token.Tile.group, style: .continuous)
        content
            .padding(padding)
            .frame(maxWidth: .infinity, alignment: .topLeading)
            .background {
                if tokens.solid {
                    shape.strokeBorder(Token.Ink.separator, lineWidth: Token.Stroke.rule)
                } else {
                    ZStack {
                        shape.fill(Token.Fill.tile(scheme))
                        if let tint { shape.fill(tint.opacity(Token.Fill.finding)) }
                    }
                }
            }
    }
}

// value: a row the pointer is over lights up across the whole row.
struct Hover: ViewModifier {
    @State private var over = false
    @Environment(\.colorScheme) private var scheme

    func body(content: Content) -> some View {
        content
            .contentShape(Rectangle())
            .background {
                if over { Rectangle().fill(Token.Fill.tile(scheme)) }
            }
            .onHover { over = $0 }
    }
}

extension View {
    // value: the panel's own window (see PanelGlass).
    public func stationPanel() -> some View { modifier(PanelGlass()) }

    // value: a tile (see TileGround).
    public func stationTile(padding: EdgeInsets = Token.Tile.groupPadding, tint: Color? = nil) -> some View { modifier(TileGround(padding: padding, tint: tint)) }

    // value: a list of the vision: a tile whose rows run edge to edge,
    // 4 pt above the first and under the last.
    public func stationRows() -> some View {
        modifier(TileGround(padding: EdgeInsets(top: Token.Tile.listPadding, leading: 0, bottom: Token.Tile.listPadding, trailing: 0), tint: nil))
    }

    public func stationHover() -> some View { modifier(Hover()) }

    // value: a field's ground (the switcher's search): a tile's fill,
    // 10 pt round.
    public func stationField() -> some View { modifier(FieldGround()) }
}

struct FieldGround: ViewModifier {
    @Environment(\.colorScheme) private var scheme

    func body(content: Content) -> some View {
        content.background(Token.Fill.tile(scheme), in: RoundedRectangle(cornerRadius: Token.Tile.fieldCorner, style: .continuous))
    }
}

// value: the hairline between two rows of a list, edge to edge.
public struct RowSeparator: View {
    @Environment(\.colorScheme) private var scheme

    public init() {}

    public var body: some View { Rectangle().fill(Token.Fill.faint(scheme)).frame(height: Token.Stroke.hair) }
}

// value: the rows of a list, a hairline between each two.
public struct Rows<Content: View>: View {
    let content: Content

    public init(@ViewBuilder content: () -> Content) { self.content = content() }

    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            SwiftUI.Group(subviews: content) { subviews in
                ForEach(Array(subviews.enumerated()), id: \.offset) { i, sub in
                    if i > 0 { RowSeparator() }
                    sub
                }
            }
        }
        .stationRows()
    }
}

// value: a section's title as the vision's: 11 pt semibold, uppercase,
// tracked, secondary.
public struct SectionHeader: View {
    let title: String

    public init(_ title: String) { self.title = title }

    // value: the header as it is drawn: its words upper-cased, a value
    // kept as it is said: from a part's first figure to its end, its unit
    // and its age with it (HEALTH, AFTER EVERY BOOT, 57 s ago).
    public static func cased(_ title: String) -> String {
        title.components(separatedBy: Token.Glyph.sep).map { part in
            var out: [String] = []
            var value = false
            for word in part.split(separator: " ", omittingEmptySubsequences: false) {
                value = value || word.first?.isNumber == true
                out.append(value ? String(word) : word.uppercased())
            }
            return out.joined(separator: " ")
        }.joined(separator: Token.Glyph.sep)
    }

    public var body: some View {
        Wrap(Self.cased(title)).font(.system(size: Token.Size.label, weight: .semibold)).tracking(Token.Size.label * Token.Size.tracking)
            .foregroundStyle(Token.Ink.secondary).lineLimit(2).fixedSize(horizontal: false, vertical: true)
            .padding(.horizontal, Token.Gap.small).padding(.top, Token.Gap.tight)
    }
}

// value: a group's caption inside a list (Workspaces, Servers): 10.5 pt
// semibold, uppercase, tracked, from the nested rows' edge.
public struct GroupCaption: View {
    let title: String

    public init(_ title: String) { self.title = title }

    public var body: some View {
        Text(title.uppercased()).font(.system(size: Token.Size.caption, weight: .semibold)).tracking(Token.Size.caption * Token.Size.tracking)
            .foregroundStyle(Token.Ink.secondary)
            .padding(.leading, Token.Tile.indent).padding(.top, Token.Gap.item).padding(.trailing, Token.Gap.edge)
            .frame(maxWidth: .infinity, alignment: .leading)
    }
}

// value: a row's icon: a 24 pt near-black rounded square in both
// schemes, its corner concentric with the row's hover, its 13 pt symbol
// hierarchical in white, centred in the 26 pt the vision's icon circle takes so
// the words beside it start where the vision's do; the accent (or the colour a finding means) tints only the
// glyph, never the tile. Increase Contrast draws its separator edge.
public struct IconTile: View {
    let symbol: String
    let active: Bool
    let tint: Color?
    let size: CGFloat
    let pulse: Bool
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationStill) private var still
    @Environment(\.stationTokens) private var tokens

    public init(_ symbol: String, active: Bool = false, tint: Color? = nil, size: CGFloat = Token.Tile.icon, pulse: Bool = false) {
        self.symbol = symbol
        self.active = active
        self.tint = tint
        self.size = size
        self.pulse = pulse
    }

    // invariant: a finding's glyph is drawn in its colour's bright form on
    // the near-black tile, whatever the scheme, so it reads in light mode.
    var ink: AnyShapeStyle {
        if let tint, tint == palette.warn { return AnyShapeStyle(Token.Icon.needs(.dark)) }
        if let tint, tint == palette.bad { return AnyShapeStyle(Token.Icon.down(.dark)) }
        if let tint { return AnyShapeStyle(tint) }
        return active ? AnyShapeStyle(palette.accent) : AnyShapeStyle(Token.Fill.iconInk)
    }

    public var body: some View {
        let scale = size / Token.Tile.icon
        let shape = RoundedRectangle(cornerRadius: Token.Tile.corner * scale, style: .continuous)
        Image(systemName: symbol)
            .symbolRenderingMode(tint == nil ? .hierarchical : .monochrome)
            .font(.system(size: Token.Tile.glyph * scale, weight: .medium))
            .foregroundStyle(ink)
            .environment(\.colorScheme, .dark)
            .compositingGroup()
            .contentTransition(.symbolEffect(.replace))
            .symbolEffect(.pulse, isActive: pulse && !still)
            .frame(width: size, height: size)
            .background {
                shape.fill(Token.Fill.icon)
                if tokens.increased { shape.strokeBorder(Token.Ink.separator, lineWidth: Token.Stroke.rule) }
            }
            .frame(width: size == Token.Tile.icon ? Token.Tile.slot : size)
    }
}

// invariant: a row as the vision's lists draw one: at least 40 pt,
// padded 7 by 12, what leads it, its title semibold, its words under it
// secondary, what ends it; the whole row lights up under the pointer and
// is one button when it opens or does something. Bare, in a tile, it
// takes the tile's padding instead.
public struct MenuRow<Leading: View, Trailing: View>: View {
    let title: String
    let subtitle: String?
    let link: Bool
    let lines: Int
    let titleLines: Int
    let bare: Bool
    let top: Bool
    let tap: (() -> Void)?
    let leading: Leading
    let trailing: Trailing

    public init(_ title: String, subtitle: String? = nil, link: Bool = false, lines: Int = 2, titleLines: Int = 1, bare: Bool = false, top: Bool = false, tap: (() -> Void)? = nil,
                @ViewBuilder leading: () -> Leading, @ViewBuilder trailing: () -> Trailing) {
        self.top = top
        self.titleLines = titleLines
        self.title = title
        self.subtitle = subtitle
        self.link = link
        self.lines = lines
        self.bare = bare
        self.tap = tap
        self.leading = leading()
        self.trailing = trailing()
    }

    var line: some View {
        HStack(spacing: Token.Tile.gap) {
            leading.frame(maxHeight: top ? .infinity : nil, alignment: .top)
            RowText(title: title, subtitle: subtitle, link: link, lines: lines, titleLines: titleLines)
                .frame(maxWidth: .infinity, alignment: .leading)
            trailing.layoutPriority(1)
        }
        .fixedSize(horizontal: false, vertical: true)
        .padding(bare ? EdgeInsets() : Token.Tile.rowPadding)
        .frame(minHeight: bare ? 0 : Token.Tile.row)
        .stationHover()
    }

    public var body: some View {
        if let tap {
            Button(action: tap) { line }.buttonStyle(.plain)
        } else {
            line
        }
    }
}

extension MenuRow where Trailing == EmptyView {
    public init(_ title: String, subtitle: String? = nil, link: Bool = false, bare: Bool = false, tap: (() -> Void)? = nil, @ViewBuilder leading: () -> Leading) {
        self.init(title, subtitle: subtitle, link: link, bare: bare, tap: tap, leading: leading) { EmptyView() }
    }
}

// value: a row's words: its title, 13 pt semibold, what follows it (a
// kind's tag), its words under it in 11.5 pt secondary or its address.
public struct RowText<Tag: View>: View {
    let title: String
    let subtitle: String?
    let link: Bool
    let lines: Int
    let titleLines: Int
    let tag: Tag

    // value: a title this short is never cut: a value beside it gives way.
    static var whole: Int { 16 }

    public init(title: String, subtitle: String?, link: Bool = false, lines: Int = 2, titleLines: Int = 1, @ViewBuilder tag: () -> Tag) {
        self.title = title
        self.subtitle = subtitle
        self.link = link
        self.lines = lines
        self.titleLines = titleLines
        self.tag = tag()
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Gap.line) {
            HStack(alignment: .center, spacing: Token.Gap.item) {
                Text(title).font(.system(size: Token.Size.text, weight: .semibold)).foregroundStyle(Token.Ink.primary).lineLimit(titleLines)
                    .fixedSize(horizontal: title.count <= Self.whole, vertical: titleLines > 1)
                    .padding(.vertical, Token.Size.leading)
                tag
            }
            if let subtitle, !subtitle.isEmpty {
                if link {
                    LinkText(subtitle)
                } else {
                    SubText(subtitle).lineLimit(lines)
                }
            }
        }
    }
}

extension RowText where Tag == EmptyView {
    public init(title: String, subtitle: String?, link: Bool = false, lines: Int = 2, titleLines: Int = 1) {
        self.init(title: title, subtitle: subtitle, link: link, lines: lines, titleLines: titleLines) { EmptyView() }
    }
}

// value: the vision's `.sub`: 11.5 pt, secondary.
public struct SubText: View {
    let text: String

    public init(_ text: String) { self.text = text }

    public var body: some View {
        Wrap(text, spacing: Token.Size.leading * 2).font(.system(size: Token.Size.sub)).foregroundStyle(Token.Ink.secondary)
            .padding(.vertical, Token.Size.leading).fixedSize(horizontal: false, vertical: true)
    }
}

// value: a row's value on its trailing edge: secondary, its digits
// monospaced.
public struct TrailingValue: View {
    let text: String

    public init(_ text: String) { self.text = text }

    public var body: some View {
        Text(text).font(.system(size: Token.Size.sub)).foregroundStyle(Token.Ink.secondary).monospacedDigit().lineLimit(1).truncationMode(.middle)
            .contentTransition(.numericText())
    }
}

// value: an address as the vision sets one: 11 pt monospaced, in the
// accent, cut at its end; centred, it wraps every 37 characters, the
// measure the vision's QR card breaks it at.
public struct LinkText: View {
    let text: String
    let center: Bool
    @Environment(\.stationPalette) private var palette

    public init(_ text: String, center: Bool = false) {
        self.text = text
        self.center = center
    }

    public var body: some View {
        Text(center ? lines : text).font(.system(size: Token.Size.label, design: .monospaced)).foregroundStyle(palette.accent)
            .lineLimit(center ? 3 : 1).truncationMode(.tail).multilineTextAlignment(center ? .center : .leading)
            .textSelection(.enabled)
    }

    var lines: String {
        let chars = Array(text)
        return stride(from: 0, to: chars.count, by: Token.Tile.linkMeasure).map { String(chars[$0..<min($0 + Token.Tile.linkMeasure, chars.count)]) }.joined(separator: "\n")
    }
}

// value: the chevron a row that opens a page ends with, in the vision's
// 14 pt glyph box.
public struct Chevron: View {
    let down: Bool
    @Environment(\.stationPalette) private var palette

    public init(down: Bool = false) { self.down = down }

    public var body: some View {
        Image(systemName: down ? Token.Symbol.down : Token.Symbol.forward).font(.system(size: Token.Detail.chevron, weight: .semibold)).foregroundStyle(palette.glyph)
            .frame(width: Token.Tile.chevron)
    }
}

// value: a machine's state as a dot: filled in its mark's colour, hollow
// when it is not running; the only colour a machine's line wears.
public struct StateDot: View {
    let mark: Mark?
    @Environment(\.stationPalette) private var palette

    public init(mark: Mark?) { self.mark = mark }

    public var body: some View {
        if let mark {
            if mark == .pending {
                PendingDot()
            } else if mark == .absent {
                Circle().strokeBorder(Token.Ink.secondary, lineWidth: Token.Stroke.graph).frame(width: Token.Tile.dot, height: Token.Tile.dot)
            } else {
                Circle().fill(palette.of(mark)).frame(width: Token.Tile.dot, height: Token.Tile.dot)
            }
        }
    }
}

public enum StationRole { case plain, primary, danger, quiet }

// invariant: a button as the vision's: a page's buttons side by side as
// wide tiles 12 pt round (Restart, Copy link filled in the accent; Delete
// in red; Stop greyed while its sheet asks for the name), a row's small
// capsule (Take back, Review, Change).
public struct StationButton: View {
    let title: String
    let role: StationRole
    let wide: Bool
    let action: () -> Void
    @Environment(\.isEnabled) private var enabled
    @Environment(\.stationPalette) private var palette
    @Environment(\.colorScheme) private var scheme

    public init(_ title: String, role: StationRole = .plain, wide: Bool = false, action: @escaping () -> Void = {}) {
        self.title = title
        self.role = role
        self.wide = wide
        self.action = action
    }

    var ink: AnyShapeStyle {
        switch role {
        case .primary: AnyShapeStyle(palette.onTint)
        case .danger: AnyShapeStyle(palette.bad)
        case .quiet: AnyShapeStyle(Token.Ink.secondary)
        case .plain: AnyShapeStyle(Token.Ink.primary)
        }
    }

    var fill: AnyShapeStyle {
        if role == .primary { return AnyShapeStyle(palette.accent) }
        return AnyShapeStyle(wide ? Token.Fill.tile(scheme) : Token.Fill.raised(scheme))
    }

    public var body: some View {
        Button(action: action) {
            SwiftUI.Group {
                if wide {
                    Wrap(title, alignment: .center).font(.system(size: Token.Size.text, weight: role == .quiet ? .medium : .semibold))
                        .fixedSize(horizontal: false, vertical: true)
                        .padding(.vertical, Token.Size.leading)
                        .padding(Token.Gap.pair).frame(maxWidth: .infinity, maxHeight: .infinity)
                        .background(fill, in: RoundedRectangle(cornerRadius: Token.Tile.buttonCorner, style: .continuous))
                        .opacity(role == .quiet ? Token.Fade.quiet : 1)
                } else {
                    Text(title).font(.system(size: Token.Size.mini, weight: .semibold)).lineLimit(1).fixedSize()
                        .padding(.vertical, Token.Size.leading)
                        .padding(.horizontal, Token.Gap.block).padding(.vertical, Token.Gap.flank)
                        .background(fill, in: Capsule())
                }
            }
            .foregroundStyle(ink)
            .opacity(enabled ? 1 : Token.Fade.off)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

// invariant: a switch as the vision draws one: 38 by 22, green when on,
// its white knob shadowed; one control, live or drawn still.
public struct StationSwitch: View {
    let on: Bool
    let tap: () -> Void
    @Environment(\.stationPalette) private var palette
    @Environment(\.colorScheme) private var scheme

    public init(on: Bool, tap: @escaping () -> Void = {}) {
        self.on = on
        self.tap = tap
    }

    public var body: some View {
        Button(action: tap) {
            ZStack(alignment: on ? .trailing : .leading) {
                Capsule().fill(on ? AnyShapeStyle(palette.good) : AnyShapeStyle(Token.Fill.faint(scheme)))
                    .frame(width: Token.Tile.switchWidth, height: Token.Tile.switchHeight)
                Circle().fill(palette.onTint).frame(width: Token.Detail.knob, height: Token.Detail.knob)
                    .shadow(color: Token.Shadow.toggle.color, radius: Token.Shadow.toggle.radius, y: Token.Shadow.toggle.y).padding(.horizontal, Token.Gap.tight)
            }
            .contentShape(Capsule())
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(.isToggle)
        .accessibilityValue(on ? "on" : "off")
    }
}

// value: a text button, the accent's or, for what destroys, red.
public struct TextAction: View {
    let label: String
    let symbol: String?
    let danger: Bool
    let tap: () -> Void
    @Environment(\.stationPalette) private var palette

    public init(_ label: String, symbol: String? = nil, danger: Bool = false, tap: @escaping () -> Void) {
        self.label = label
        self.symbol = symbol
        self.danger = danger
        self.tap = tap
    }

    public var body: some View {
        Button(action: tap) {
            HStack(spacing: Token.Gap.small) {
                if let symbol { Image(systemName: symbol).font(.system(size: Token.Size.sub, weight: .medium)) }
                Text(label).font(.system(size: Token.Size.text)).lineLimit(1).fixedSize()
            }
            .foregroundStyle(danger ? palette.bad : palette.accent)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

// value: a symbol that is a button: a row's (copy, QR, open) on a raised
// 26 pt square 8 pt round; the header's settings on a 28 pt circle.
public struct PlainIcon: View {
    let symbol: String
    let picked: Bool
    let round: Bool
    let tap: () -> Void
    @Environment(\.stationPalette) private var palette
    @Environment(\.colorScheme) private var scheme

    public init(_ symbol: String, picked: Bool = false, round: Bool = false, tap: @escaping () -> Void) {
        self.symbol = symbol
        self.picked = picked
        self.round = round
        self.tap = tap
    }

    public var body: some View {
        let side = round ? Token.Tile.roundButton : Token.Tile.squareButton
        Button(action: tap) {
            Image(systemName: symbol).font(.system(size: round ? Token.Detail.roundGlyph : Token.Detail.squareGlyph, weight: .medium))
                .foregroundStyle(picked ? palette.accent : palette.glyph)
                .frame(width: side, height: side)
                .background {
                    if round {
                        Circle().fill(Token.Fill.tile(scheme))
                    } else {
                        RoundedRectangle(cornerRadius: Token.Tile.squareCorner, style: .continuous).fill(Token.Fill.raised(scheme))
                    }
                }
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

// value: Copy as a row's symbol: it says Copied with a check for a moment
// once the words are on the pasteboard.
public struct CopyIcon: View {
    let tap: () -> Void
    @State private var copied = false
    @Environment(\.stationPalette) private var palette

    public init(tap: @escaping () -> Void) { self.tap = tap }

    public var body: some View {
        PlainIcon(copied ? Token.Symbol.check : Token.Symbol.copy, picked: copied) {
            tap()
            copied = true
            Task { @MainActor in
                try? await Task.sleep(nanoseconds: 1_500_000_000)
                copied = false
            }
        }
        .accessibilityLabel(copied ? "Copied" : "Copy")
        .help(copied ? "Copied" : "Copy")
    }
}

// value: the way back of a pushed page: the chevron and the page it goes
// back to, in the accent.
public struct BackButton: View {
    let title: String
    let action: () -> Void
    @Environment(\.stationPalette) private var palette

    public init(_ title: String, action: @escaping () -> Void) {
        self.title = title
        self.action = action
    }

    public var body: some View {
        Button(action: action) {
            HStack(spacing: Token.Gap.tight) {
                Image(systemName: Token.Symbol.back).font(.system(size: Token.Detail.back, weight: .bold)).foregroundStyle(palette.glyph)
                Text(title).font(.system(size: Token.Size.text, weight: .medium)).padding(.vertical, Token.Size.leading)
            }
            .foregroundStyle(palette.accent)
            .padding(Token.Gap.tight)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

// value: a pushed page's head as the vision draws it: the way back, then
// its icon, its 19 pt title and its tag, its words, what it ends with (a
// state).
public struct PageHeader<Trailing: View>: View {
    let back: String
    let title: String
    let subtitle: String?
    let symbol: String?
    let active: Bool
    let badge: String?
    let action: () -> Void
    let trailing: Trailing

    public init(back: String, title: String, subtitle: String? = nil, symbol: String? = nil, active: Bool = false, badge: String? = nil, action: @escaping () -> Void = {}, @ViewBuilder trailing: () -> Trailing) {
        self.back = back
        self.title = title
        self.subtitle = subtitle
        self.symbol = symbol
        self.active = active
        self.badge = badge
        self.action = action
        self.trailing = trailing()
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Tile.gap) {
            BackButton(back, action: action)
            HStack(alignment: .center, spacing: Token.Gap.pair) {
                if let symbol { IconTile(symbol, active: active, size: Token.Tile.headIcon) }
                VStack(alignment: .leading, spacing: 0) {
                    HStack(alignment: .center, spacing: Token.Gap.small) {
                        Text(title).font(.system(size: Token.Size.title, weight: .bold)).tracking(Token.Detail.titleTracking * Token.Size.title).foregroundStyle(Token.Ink.primary)
                            .padding(.vertical, Token.Size.titleLeading)
                        if let badge { KindTag(badge) }
                    }
                    if let subtitle, !subtitle.isEmpty {
                        Wrap(subtitle, spacing: Token.Size.headLeading * 2).font(.system(size: Token.Size.sub)).foregroundStyle(Token.Ink.secondary)
                            .padding(.vertical, Token.Size.headLeading).fixedSize(horizontal: false, vertical: true)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .layoutPriority(1)
                trailing
            }
            .padding(EdgeInsets(top: Token.Gap.tight, leading: Token.Gap.tight, bottom: 0, trailing: Token.Gap.tight))
        }
    }
}

extension PageHeader where Trailing == EmptyView {
    public init(back: String, title: String, subtitle: String? = nil, symbol: String? = nil, active: Bool = false, badge: String? = nil, action: @escaping () -> Void = {}) {
        self.init(back: back, title: title, subtitle: subtitle, symbol: symbol, active: active, badge: badge, action: action) { EmptyView() }
    }
}

// value: a machine's state as the vision's chip: "Running" green on its
// green, a state that is not running secondary on the hairline's grey.
public struct StateWord: View {
    let word: String
    let mark: Mark
    @Environment(\.stationPalette) private var palette
    @Environment(\.colorScheme) private var scheme

    public init(_ word: String, mark: Mark) {
        self.word = word
        self.mark = mark
    }

    public var body: some View {
        let on = mark != .absent && mark != .pending
        let hue = palette.of(mark)
        Text(word).font(.system(size: Token.Size.label, weight: .semibold)).lineLimit(1).fixedSize()
            .foregroundStyle(on ? AnyShapeStyle(hue) : AnyShapeStyle(Token.Ink.secondary))
            .padding(.horizontal, Token.Gap.row).padding(.vertical, Token.Gap.inner)
            .background(on ? AnyShapeStyle(hue.opacity(Token.Fill.chip)) : AnyShapeStyle(Token.Fill.faint(scheme)), in: Capsule())
    }
}

// value: a VM's kind beside its name: Server in teal on its teal, any
// other kind secondary on the hairline's grey.
public struct KindTag: View {
    let kind: String
    @Environment(\.colorScheme) private var scheme

    public init(_ kind: String) { self.kind = kind }

    public var body: some View {
        let server = kind == "Server"
        Text(kind).font(.system(size: Token.Size.tag, weight: .semibold)).lineLimit(1).fixedSize()
            .foregroundStyle(server ? AnyShapeStyle(Token.Hue.teal(scheme)) : AnyShapeStyle(Token.Ink.secondary))
            .padding(.horizontal, Token.Gap.item).padding(.vertical, Token.Gap.inner)
            .background(server ? AnyShapeStyle(Token.Hue.teal(scheme).opacity(Token.Fill.tag)) : AnyShapeStyle(Token.Fill.faint(scheme)), in: Capsule())
    }
}

// value: the hairline above the panel's menu, never between rows.
public struct SectionDivider: View {
    public init() {}

    public var body: some View { RowSeparator() }
}

// value: a short note under a section: 11.5 pt, secondary.
public struct FootNote: View {
    let text: String

    public init(_ text: String) { self.text = text }

    public var body: some View {
        SubText(text).padding(.horizontal, Token.Gap.small)
    }
}

// value: the machine switcher as the vision's capsule: This Mac's icon,
// its name in 13.5 pt semibold, a chevron; open, raised and ringed in the
// accent.
struct SwitcherPill: View {
    let open: Bool
    @Environment(\.colorScheme) private var scheme
    @Environment(\.stationPalette) private var palette

    var body: some View {
        HStack(spacing: Token.Gap.item) {
            IconTile(Token.Symbol.mac, active: true, size: Token.Detail.pillIcon)
            Text(Page.ThisMacWord).font(.system(size: Token.Size.pill, weight: .semibold)).foregroundStyle(Token.Ink.primary)
            Image(systemName: Token.Symbol.down).font(.system(size: Token.Detail.caret, weight: .bold)).foregroundStyle(palette.glyph)
        }
        .padding(EdgeInsets(top: Token.Gap.flank, leading: Token.Gap.flank, bottom: Token.Gap.flank, trailing: Token.Gap.block))
        .background {
            Capsule().fill(open ? Token.Fill.raised(scheme) : Token.Fill.tile(scheme))
            Capsule().strokeBorder(Token.Fill.edge(scheme), lineWidth: Token.Stroke.hair)
            if open { Capsule().stroke(palette.accent.opacity(Token.Fill.ring), lineWidth: Token.Stroke.open).padding(-Token.Stroke.open / 2) }
        }
        .contentShape(Capsule())
    }
}

// invariant: the panel never grows past the screen: live, it scrolls once
// its content is taller than the screen's visible frame, and is exactly as
// tall as its content below that, its words laid out at the panel's width;
// drawn still, it is its content.
struct PanelScroll: ViewModifier {
    let scrolls: Bool
    // value: what the panel shows; another starts at the top.
    let key: String
    // value: the sheet open, if any: opening one scrolls it into sight.
    let sheet: String?
    // value: the height kept under the scroll for the status line; the
    // rows it takes the room of are scrolled up into sight, so the line
    // never hides the control it is about.
    var reserve: CGFloat = 0
    static let sheetID = "station.sheet"
    @Environment(\.stationStill) private var still
    @Environment(\.stationPanelWidth) private var width
    @Environment(\.stationTokens) private var tokens
    @State private var height: CGFloat = 0
    @State private var offset: CGFloat = 0
    @State private var position = ScrollPosition(edge: .top)
    // value: a sheet just opened whose room the content has not grown by
    // yet: it is scrolled into sight again once it has.
    @State private var opening = false
    @State private var scroller = Self.scrollerWidth

    // value: the room the panel may take: the screen's visible height
    // under the menu bar, less the gap macOS leaves above the Dock.
    static var room: CGFloat { max((NSScreen.main?.visibleFrame.height ?? Token.Metric.Panel.unknownScreen) - Token.Metric.Panel.dockGap, Token.Metric.Panel.leastRoom) }

    // value: the content is taller than the panel's room: it scrolls.
    var scrolled: Bool { height > Self.room - reserve }

    // value: the width a legacy scroller takes beside the content (a
    // mouse plugged in), kept on every page so the panel's width and its
    // left margin never change from one page to the next, nor once the
    // page is read; an overlay scroller takes none.
    static var scrollerWidth: CGFloat {
        NSScroller.preferredScrollerStyle == .legacy ? NSScroller.scrollerWidth(for: .regular, scrollerStyle: .legacy) : 0
    }

    func body(content: Content) -> some View {
        if still || !scrolls {
            content
        } else {
            ScrollViewReader { proxy in
                ScrollView(.vertical) {
                    content.onGeometryChange(for: CGFloat.self) { $0.size.height } action: { height = $0 }
                        .padding(.trailing, scrolled ? 0 : scroller)
                }
                .onChange(of: sheet) { _, open in
                    opening = open != nil
                    guard open != nil else { return }
                    DispatchQueue.main.async { withAnimation(tokens.motion(Token.Motion.smooth)) { proxy.scrollTo(Self.sheetID, anchor: .bottom) } }
                }
                .onChange(of: height) {
                    guard opening else { return }
                    opening = false
                    DispatchQueue.main.async { withAnimation(tokens.motion(Token.Motion.smooth)) { proxy.scrollTo(Self.sheetID, anchor: .bottom) } }
                }
                .scrollPosition($position)
                .onScrollGeometryChange(for: CGFloat.self) { $0.contentOffset.y } action: { _, y in offset = y }
                .onChange(of: reserve) { was, now in
                    guard now > was, height > Self.room - now else { return }
                    position.scrollTo(y: offset + now - was)
                }
            }
            .id(key)
            .onReceive(NotificationCenter.default.publisher(for: NSScroller.preferredScrollerStyleDidChangeNotification)) { _ in
                scroller = Self.scrollerWidth
            }
            .onAppear { scroller = Self.scrollerWidth }
            .onChange(of: key) {
                offset = 0
                position.scrollTo(edge: .top)
            }
            .scrollBounceBehavior(.basedOnSize)
            .scrollIndicators(scrolled ? .automatic : .never)
            .frame(width: width + scroller, height: height > 0 ? min(height, Self.room - reserve) : nil)
        }
    }
}

// invariant: the panel's content takes the width the scroll leaves it, so
// its margins never shift under a scroller; drawn still, the panel's width.
struct PanelWidth: ViewModifier {
    let scrolls: Bool
    @Environment(\.stationStill) private var still
    @Environment(\.stationPanelWidth) private var width

    func body(content: Content) -> some View {
        if still || !scrolls {
            content.frame(width: width, alignment: .topLeading)
        } else {
            content.frame(maxWidth: .infinity, alignment: .topLeading)
        }
    }
}

extension View {
    // value: the panel's width (see PanelWidth).
    public func stationWidth(_ scrolls: Bool = true) -> some View {
        modifier(PanelWidth(scrolls: scrolls))
    }

    // value: the panel's scroll (see PanelScroll).
    public func stationScroll(_ scrolls: Bool, key: String = "", sheet: String? = nil, status: Bool = false) -> some View {
        modifier(PanelScroll(scrolls: scrolls, key: key, sheet: sheet, reserve: status ? Token.Space.statusRoom : 0))
    }
}
