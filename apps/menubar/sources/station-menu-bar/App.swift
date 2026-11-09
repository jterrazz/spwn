import AppKit
import StationKit
import SwiftUI

@main
struct StationMenuBarApp: App {
    @StateObject private var model = Model()
    @StateObject private var trial = Trial()

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

    var body: some Scene {
        MenuBarExtra {
            if Trial.on {
                TrialPanel(cycle: Trial.cycle) { line in trial.log(line) }
                    .stationSystemTheme()
                    .stationSystemAccessibility()
                    .onAppear { trial.opened() }
                    .onDisappear { trial.closed() }
            } else {
                Popover(model.state, handlers: handlers)
                    .environment(\.stationLive, model.live)
                    .modifier(PanelOpening())
                    .stationSystemTheme()
                    .stationSystemAccessibility()
                    .onAppear { model.opened() }
                    .onDisappear { model.closed() }
            }
        } label: {
            if Trial.on {
                Image(nsImage: trial.label)
                    .task { trial.start() }
            } else {
                Image(nsImage: model.iconImage)
                    .accessibilityLabel(model.icon.label)
                    .task {
                        model.start()
                        FindHotKey.shared.register { model.findFromAnywhere() }
                    }
            }
        }
        .menuBarExtraStyle(.window)
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
