import Foundation
import StationKit
import SwiftUI

@MainActor
final class Catalog: ObservableObject {
    nonisolated static let speeds: [Double] = [1, 0.25, 0.1]

    @Published private(set) var stories: [Story] = []
    @Published private(set) var failure: String?
    @Published private(set) var drifted: Set<String> = []
    @Published var selection: String? { didSet { select() } }
    @Published var search = ""
    @Published var knobs = StoryKnobs()
    @Published var live = false
    @Published var speed = 1.0
    @Published var progress = 0.0
    @Published private(set) var session: Live?
    let trial = CommandLine.arguments.contains("--trial")

    init() {
        reload()
    }

    func reload() {
        do {
            let order = Dictionary(Paths.order().enumerated().map { ($1.id, $0) }) { first, _ in first }
            stories = try Stories.load(root: Paths.stories).sorted { (order[$0.id] ?? .max, $0.id) < (order[$1.id] ?? .max, $1.id) }
            failure = nil
        } catch {
            stories = []
            failure = String(describing: error)
        }
        drifted = Set(stories.map(\.id).filter(Paths.drifted))
        if selection == nil || story(selection) == nil { selection = stories.first?.id } else { select() }
    }

    func story(_ id: String?) -> Story? { stories.first { $0.id == id } }

    var current: Story? { story(selection) }

    var shown: [(group: StoryGroup, stories: [Story])] {
        let query = search.trimmingCharacters(in: .whitespaces).lowercased()
        let matching = query.isEmpty ? stories : stories.filter { $0.id.lowercased().contains(query) || $0.note.lowercased().contains(query) }
        return StoryGroup.allCases.compactMap { group in
            let inGroup = matching.filter { $0.group == group }
            return inGroup.isEmpty ? nil : (group, inGroup)
        }
    }

    private func select() {
        session = current.map { Live($0.state) }
        progress = current?.freeze ?? 0
    }

    var playable: Bool { current?.play.flatMap { story($0.from) } != nil }

    // value: a transition story's replay, its time from 0 to 1 over its
    // movement's run slowed by the speed.
    nonisolated static func replay(_ story: Story, speed: Double) -> Double? {
        StoryMotion.of(story.id).map { $0.motion.duration / speed }
    }

    func play() {
        guard let target = current, let from = target.play.flatMap({ story($0.from) }) else { return }
        if let run = Self.replay(target, speed: speed) {
            live = false
            progress = 0
            DispatchQueue.main.async { withAnimation(.linear(duration: run)) { self.progress = 1 } }
            return
        }
        let session = Live(from.state)
        self.session = session
        live = true
        let speed = speed
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.4) {
            var transaction = Transaction(animation: .default.speed(speed))
            transaction.disablesAnimations = false
            withTransaction(transaction) { session.state = target.state }
        }
    }
}
