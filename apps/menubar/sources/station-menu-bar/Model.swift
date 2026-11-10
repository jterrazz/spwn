import AppKit
import Foundation
import StationKit
import UserNotifications

@MainActor
final class Model: ObservableObject {
    @Published var state = PanelState(page: nil, connection: .connecting) {
        didSet { turner.follow(icon) }
    }
    @Published private(set) var turn = 0.0
    @Published var loginItemOn = LoginItem.on
    @Published var sparkline = UserDefaults.standard.bool(forKey: Model.sparklineKey)

    // value: the menu bar's own display option, in its macOS user defaults,
    // never in station's preferences.
    static let sparklineKey = "cpuSparkline"
    // value: the machines last opened, Find's Recent (SPEC B5.21): the menu
    // bar's own, in its macOS user defaults.
    static let recentKey = "recentMachines"
    static let recentKept = 3
    // value: Activity's window last picked, the menu bar's own.
    nonisolated static let windowKey = "activityWindow"
    // value: the window last picked on each machine's Activity.
    static let windowsKey = "activityWindows"
    // value: how long a control stays pending once its call returned, at
    // most, for the page that says its outcome.
    static let settle: UInt64 = 2_500_000_000

    let socket = EnginePath.socket()
    private var lastPage: Date?
    private var subscription: Task<Void, Never>?
    private var events: Task<Void, Never>?
    private var clearing: Task<Void, Never>?
    private var watching: Task<Void, Never>?
    private var reviving: Task<Void, Never>?
    private var finding: Task<Void, Never>?
    private var asked = false
    private var outdated = false
    private(set) var navigation: Navigation!
    let live = LiveStore()
    let hud = HUDWindow()
    private var open = false

    private var settled: Task<Void, Never>?
    private lazy var turner = Turner { [weak self] in self?.turn = $0 }

    init() {
        navigation = Navigation(socket: socket, window: UserDefaults.standard.string(forKey: Self.windowKey)) { [weak self] page, shown in
            Task { @MainActor in self?.show(page, shown) }
        }
        navigation.remember = { flag in UserDefaults.standard.set(flag, forKey: Model.windowKey) }
        state.windows = UserDefaults.standard.dictionary(forKey: Self.windowsKey) as? [String: String] ?? [:]
        navigation.sampled = { [weak self] sample in Task { @MainActor in self?.live.take(sample) } }
    }

    // invariant: a pushed page is shown only while its watch is the one held;
    // the page read after a call ends its pending.
    func show(_ watch: String, _ page: Page?) {
        guard navigation.watched == (page == nil ? nil : watch) else { return }
        let changed = state.pushed?.id != page?.id
        if let page { state.held = AwakePage.settled(state.held, page) } else { state.held = [:] }
        state.pushed = page
        state.showing(page == nil ? nil : watch)
        state.back = navigation.backWords
        state.settling = nil
        if page == nil || changed {
            state.confirming = nil
            state.qr = nil
            state.about = nil
            DispatchQueue.main.async { PanelWindow.key(NSApp.windows) }
        }
    }

    var icon: MenuBarIcon {
        var icon = MenuBarIcon.of(connection: state.connection, page: state.page, sparkline: sparkline, live: live)
        icon.busy = state.running != nil
        return icon
    }

    var iconImage: NSImage { icon.nsImage(turn: turn) }

    func start() {
        guard subscription == nil else { return }
        let sub = Subscription(path: socket) { event in
            Task { @MainActor in self.receive(event) }
        }
        subscription = Task.detached { await sub.run() }
        let told = Subscription(path: socket, watch: Notifications.watch) { event in
            guard let text = Notifications.posted(event, from: Notifications.watch) else { return }
            Task { @MainActor in self.notify(text) }
        }
        events = Task.detached { await told.run() }
    }

    // invariant: a pushed page is watched only while it is shown: opening
    // another or going back ends its watch (SPEC §10.3), and a page of the
    // watch ended never comes back.
    func push(_ page: String) {
        state.confirming = nil
        state.showing(page)
        navigation.open(page)
    }

    func openMachine(_ name: String) {
        state.recent = Array(([name] + state.recent.filter { $0 != name }).prefix(Self.recentKept))
        UserDefaults.standard.set(state.recent, forKey: Self.recentKey)
        push(PageID.of(name))
    }

    func remember(_ machine: String, window: String) {
        state.windows[machine] = window
        UserDefaults.standard.set(state.windows, forKey: Self.windowsKey)
    }

