import SwiftUI

private struct ContrastKey: EnvironmentKey {
    static let defaultValue = ColorSchemeContrast.standard
}

private struct ReduceTransparencyKey: EnvironmentKey {
    static let defaultValue = false
}

private struct ReduceMotionKey: EnvironmentKey {
    static let defaultValue = false
}

private struct PanelWidthKey: EnvironmentKey {
    static let defaultValue = Token.Space.width
}

private struct PanelRoomKey: EnvironmentKey {
    static let defaultValue: CGFloat? = nil
}

extension EnvironmentValues {
    public var stationContrast: ColorSchemeContrast {
        get { self[ContrastKey.self] }
        set { self[ContrastKey.self] = newValue }
    }

    public var stationReduceTransparency: Bool {
        get { self[ReduceTransparencyKey.self] }
        set { self[ReduceTransparencyKey.self] = newValue }
    }

    public var stationReduceMotion: Bool {
        get { self[ReduceMotionKey.self] }
        set { self[ReduceMotionKey.self] = newValue }
    }

    public var stationPanelWidth: CGFloat {
        get { self[PanelWidthKey.self] }
        set { self[PanelWidthKey.self] = newValue }
    }

    public var stationPanelRoom: CGFloat? {
        get { self[PanelRoomKey.self] }
        set { self[PanelRoomKey.self] = newValue }
    }
}

extension View {
    public func stationAccessibility(contrast: ColorSchemeContrast = .standard, reduceTransparency: Bool = false, reduceMotion: Bool = false) -> some View {
        environment(\.stationContrast, contrast)
            .environment(\.stationReduceTransparency, reduceTransparency)
            .environment(\.stationReduceMotion, reduceMotion)
    }

    public func stationSystemAccessibility() -> some View { modifier(SystemAccessibility()) }
}

struct SystemAccessibility: ViewModifier {
    @Environment(\.colorSchemeContrast) private var contrast
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    func body(content: Content) -> some View {
        content.stationAccessibility(contrast: contrast, reduceTransparency: reduceTransparency, reduceMotion: reduceMotion)
    }
}
