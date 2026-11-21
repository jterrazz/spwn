import AppKit
import Combine
import Observation
import StationKit
import SwiftUI

@main
struct StationMenuBarApp: App {
    @NSApplicationDelegateAdaptor(StationDelegate.self) private var station

    init() {
        let args = CommandLine.arguments
        if args.contains("--login-item-status") {
            print(LoginItem.status)
            exit(0)
        }
        if args.contains("--unregister-login-item") {
            do {
                try LoginItem.unregister()
                exit(0)
            } catch {
                FileHandle.standardError.write(Data("Error: \(error.localizedDescription)\n".utf8))
                exit(1)
            }
        }
    }

    // value: an App has a scene; the panel is the status item's, not one.
    var body: some Scene { Settings { EmptyView() } }
}

@MainActor
final class StationDelegate: NSObject, NSApplicationDelegate {
    let model = Model()
    let trial = Trial()
    private var status: StatusPanel?
    private var labelling: AnyCancellable?
    private var drawing = StatusDrawing()

    func applicationDidFinishLaunching(_: Notification) {
        let trial = self.trial
        let report = Trial.frames ? { line in trial.log(line) } : nil as ((String) -> Void)?
        if Trial.on {
            status = StatusPanel(
                content: { AnyView(TrialPanel(cycle: Trial.cycle) { line in trial.log(line) }.stationSystemTheme().stationSystemAccessibility()) },
                opened: { trial.opened() },
                closed: { trial.closed() },
                report: report)
            labelling = trial.$label.sink { [weak self] image in self?.status?.item.button?.image = image }
            trial.start()
            return
        }
        let model = self.model
        status = StatusPanel(content: { AnyView(ModelPanel(model: model)) }, opened: { model.opened() }, closed: { model.closed() }, report: report,
                             generation: report == nil ? nil : { model.state.generation })
        label()
        model.start()
        if Bench.runs > 0, let status, let report {
            let bench = Bench(panel: status, model: model, log: report)
            Task { await bench.run() }
        }
    }

    // invariant: the status item is drawn again only when its icon or its
    // turn changed, whatever else of the model moved; it follows the slices
    // the icon reads and none other.
    func label() {
        let (icon, turn) = withObservationTracking { (model.icon, model.turn) } onChange: {
            DispatchQueue.main.async { [weak self] in self?.label() }
        }
        guard drawing.take(icon, turn: turn) else { return }
        status?.item.button?.image = icon.nsImage(turn: turn)
        status?.item.button?.setAccessibilityLabel(icon.label)
    }
}

struct ModelPanel: View {
    let model: Model

    var body: some View {
        StorePanel(store: model.store, handlers: handlers)
            .environment(\.stationLive, model.live)
            .stationSystemTheme()
            .stationSystemAccessibility()
    }

    var handlers: PopoverHandlers {
        var h = model.navigation.wire(PopoverHandlers(
            run: { action, answers in model.run(action, answers: answers) },
            confirm: { action in model.state.confirming = action?.id },
            answer: { id, key, value in model.state.answers[id, default: [:]][key] = value },
            open: { url in model.open(url) },
            toggleQR: { url in model.state.qr = model.state.qr == url ? nil : url },
            toggleMenu: { model.state.menu.toggle() },
            loginItem: { on in model.setLoginItem(on) },
            loginItemOn: model.loginItemOn,
            quit: { NSApplication.shared.terminate(nil) },
            openMachine: { name in model.openMachine(name) },
            copy: { text in
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(text, forType: .string)
            },
            close: { model.closePanel() }))
        h.push = { page in model.push(page) }
        h.scrub = { index in model.state.scrub = index }
        h.find = { query in model.find(query) }
        h.remember = { machine, window in model.remember(machine, window: window) }
        h.toggleAbout = { id in model.state.about = model.state.about == id ? nil : id }
        h.hud = { notice in model.hud.show(notice) }
        return h.deferred { DispatchQueue.main.async(execute: $0) }
    }
}
