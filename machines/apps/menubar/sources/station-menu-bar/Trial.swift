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
        let leg = t.truncatingRemainder(dividingBy: 2 * Trial.leg) / Trial.leg
        let progress = Token.Motion.smoothCurve.value(at: leg <= 1 ? leg : 2 - leg)
        let turn = t.truncatingRemainder(dividingBy: Trial.turn) / Trial.turn
        let dark = NSApp.effectiveAppearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
        label = TrialItemView.image(progress: progress, turn: turn, scheme: dark ? .dark : .light)
    }

    // value: the badge grows and shrinks on the smooth curve's 340 ms, and
    // the busy mark turns once per 800 ms, without a pause (SPEC M16, M29).
    static let leg = 0.34
    static let turn = 0.8

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
// frame, `open t=<ms since the window was shown> frame=<n> glass=<x y w h>`, for its
// first `span` seconds, so the capture rig reads whether the panel's outline moves
// between two frames.
@MainActor
final class OpeningLog: NSObject {
    private var link: CADisplayLink?
    private var started = Date()
    private var frames = 0
    private var report: ((String) -> Void)?
    private weak var panel: NSWindow?

    // value: past the height's own 340 ms morph, which the first sizes of a content can start.
    static let span = 1.0

    func begin(_ panel: NSWindow, report: @escaping (String) -> Void) {
        end()
        self.panel = panel
        self.report = report
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
        guard t < Self.span else { return end() }
        frames += 1
        let glass = panel.frame.insetBy(dx: StatusPanel.shade, dy: StatusPanel.shade)
        report?("open t=\(Int(t * 1000)) frame=\(frames) glass=\(StatusPanel.frame(glass))")
    }
}
