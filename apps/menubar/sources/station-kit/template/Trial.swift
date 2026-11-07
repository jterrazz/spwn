import AppKit
import SwiftUI

public enum TrialFocus: Hashable, Sendable {
    case row(Int)

    var index: Int {
        switch self {
        case .row(let i): i
        }
    }
}

// invariant: lot L06's trial of SPEC §D2's first three risks, never shown
// unless the app or the Catalog is started with it: the panel's height
// rides one animated frame over its measured content, Tab and Shift-Tab
// walk a programmatic focus over rows focusable for edit only, and every
// change is reported in words the trial log reads.
public struct TrialPanel: View {
    let short: Int
    let tall: Int
    let cycle: Duration?
    let report: (String) -> Void

    @Environment(\.stationTokens) private var tokens
    @State private var tallNow = false
    @State private var shown: CGFloat = 0
    @FocusState private var focus: TrialFocus?

    public init(short: Int = 3, tall: Int = 9, cycle: Duration? = nil, report: @escaping (String) -> Void) {
        self.short = short
        self.tall = tall
        self.cycle = cycle
        self.report = report
    }

    var count: Int { tallNow ? tall : short }

    // value: Shift-Tab reaches a key press as the back-tab character, not as
    // Tab with Shift held.
    static let backTab = KeyEquivalent("\u{19}")

    public var body: some View {
        VStack(spacing: 0) {
            ForEach(0..<count, id: \.self) { i in
                row(i)
            }
        }
        .padding(Token.Space.margin)
        .fixedSize(horizontal: false, vertical: true)
        .onGeometryChange(for: CGFloat.self) { $0.size.height } action: { measured($0) }
        .frame(width: Token.Space.width, height: shown > 0 ? shown : nil, alignment: .top)
        .clipped()
        .onKeyPress(.tab, phases: .down) { press in
            report("key tab shift=\(press.modifiers.contains(.shift))")
            walk(press.modifiers.contains(.shift) ? -1 : 1)
            return .handled
        }
        .onKeyPress(Self.backTab, phases: .down) { _ in
            report("key backtab")
            walk(-1)
            return .handled
        }
        .onKeyPress(.return) {
            toggle("return")
            return .handled
        }
        .onKeyPress(.space) {
            toggle("space")
            return .handled
        }
        .onChange(of: focus) { _, now in
            report("focus now \(Self.name(now))")
        }
        .onAppear {
            report("appear keyboardNavigation=\(NSApp?.isFullKeyboardAccessEnabled ?? false)")
            walk(1)
        }
        .task(id: cycle) {
            guard let cycle else { return }
            while !Task.isCancelled {
                try? await Task.sleep(for: cycle)
                if Task.isCancelled { break }
                toggle("cycle")
            }
        }
    }

    func row(_ i: Int) -> some View {
        let lit = focus == .row(i)
        return RoundedRectangle(cornerRadius: Token.Metric.Row.radius, style: .continuous)
            .fill(lit ? tokens.color.q : tokens.color.qq)
            .overlay {
                if lit {
                    RoundedRectangle(cornerRadius: Token.Metric.Row.radius, style: .continuous)
                        .strokeBorder(tokens.color.accent, lineWidth: Token.Metric.Row.padding / 2)
                }
            }
            .frame(height: Token.Metric.Row.height)
            .padding(.vertical, Token.Metric.Row.hairline)
            .focusable(interactions: .edit)
            .focused($focus, equals: .row(i))
            .focusEffectDisabled()
    }

    func measured(_ height: CGFloat) {
        report("content height=\(Int(height))")
        if shown == 0 {
            shown = height
        } else {
            withAnimation(tokens.motion(Token.Motion.smooth)) { shown = height }
        }
    }

    func toggle(_ cause: String) {
        report("toggle by \(cause) rows=\(tallNow ? short : tall)")
        tallNow.toggle()
        if let at = focus?.index, at >= count { focus = .row(count - 1) }
    }

    func walk(_ step: Int) {
        let from = focus?.index ?? -step
        let wanted = TrialFocus.row((from + step + count) % count)
        focus = wanted
        report("walk wanted \(Self.name(wanted))")
        Task { @MainActor in
            await Task.yield()
            report("walk held \(Self.name(focus)) for \(Self.name(wanted)) keyboardNavigation=\(NSApp?.isFullKeyboardAccessEnabled ?? false)")
        }
    }

    static func name(_ focus: TrialFocus?) -> String {
        focus.map { "row\($0.index)" } ?? "none"
    }
}

// invariant: the status item's motion as the app would draw it: the
// badge's width at a progress and the Orbit turned by a share of a turn,
// one bitmap per screen scale, every frame rendered offscreen.
public struct TrialItemView: View {
    let progress: Double
    let turn: Double
    let scheme: ColorScheme

    public init(progress: Double, turn: Double, scheme: ColorScheme) {
        self.progress = progress
        self.turn = turn
        self.scheme = scheme
    }

    public var body: some View {
        HStack(spacing: Token.Icon.gap) {
            Image(Token.Icon.orbit, bundle: .module)
                .font(.system(size: Token.Icon.size, weight: Token.Icon.weight))
                .imageScale(.medium)
                .foregroundStyle(Token.Icon.ink(scheme))
                .rotationEffect(.radians(turn * .pi * 2))
            Capsule()
                .fill(Token.Icon.needs(scheme))
                .frame(width: Token.Icon.sparkHeight + (Token.Icon.sparkWidth - Token.Icon.sparkHeight) * progress, height: Token.Icon.sparkHeight)
        }
    }

    @MainActor
    public static func image(progress: Double, turn: Double, scheme: ColorScheme) -> NSImage {
        let image = NSImage()
        for scale in [1, 2, 3] as [CGFloat] {
            let renderer = ImageRenderer(content: TrialItemView(progress: progress, turn: turn, scheme: scheme))
            renderer.scale = scale
            guard let cg = renderer.cgImage else { continue }
            let rep = NSBitmapImageRep(cgImage: cg)
            rep.size = NSSize(width: CGFloat(cg.width) / scale, height: CGFloat(cg.height) / scale)
            image.addRepresentation(rep)
            image.size = rep.size
        }
        return image
    }
}
