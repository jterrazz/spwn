import AppKit
import Combine
import Observation
import StationKit
import SwiftUI

// value: why the app is asked to end, numbered as the C entry points carry it.
public enum QuitReason: Int32, Sendable {
    case user = 0
    case logout = 1
    case restart = 2
    case shutdown = 3
}

// value: what the panel asks of the app whose process draws it.
public struct MenuBarShell: Sendable {
    public let openWindow: @MainActor @Sendable (String?) -> Void
    public let quit: @MainActor @Sendable (QuitReason) -> Void

    public init(openWindow: @escaping @MainActor @Sendable (String?) -> Void, quit: @escaping @MainActor @Sendable (QuitReason) -> Void) {
        self.openWindow = openWindow
        self.quit = quit
    }
}

// invariant: one panel per process, started once on the main thread; the
// item's visibility is the owner's, kept in the app's own defaults, shown
// unless he hid it.
@MainActor
public enum SpwnMenuBar {
    static var panel: MenuBarPanel?
    nonisolated static let shownKey = "showInMenuBar"

    public static func start(_ shell: MenuBarShell) {
        guard panel == nil else { return }
        let started = MenuBarPanel(shell: shell)
        panel = started
        started.start()
        started.show(shown)
    }

    public nonisolated static var shown: Bool {
        UserDefaults.standard.object(forKey: shownKey) as? Bool ?? true
    }

    public static func show(_ on: Bool) {
        UserDefaults.standard.set(on, forKey: shownKey)
        panel?.show(on)
    }
}

public typealias SpwnOpenWindow = @convention(c) (UnsafeMutableRawPointer?, UnsafePointer<CChar>?) -> Void
public typealias SpwnQuit = @convention(c) (UnsafeMutableRawPointer?, Int32) -> Void

struct ShellContext: @unchecked Sendable {
    let raw: UnsafeMutableRawPointer?
}

func onMain(_ work: @escaping @MainActor @Sendable () -> Void) {
    if Thread.isMainThread {
        MainActor.assumeIsolated(work)
    } else {
        DispatchQueue.main.async { MainActor.assumeIsolated(work) }
    }
}

@_cdecl("spwn_menu_bar_start")
public func spwn_menu_bar_start(_ ctx: UnsafeMutableRawPointer?, _ openWindow: SpwnOpenWindow, _ quit: SpwnQuit) {
    let context = ShellContext(raw: ctx)
    let shell = MenuBarShell(
        openWindow: { machine in
            guard let machine else { return openWindow(context.raw, nil) }
            machine.withCString { openWindow(context.raw, $0) }
        },
        quit: { reason in quit(context.raw, reason.rawValue) })
    onMain { SpwnMenuBar.start(shell) }
}

@_cdecl("spwn_menu_bar_set_shown")
public func spwn_menu_bar_set_shown(_ shown: Bool) {
    onMain { SpwnMenuBar.show(shown) }
}

@_cdecl("spwn_menu_bar_shown")
public func spwn_menu_bar_shown() -> Bool {
    SpwnMenuBar.shown
}

@MainActor
final class MenuBarPanel {
    let shell: MenuBarShell
    let model = Model()
    let trial = Trial()
    private var status: StatusPanel?
    private var labelling: AnyCancellable?
    private var drawing = StatusDrawing()

    init(shell: MenuBarShell) {
        self.shell = shell
    }

    func start() {
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
        let shell = self.shell
        status = StatusPanel(content: { AnyView(ModelPanel(model: model, shell: shell)) }, opened: { model.opened() }, closed: { model.closed() }, report: report,
                             generation: report == nil ? nil : { model.state.generation })
        label()
        model.start()
        if Bench.runs > 0, let status, let report {
            let bench = Bench(panel: status, model: model, log: report)
            Task { await bench.run() }
        }
    }

    func show(_ on: Bool) {
        if !on { status?.hide() }
        status?.item.isVisible = on
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
    let shell: MenuBarShell

    var body: some View {
        StorePanel(store: model.store, handlers: handlers)
            .environment(\.stationLive, model.live)
            .stationSystemTheme()
            .stationSystemAccessibility()
    }

    var handlers: PopoverHandlers {
        let shell = self.shell
        var h = model.navigation.wire(PopoverHandlers(
            run: { action, answers in model.run(action, answers: answers) },
            confirm: { action in model.state.confirming = action?.id },
            answer: { id, key, value in model.state.answers[id, default: [:]][key] = value },
            open: { url in model.open(url) },
            toggleQR: { url in model.state.qr = model.state.qr == url ? nil : url },
            toggleMenu: { model.state.menu.toggle() },
            quit: { shell.quit(.user) },
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
