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
                Image(nsImage: model.iconImage(dark: NSApp.effectiveAppearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua))
                    .accessibilityLabel(model.icon.label)
                    .task { model.start() }
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
            openSwitcher: { model.openSwitcher() },
            search: { text in model.state.search = text },
            copy: { text in
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(text, forType: .string)
            },
            close: { model.closePanel() }))
        h.push = { page in model.push(page) }
        return h.deferred { DispatchQueue.main.async(execute: $0) }
    }
}