    // invariant: SPEC A6: Find opens in the panel's place over the machines
    // opened last, its index (`menubar.find`) watched only while it is open;
    // nil closes it and ends that watch. The index last read is kept, so
    // Find opens with its list at once and the watch brings it up to date.
    func find(_ query: FindQuery?) {
        guard let query else {
            finding?.cancel()
            finding = nil
            state.find = nil
            return
        }
        if state.find == nil || finding == nil {
            state.recent = UserDefaults.standard.stringArray(forKey: Self.recentKey) ?? []
            state.confirming = nil
            watchFind()
        }
        state.find = query
    }

    func watchFind() {
        finding?.cancel()
        let sub = Subscription(path: socket, pages: [PageID.find]) { [weak self] event in
            guard case .page(let page) = event, page.id == PageID.find else { return }
            Task { @MainActor in
                guard let self, self.state.find != nil else { return }
                self.state.findIndex = page
            }
        }
        finding = Task.detached { await sub.run() }
    }

    func setSparkline(_ on: Bool) {
        UserDefaults.standard.set(on, forKey: Self.sparklineKey)
        sparkline = on
    }

    func receive(_ event: StreamEvent) {
        switch event {
        case .connected:
            state.connection = .live
            askEngine()
        case .page(let page):
            guard !outdated else { return }
            state.connection = .live
            state.page = page
            lastPage = Date()
            if navigation.watched == nil { state.settling = nil }
        case .tick:
            break
        case .sample(let sample):
            live.take(sample)
            if sparkline { objectWillChange.send() }
        case .notice(_, let icon, let text):
            state.say(Status(text, failed: false), on: navigation.watched)
            clearLater()
            if !open, let notice = HUDNotice(notice: text, icon: icon) { hud.show(notice) }
        case .down:
            if FileManager.default.fileExists(atPath: socket) && state.page != nil && state.connection == .live {
                state.connection = .connecting
            } else {
                showStopped()
            }
        }
    }

    func showStopped() {
        if watching != nil { revive() }
        let installed = FileManager.default.fileExists(atPath: EnginePath.agentPlist())
        state.connection = installed ? .stopped : .notInstalled
        guard let url = Bundle.main.url(forResource: "stopped", withExtension: "json"),
              let data = try? Data(contentsOf: url), var page = try? Page.decode(data) else { return }
        page.context = Context(machine: state.page?.context?.machine ?? Format.bareName(ProcessInfo.processInfo.hostName), kind: page.context?.kind)
        if !installed {
            page = notInstalled(page)
        }
        state.page = page
    }

    func askEngine() {
        let socket = self.socket
        Task {
            guard let hello = await Client.hello(path: socket) else { return }
            outdated = hello.older
            if outdated { showOutdated() }
        }
    }

    func showOutdated() {
        state.connection = .stopped
        guard let url = Bundle.main.url(forResource: "outdated", withExtension: "json"),
              let data = try? Data(contentsOf: url), var page = try? Page.decode(data) else { return }
        page.context = Context(machine: state.page?.context?.machine ?? Format.bareName(ProcessInfo.processInfo.hostName), kind: page.context?.kind)
        state.page = page
    }

    func notInstalled(_ page: Page) -> Page {
        var p = page
        p.actions = p.actions?.filter { $0.id == "open-station" }
        if var s = p.sections?.first, var row = s.items?.first?.row {
            row.cells = [Value(kind: "text", text: "not installed")]
            row.hint = Hint(command: "station engine start")
            s.items = [Item(row: row)]
            s.caption = [Value(kind: "text", text: "The engine is not installed on this Mac")]
            p.sections = [s]
        }
        return p
    }

    // invariant: the panel opened is the key window, so its first click acts
    // and its keys are its own; the app it hands the owner to takes the
    // focus back, and the panel closes then.
    func opened() {
        open = true
        NSApp.activate()
        for delay in PanelWindow.tries {
            DispatchQueue.main.asyncAfter(deadline: .now() + delay) { PanelWindow.key(NSApp.windows) }
        }
        state.openings += 1
        let socket = self.socket
        watching?.cancel()
        watching = Task.detached { await Client.hold(path: socket, watch: "engine.watch") }
        revive()
        if let last = lastPage, Date().timeIntervalSince(last) > 60 {
            Task { _ = await Client.call(path: socket, name: ActionRunner.refresh, input: ["pages": [PageID.home]]) }
        }
    }

