import SwiftUI

public struct HUDNotice: Equatable, Sendable {
    public let words: Value
    public let symbol: String

    public init(words: Value, symbol: String) {
        self.words = words
        self.symbol = symbol
    }

    public init?(_ action: Action) {
        guard let hud = action.hud else { return nil }
        self.init(words: hud, symbol: action.icon ?? Self.symbol(of: action.run))
    }

    public init?(notice text: String, icon: String) {
        guard !text.isEmpty else { return nil }
        self.init(words: Value(kind: "text", text: text), symbol: icon.isEmpty ? Token.Symbol.hudDone : icon)
    }

    public var text: String { Format.show(words) }

    public static func symbol(of run: Run?) -> String {
        if run?.copy != nil { return Token.Symbol.hudCopy }
        if run?.open != nil { return Token.Symbol.hudOpen }
        if run?.terminal == true { return Token.Symbol.hudTerminal }
        return Token.Symbol.hudDone
    }

    public static func all(in pages: [Page]) -> [HUDNotice] {
        var seen = Set<String>()
        var out: [HUDNotice] = []
        for page in pages {
            for action in page.actions ?? [] {
                guard let notice = HUDNotice(action), seen.insert(notice.text).inserted else { continue }
                out.append(notice)
            }
        }
        return out.sorted { $0.text < $1.text }
    }
}

public struct HUDFlight: View {
    let notice: HUDNotice

    public init(_ notice: HUDNotice) { self.notice = notice }

    public var body: some View {
        HUDCapsule(notice)
            .fixedSize()
            .keyframeAnimator(initialValue: 0.0) { capsule, progress in
                capsule.modifier(HUDReveal(progress: progress))
            } keyframes: { _ in
                LinearKeyframe(1.0, duration: Token.Motion.freshDuration)
            }
    }
}
