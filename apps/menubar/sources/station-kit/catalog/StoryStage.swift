import AppKit
import SwiftUI

public enum StoryDesk: String, CaseIterable, Sendable {
    case flat, wallpaper
}

public enum StoryAccent: String, CaseIterable, Sendable {
    case blue, system
}

public struct StoryKnobs: Equatable, Sendable {
    public static let widths = Token.Stage.widths
    public static let scales = Token.Stage.scales

    public var scheme: ColorScheme
    public var increaseContrast: Bool
    public var reduceTransparency: Bool
    public var reduceMotion: Bool
    public var width: CGFloat
    public var scale: CGFloat
    public var desk: StoryDesk
    public var accent: StoryAccent

    public init(scheme: ColorScheme = .dark, increaseContrast: Bool = false, reduceTransparency: Bool = false, reduceMotion: Bool = false,
                width: CGFloat = Token.Space.width, scale: CGFloat = 2, desk: StoryDesk = .flat, accent: StoryAccent = .blue) {
        self.scheme = scheme
        self.increaseContrast = increaseContrast
        self.reduceTransparency = reduceTransparency
        self.reduceMotion = reduceMotion
        self.width = width
        self.scale = scale
        self.desk = desk
        self.accent = accent
    }
}

extension View {
    public func storyKnobs(_ knobs: StoryKnobs) -> some View {
        stationTheme(knobs.scheme)
            .stationAccessibility(contrast: knobs.increaseContrast ? .increased : .standard,
                                  reduceTransparency: knobs.reduceTransparency, reduceMotion: knobs.reduceMotion)
            .environment(\.stationPanelWidth, knobs.width)
            .tint(knobs.accent == .blue ? Token.Hue.blue(knobs.scheme) : nil)
    }
}

public struct StoryDeskView: View {
    let desk: StoryDesk
    let scheme: ColorScheme

    public init(_ desk: StoryDesk, scheme: ColorScheme) {
        self.desk = desk
        self.scheme = scheme
    }

    public var body: some View {
        switch desk {
        case .flat:
            Rectangle().fill(Token.Glass.wallBase(scheme)[0])
        case .wallpaper:
            ZStack {
                LinearGradient(colors: Token.Glass.wallBase(scheme), startPoint: Token.Glass.wallStart, endPoint: Token.Glass.wallEnd)
                ForEach(Array(Token.Glass.wallpaper(scheme).enumerated()), id: \.offset) { _, blob in
                    EllipticalGradient(colors: [blob.0, blob.0.opacity(0)], center: blob.1, startRadiusFraction: 0, endRadiusFraction: blob.2)
                }
            }
            .drawingGroup()
        }
    }
}

public struct StoryStage: View {
    public static let strip = Token.Stage.strip
    public static let room = Token.Stage.room
    public static let now = Calendar.current.date(from: DateComponents(year: 2026, month: 11, day: 7, hour: 10, minute: 30))!

    let story: Story
    let knobs: StoryKnobs
    let state: PanelState?
    let handlers: PopoverHandlers

    public init(_ story: Story, knobs: StoryKnobs = StoryKnobs(), state: PanelState? = nil, handlers: PopoverHandlers = PopoverHandlers()) {
        self.story = story
        self.knobs = knobs
        self.state = state
        self.handlers = handlers
    }

    public var body: some View {
        SwiftUI.Group {
            if story.ui.statusItem == true {
                StoryView(story, state: state, handlers: handlers)
                    .background(StoryDeskView(knobs.desk, scheme: knobs.scheme))
            } else {
                StoryView(story, state: state, handlers: handlers)
                    .fixedSize()
                    .padding(Self.room)
                    .background(StoryDeskView(knobs.desk, scheme: knobs.scheme))
            }
        }
        .environment(\.stationNow, Self.now)
        .storyKnobs(knobs)
    }

    @MainActor
    public static func png(_ story: Story, knobs: StoryKnobs) -> Data? {
        let renderer = ImageRenderer(content: StoryStage(story, knobs: knobs).environment(\.stationStill, true))
        renderer.scale = knobs.scale
        guard let image = renderer.cgImage else { return nil }
        return NSBitmapImageRep(cgImage: image).representation(using: .png, properties: [:])
    }
}
