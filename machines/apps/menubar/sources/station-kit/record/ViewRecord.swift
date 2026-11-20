import SwiftUI

public enum ViewRole: String, Sendable {
    case group
    case heading
    case text
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

    public init(_ role: ViewRole, _ name: String? = nil, kit: String? = nil, attrs: [String] = [], reads: [TokenRead] = [], clip: ViewClip? = nil) {
        self.role = role
        self.name = name
        self.kit = kit
        self.attrs = attrs
        self.reads = reads
        self.clip = clip
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