    // invariant: while the panel is open over an engine that does not
    // answer, the socket is tried every 2 s: an engine back is streamed at
    // once, never after the subscription's backoff of up to 30 s.
    func revive() {
        reviving?.cancel()
        let socket = self.socket
        reviving = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 2_000_000_000)
                guard let self, !Task.isCancelled, self.state.connection != .live else { return }
                if (try? UnixSocket.connect(socket)) != nil {
                    self.subscription?.cancel()
                    self.subscription = nil
                    self.events?.cancel()
                    self.events = nil
                    self.start()
                    return
                }
            }
        }
    }

    // invariant: the panel hidden is back at its top, so its next opening
    // draws its first frame there.
    func closed() {
        open = false
        state.openings += 1
        find(nil)
        reviving?.cancel()
        reviving = nil
        watching?.cancel()
        watching = nil
        navigation.home()
    }

    // value: the panel's window, closed as a menu closes once it handed the
    // owner to another app.
    func closePanel() {
        PanelWindow.close(NSApp.windows)
    }

    func open(_ url: URL) {
        NSWorkspace.shared.open(url)
        closePanel()
    }

    func run(_ action: Action, answers: [String: String]) {
        if state.find != nil { find(nil) }
        state.running = action.id
        state.settling = nil
        state.status = nil
        state.sent = answers
        state.stopping = action.id == StopAll.id ? StopAll.vms(state.page) : []
        let on = navigation.watched
        let socket = self.socket
        let reread = [PageID.shown(on)]
        if let id = SignInWatch.start(action, pending: &state.pending) { wait(SignInWait(id: id, watch: reread[0], on: state.shown)) }
        if action.run?.terminal == true { closePanel() }
        Task {
            let executable = await Client.executable(path: socket) ?? Self.fallbackExecutable
            let status = await ActionRunner(executable: executable, socket: socket, reread: reread).run(action, answers: answers)
            state.running = nil
            state.say(status, on: on)
            if status == nil, let notice = HUDNotice(action) { hud.show(notice) }
            state.showing(navigation.watched)
            if status?.failed != true, action.run?.call == ActionRunner.awake, state.pushed?.id == MenuPage.awake {
                let sent = (action.run?.in ?? [:]).merging(answers) { $1 }
                for key in ["Lid", "Battery"] { if let v = sent[key], v == "true" || v == "false" { state.held[key] = v } }
                state.held.merge(AwakePage.holding(action, answers: answers)) { $1 }
            }
            if action.mutates == true, status?.failed != true, action.run?.call != nil {
                state.settling = action.id
                settled?.cancel()
                settled = Task {
                    try? await Task.sleep(nanoseconds: Self.settle)
                    if !Task.isCancelled, state.settling == action.id { state.settling = nil }
                }
            }
            if let text = status?.posted { notify(text) }
            clearing?.cancel()
            if status != nil { clearLater() }
        }
    }

    func wait(_ wait: SignInWait) {
        let socket = self.socket
        let reread = [wait.watch]
        let following = wait.follow(socket: socket)
        Task { @MainActor in
            await wait.run(since: Date(), now: { Date() }, sleep: { try? await Task.sleep(nanoseconds: UInt64($0 * 1_000_000_000)) },
                           refresh: { _ = await Client.call(path: socket, name: ActionRunner.refresh, input: ["pages": reread]) })
            following.cancel()
            state.pending.removeAll { $0 == wait.id }
        }
    }

    // invariant: a status line, a call's outcome or a notice the engine
    // told, goes after 30 s.
    func clearLater() {
        clearing?.cancel()
        clearing = Task {
            try? await Task.sleep(nanoseconds: 30_000_000_000)
            if !Task.isCancelled { state.status = nil }
        }
    }

    static var fallbackExecutable: String { NSHomeDirectory() + "/.local/bin/station" }

    func setLoginItem(_ on: Bool) {
        do {
            try LoginItem.set(on)
        } catch {
            state.say(Status(error.localizedDescription, failed: true), on: navigation.watched)
        }
        loginItemOn = LoginItem.on
    }

    func notify(_ text: String) {
        let center = UNUserNotificationCenter.current()
        let post = {
            let content = UNMutableNotificationContent()
            content.title = "station"
            content.body = text
            center.add(UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: nil))
        }
        if asked {
            post()
            return
        }
        asked = true
        center.requestAuthorization(options: [.alert]) { granted, _ in
            if granted { post() }
        }
    }
}
