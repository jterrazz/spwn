import AppKit
import Darwin
import StationKit
import SwiftUI

// invariant: lot L06's trial, on only when Station.app is started with
// STATION_TRIAL=1: the panel is the kit's TrialPanel in the StatusPanel, the window's height is
// sampled on every display frame while it is open, the label is redrawn at
// STATION_TRIAL_FPS (30 unless named; 0 draws it once), and the process's own
// CPU is reported every ten seconds. Every line goes to stderr.
@MainActor
final class Trial: ObservableObject {
    static let environment = ProcessInfo.processInfo.environment
    static let on = environment["STATION_TRIAL"] == "1"
    static let fps = Double(environment["STATION_TRIAL_FPS"] ?? "") ?? 30
    static let cycle = Double(environment["STATION_TRIAL_CYCLE"] ?? "").map { Duration.milliseconds(Int($0 * 1000)) }
    // value: any STATION_TRIAL_ flag (STATION_TRIAL_FRAMES=1 alone) logs
    // the item's and the panel's frames, which the rig clicks and crops by.
    static let frames = environment.keys.contains { $0.hasPrefix("STATION_TRIAL") }

    @Published private(set) var label = NSImage()

    private let started = Date()
    private var frame = 0
    private var sampler: Timer?
    private var drawer: Timer?
    private var meter: Timer?
    private var lastHeight: CGFloat = -1
    private var lastCPU = Trial.cpuSeconds()
    private var lastWall = Date()

    func log(_ line: String) {
        let ms = Int(Date().timeIntervalSince(started) * 1000)
        FileHandle.standardError.write(Data("trial \(ms) \(line)\n".utf8))
    }

    func start() {
        guard drawer == nil else { return }
        log("start fps=\(Self.fps) keyboardNavigation=\(NSApp.isFullKeyboardAccessEnabled)")
        draw()
        if Self.fps > 0 {
            drawer = Timer.scheduledTimer(withTimeInterval: 1 / Self.fps, repeats: true) { [weak self] _ in
                MainActor.assumeIsolated { self?.draw() }
            }
            RunLoop.main.add(drawer!, forMode: .common)
        }
        meter = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated { self?.measure() }
        }
    }

    func draw() {
        frame += 1
        let t = Double(frame) / max(Self.fps, 1)
        let turn = t.truncatingRemainder(dividingBy: Trial.turn) / Trial.turn
        let dark = NSApp.effectiveAppearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
        label = TrialItemView.image(turn: turn, scheme: dark ? .dark : .light)
    }

    // value: the mark's wave crosses its dots once per 1.2 s, without a
    // pause (SPEC M16, M29).
    static let turn = Token.Busy.period

    func measure() {
        let cpu = Self.cpuSeconds(), now = Date()
        let share = (cpu - lastCPU) / now.timeIntervalSince(lastWall) * 100
        log(String(format: "cpu %.1f%% over %.0fs frames=%d", share, now.timeIntervalSince(lastWall), frame))
        lastCPU = cpu
        lastWall = now
    }

    func opened() {
        log("opened")
        lastHeight = -1
        sampler?.invalidate()
        sampler = Timer.scheduledTimer(withTimeInterval: 1.0 / 240, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated { self?.sample() }
        }
        RunLoop.main.add(sampler!, forMode: .common)
        for delay in Token.Motion.keyTries {
            DispatchQueue.main.asyncAfter(deadline: .now() + delay) { PanelWindow.key(NSApp.windows) }
        }
    }

    func closed() {
        log("closed")
        sampler?.invalidate()
        sampler = nil
    }

    func sample() {
        guard let window = PanelWindow.panel(NSApp.windows) else { return }
        let glass = window.frame.insetBy(dx: StatusPanel.shade, dy: StatusPanel.shade)
        let height = glass.height
        guard height != lastHeight else { return }
        lastHeight = height
        log("window height=\(String(format: "%.1f", height)) y=\(String(format: "%.1f", glass.minY)) top=\(String(format: "%.1f", glass.maxY)) x=\(String(format: "%.1f", glass.minX)) key=\(window.isKeyWindow)")
    }

    static func cpuSeconds() -> Double {
        var usage = rusage()
        getrusage(RUSAGE_SELF, &usage)
        func seconds(_ t: timeval) -> Double { Double(t.tv_sec) + Double(t.tv_usec) / 1_000_000 }
        return seconds(usage.ru_utime) + seconds(usage.ru_stime)
    }
}

