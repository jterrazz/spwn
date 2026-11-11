import AppKit
import Combine
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
        status = StatusPanel(content: { AnyView(ModelPanel(model: model)) }, opened: { model.opened() }, closed: { model.closed() }, report: report)
        label()
        labelling = model.objectWillChange.sink { [weak self] _ in DispatchQueue.main.async { self?.label() } }
        model.start()
        FindHotKey.shared.register { model.findFromAnywhere() }
    }

    func label() {
        status?.item.button?.image = model.iconImage
        status?.item.button?.setAccessibilityLabel(model.icon.label)
    }
}

struct ModelPanel: View {
    @ObservedObject var model: Model

    var body: some View {
        Popover(model.state, handlers: handlers)
            .environment(\.stationLive, model.live)
            .modifier(PanelOpening())
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
            sparkline: { on in model.setSparkline(on) },
            sparklineOn: model.sparkline,
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
