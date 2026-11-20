import CoreText
import SwiftUI

public enum ViewRole: String, Sendable {
    case button
    case checkbox
    case dialog
    case group
    case heading
    case img
    case link
    case list
    case listitem
    case menu
    case menuitem
    case progressbar
    case region
    case separator
    case slider
    case status
    case `switch`
    case tablist
    case text
    case textbox
}

// invariant: a state is what the pixels show as a look and a view golden
// says as a word: a lit row, an open card, a switch on.
public enum ViewState: String, Sendable {
    case active
    case asleep
    case busy
    case checked
    case danger
    case dimmed
    case disabled
    case expanded
    case failed
    case lit
    case low
    case on
    case picked
    case pinging
    case stale
    case waiting

    public static func of(_ pairs: KeyValuePairs<ViewState, Bool>) -> [ViewState] { pairs.filter(\.value).map(\.key) }
}

public struct ViewClip: Equatable, Sendable {
    public let fade: CGFloat
    public let token: String

    public init(fade: CGFloat, token: String) {
        self.fade = fade
        self.token = token
    }
}

public struct TokenRead: Equatable, Sendable {
    public let name: String
    public let value: String

    public init(_ name: String, value: String) {
        self.name = name
        self.value = value
    }

    public init(_ name: String, _ length: CGFloat) {
        self.init(name, value: ViewNumber.text(length))
    }

    public init(_ name: String, _ insets: EdgeInsets) {
        self.init(name, value: "[" + [insets.top, insets.leading, insets.bottom, insets.trailing].map(ViewNumber.text).joined(separator: ", ") + "]")
    }

    public init(_ name: String, _ color: Color) {
        self.init(name, value: ViewNumber.color(color))
    }

    public init(_ name: String, _ shades: [Token.Pane.Shade]) {
        self.init(name, value: "[" + shades.map { "\(ViewNumber.color($0.color)) \(ViewNumber.text($0.radius)) \(ViewNumber.text($0.y))" }.joined(separator: ", ") + "]")
    }

    public init(_ role: Token.Role) {
        var parts = ["system", ViewNumber.text(role.size), ViewNumber.weight(role.weight)]
        if role.design != .default { parts.append(ViewNumber.design(role.design)) }
        if role.tracking != 0 { parts.append("tracking " + ViewNumber.text(role.tracking)) }
        parts.append(String(describing: role.tone))
        self.init("Role." + role.name, value: parts.joined(separator: " "))
    }
}

// invariant: a mark is what a kit view says of itself in its view golden:
// its role, its words, the kit's name for a container with no role, the
// paint and the states it draws as words, and every token it read, by
// its source name; its frame is the view's bounds, read by the recorder.
public struct ViewMark: Equatable, Sendable {
    public let role: ViewRole
    public let name: String?
    public let kit: String?
    public let attrs: [String]
    public let reads: [TokenRead]
    public let clip: ViewClip?
    public let states: [ViewState]
    public let pitch: CGFloat?

    public init(_ role: ViewRole, _ name: String? = nil, kit: String? = nil, attrs: [String] = [], reads: [TokenRead] = [], clip: ViewClip? = nil,
                states: [ViewState] = [], pitch: CGFloat? = nil) {
        self.role = role
        self.name = name
        self.kit = kit
        self.attrs = attrs
        self.reads = reads
        self.clip = clip
        self.states = states
        self.pitch = pitch
    }

    public static func font(_ role: ViewRole, _ name: String? = nil, kit: String, _ font: Token.Role, states: [ViewState] = []) -> ViewMark {
        ViewMark(role, name, kit: kit, attrs: ["font=Role." + font.name], reads: [TokenRead(font)], states: states)
    }

    public static func text(_ words: String, _ role: Token.Role, heading: Bool = false) -> ViewMark {
        ViewMark(heading ? .heading : .text, words, attrs: ["font=Role." + role.name], reads: [TokenRead(role)])
    }
}

public struct RecordedView {
    public let mark: ViewMark
    public let anchor: Anchor<CGRect>
    public let children: [RecordedView]
}

public struct RecordedViews: PreferenceKey {
    public static let defaultValue: [RecordedView] = []
    public static func reduce(value: inout [RecordedView], nextValue: () -> [RecordedView]) { value += nextValue() }
}

extension Token.Role {
    // value: the height of one line of the role's words, what a text's
    // height is a whole number of.
    public var pitch: CGFloat {
        guard let font = CTFontCreateUIFontForLanguage(.system, size, nil) else { return size }
        return CTFontGetAscent(font) + CTFontGetDescent(font) + CTFontGetLeading(font)
    }
}

extension Text {
    // invariant: a text records its words where it is laid out, before
    // any frame of its view's, so its frame is the words' own and its
    // lines are its height over its role's pitch.
    public func recorded(_ words: some StringProtocol, _ role: Token.Role, kit: String? = nil, heading: Bool = false, states: [ViewState] = []) -> some View {
        font(role.font).recorded(ViewMark(heading ? .heading : .text, String(words), kit: kit, attrs: ["font=Role." + role.name], reads: [TokenRead(role)],
                                          states: states, pitch: role.pitch))
    }
}

extension View {
    // invariant: a recorded view holds what its content recorded as its
    // children, so the recorder reads a tree; nil records nothing and
    // passes the content's marks through.
    public func recorded(_ mark: ViewMark?) -> some View {
        transformAnchorPreference(key: RecordedViews.self, value: .bounds) { value, anchor in
            guard let mark else { return }
            value = [RecordedView(mark: mark, anchor: anchor, children: value)]
        }
    }

    // invariant: a copy of a content drawn only to shape a mask records
    // nothing, so a view is one line however many times it is laid out.
    public func unrecorded() -> some View {
        transformPreference(RecordedViews.self) { $0 = [] }
    }
}

public enum ViewNumber {
    // value: a length on the half-point grid, whole when it is.
    public static func text(_ value: CGFloat) -> String {
        let half = (value * 2).rounded() / 2
        if half == 0 { return "0" }
        return half == half.rounded() ? String(Int(half)) : String(format: "%.1f", Double(half))
    }

    public static func share(_ value: Double) -> String {
        let cents = (value * 100).rounded() / 100
        return cents == cents.rounded() ? String(Int(cents)) : String(format: "%g", cents)
    }

    public static func color(_ color: Color) -> String {
        let c = color.resolve(in: EnvironmentValues())
        let channels = [c.red, c.green, c.blue].map { String(Int((Double($0) * 255).rounded())) }
        return "rgba(" + channels.joined(separator: " ") + " / " + share(Double(c.opacity)) + ")"
    }

    public static func weight(_ weight: Font.Weight) -> String {
        let names: [(Font.Weight, String)] = [
            (.ultraLight, "ultraLight"), (.thin, "thin"), (.light, "light"), (.regular, "regular"), (.medium, "medium"),
            (.semibold, "semibold"), (.bold, "bold"), (.heavy, "heavy"), (.black, "black"),
        ]
        return names.first { $0.0 == weight }?.1 ?? "weight"
    }

    public static func design(_ design: Font.Design) -> String {
        switch design {
        case .serif: "serif"
        case .rounded: "rounded"
        case .monospaced: "monospaced"
        default: "default"
        }
    }
}