// invariant: under any STATION_TRIAL_ flag, each opening logs one line per display
// frame, `<what> t=<ms since it began> frame=<n> glass=<x y w h> gen=<word>`, for its
// first `span` seconds: `open` from the window shown, `push` from a page opened over
// it, for `pushSpan`. The capture rig reads whether the panel's outline moves between
// two frames, and the bench how many content generations (PanelState.generation) an
// opening draws before its last.
@MainActor
final class OpeningLog: NSObject {
    private var link: CADisplayLink?
    private var started = Date()
    private var frames = 0
    private var what = "open"
    private var span = OpeningLog.span
    private var report: ((String) -> Void)?
    private var generation: (() -> String)?
    private weak var panel: NSWindow?

    // value: past the height's own 340 ms morph, which the first sizes of a content can start.
    static let span = 1.0
    // value: past the slowest machine page the bench meets on a lone Mac.
    static let pushSpan = 6.0

    func begin(_ panel: NSWindow, what: String = "open", report: @escaping (String) -> Void, generation: (() -> String)?) {
        end()
        self.panel = panel
        self.report = report
        self.generation = generation
        self.what = what
        span = what == "open" ? Self.span : Self.pushSpan
        started = Date()
        frames = 0
        link = panel.displayLink(target: self, selector: #selector(tick(_:)))
        link?.add(to: .main, forMode: .common)
    }

    func end() {
        link?.invalidate()
        link = nil
    }

    @objc private func tick(_: CADisplayLink) {
        guard let panel else { return end() }
        let t = Date().timeIntervalSince(started)
        guard t < span else { return end() }
        frames += 1
        let glass = panel.frame.insetBy(dx: StatusPanel.shade, dy: StatusPanel.shade)
        report?("\(what) t=\(Int(t * 1000)) frame=\(frames) glass=\(StatusPanel.frame(glass)) gen=\(generation?() ?? "-")")
    }
}

// invariant: STATION_TRIAL_BENCH=<n> (with STATION_TRIAL_FRAMES=1) makes the app its
// own bench over the engine it reaches: once Home is drawn it opens the panel n times,
// then opens This Mac's page n times, each from a closed panel, and logs `bench open
// <i>` or `bench push <i>` before each, then `bench done`. The frames of each are
// OpeningLog's; the rig reads them back (capture/bench.py).
@MainActor
final class Bench {
    static let runs = Int(Trial.environment["STATION_TRIAL_BENCH"] ?? "") ?? 0

    // value: an opening is held past OpeningLog's spans, then the panel rests closed.
    static let held = 1.5
    static let rest = 0.6

    let panel: StatusPanel
    let model: Model
    let log: (String) -> Void

    init(panel: StatusPanel, model: Model, log: @escaping (String) -> Void) {
        self.panel = panel
        self.model = model
        self.log = log
    }

    func run() async {
        for _ in 0 ..< 120 where model.state.page?.header == nil {
            try? await Task.sleep(for: .milliseconds(500))
        }
        guard let mac = model.state.page?.context?.machine else { return log("bench no-home") }
        log("bench home machine=\(mac)")
        for i in 0 ..< Self.runs {
            log("bench open \(i)")
            panel.show()
            try? await Task.sleep(for: .seconds(Self.held))
            panel.hide()
            try? await Task.sleep(for: .seconds(Self.rest))
        }
        for i in 0 ..< Self.runs {
            panel.show()
            try? await Task.sleep(for: .seconds(Self.rest))
            log("bench push \(i)")
            PanelGesture.mark()
            model.openMachine(mac)
            panel.pushed()
            try? await Task.sleep(for: .seconds(OpeningLog.pushSpan + 0.5))
            panel.hide()
            try? await Task.sleep(for: .seconds(Self.rest))
        }
        log("bench done")
    }
}
