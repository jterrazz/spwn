import Foundation

// invariant: the mark turns at Token.Busy.fps while the icon says it turns,
// a call running or a server restarting, and its timer is gone otherwise.
@MainActor
public final class Turner {
    public private(set) var turn = 0.0
    private let turned: (Double) -> Void
    private var timer: Timer?
    private var frame = 0

    public init(_ turned: @escaping (Double) -> Void) {
        self.turned = turned
    }

    public var running: Bool { timer != nil }

    public func follow(_ icon: MenuBarIcon) {
        guard icon.turning else {
            timer?.invalidate()
            timer = nil
            frame = 0
            set(0)
            return
        }
        guard timer == nil else { return }
        timer = Timer.scheduledTimer(withTimeInterval: 1 / Token.Busy.fps, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated {
                guard let self else { return }
                self.frame += 1
                self.set(Token.Busy.turn(frame: self.frame))
            }
        }
    }

    private func set(_ value: Double) {
        guard value != turn else { return }
        turn = value
        turned(value)
    }
}
