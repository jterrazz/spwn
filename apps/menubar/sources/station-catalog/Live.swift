import Foundation
import StationKit

@MainActor
final class Live: ObservableObject {
    @Published var state: PanelState
    @Published private(set) var calls: [String] = []

    init(_ state: PanelState) {
        self.state = state
    }

    func log(_ call: String) {
        calls.append(call)
        print("catalog: \(call)")
    }

    static func describe(_ action: Action, answers: [String: String]) -> String {
        let name = action.run?.call ?? action.id ?? action.label ?? "action"
        var words = ["run \(name)"]
        if let target = action.run?.target { words.append("target \(String(describing: target))") }
        let input = (action.run?.in ?? [:]).merging(answers) { _, answer in answer }
        if !input.isEmpty { words.append("input " + input.sorted { $0.key < $1.key }.map { "\($0.key)=\($0.value)" }.joined(separator: " ")) }
        if let argv = action.run?.argv { words.append("argv " + argv.joined(separator: " ")) }
        return words.joined(separator: ", ")
    }

    nonisolated static func later(_ work: @escaping () -> Void) {
        nonisolated(unsafe) let work = work
        DispatchQueue.main.async { work() }
    }

    var handlers: PopoverHandlers {
        var h = PopoverHandlers(
            run: { [weak self] action, answers in self?.log(Self.describe(action, answers: answers)) },
            confirm: { [weak self] action in self?.state.confirming = action?.id },
            answer: { [weak self] id, key, value in self?.state.answers[id, default: [:]][key] = value },
            open: { [weak self] url in self?.log("open \(url.absoluteString)") },
            toggleQR: { [weak self] url in
                guard let self else { return }
                state.qr = state.qr == url ? nil : url
            },
            toggleMenu: { [weak self] in self?.state.menu.toggle() },
            loginItem: { [weak self] on in self?.log("login item \(on ? "on" : "off")") },
            sparkline: { [weak self] on in self?.log("sparkline \(on ? "on" : "off")") },
            quit: { [weak self] in self?.log("quit") },
            openMachine: { [weak self] name in self?.log("open machine \(name)") },
            openSwitcher: { [weak self] in self?.log("open switcher") },
            search: { [weak self] text in self?.state.search = text },
            copy: { [weak self] text in self?.log("copy \(text)") },
            window: { [weak self] window in self?.log("window \(window)") },
            push: { [weak self] page in self?.log("push \(page)") },
            back: { [weak self] in
                guard let self else { return }
                if state.page != nil, state.pushed != nil { state.pushed = nil }
                log("back")
            },
            close: { [weak self] in self?.log("close") })
        h.scrub = { [weak self] index in self?.state.scrub = index }
        return h.deferred(Self.later)
    }
}
