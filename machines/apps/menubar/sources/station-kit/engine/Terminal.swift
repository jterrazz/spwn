import Foundation

public struct Launched: Sendable, Equatable {
    public var code: Int32
    public var stderr: String
    public var stdout: String
    public init(code: Int32, stderr: String, stdout: String = "") {
        self.code = code
        self.stderr = stderr
        self.stdout = stdout
    }
}

public protocol Launcher: Sendable {
    func run(_ path: String, _ args: [String], environment: [String: String]) async -> Launched
}

// invariant: the one place the menu bar starts a program: the opening of a
// terminal (Open in Terminal) and the read of whether the engine is
// installed, which no socket can answer while the engine is down; every other
// action is a call over the engine's socket.
public struct ProcessLauncher: Launcher {
    public static let budget = 60.0

    public init() {}

    public func run(_ path: String, _ args: [String], environment: [String: String]) async -> Launched {
        await withCheckedContinuation { c in
            DispatchQueue.global(qos: .userInitiated).async {
                let p = Process()
                p.executableURL = URL(fileURLWithPath: path)
                p.arguments = args
                p.environment = ProcessInfo.processInfo.environment.merging(environment) { $1 }
                p.standardInput = FileHandle.nullDevice
                let out = Pipe()
                let err = Pipe()
                p.standardOutput = out
                p.standardError = err
                do { try p.run() } catch {
                    c.resume(returning: Launched(code: 127, stderr: "Error: \(path) could not run"))
                    return
                }
                let errData = Captured()
                let draining = DispatchGroup()
                draining.enter()
                DispatchQueue.global().async {
                    errData.set(err.fileHandleForReading.readDataToEndOfFile())
                    draining.leave()
                }
                let timer = DispatchWorkItem { if p.isRunning { p.terminate() } }
                DispatchQueue.global().asyncAfter(deadline: .now() + Self.budget, execute: timer)
                let data = out.fileHandleForReading.readDataToEndOfFile()
                draining.wait()
                p.waitUntilExit()
                timer.cancel()
                c.resume(returning: Launched(code: p.terminationStatus, stderr: String(decoding: errData.value, as: UTF8.self), stdout: String(decoding: data, as: UTF8.self)))
            }
        }
    }
}

private final class Captured: @unchecked Sendable {
    private let lock = NSLock()
    private var data = Data()
    func set(_ d: Data) { lock.withLock { data = d } }
    var value: Data { lock.withLock { data } }
}

public enum Terminal {
    // value: the program and arguments that open terminal on station, with
    // these words after it: a bare open with none, else a script that types
    // the command line, each word quoted for the shell.
    public static func opening(_ terminal: String, executable: String, words: [String]) -> (path: String, args: [String]) {
        guard !words.isEmpty else { return ("/usr/bin/open", ["-a", terminal, executable]) }
        let line = ([executable] + words).map { "'" + $0.replacingOccurrences(of: "'", with: #"'\''"#) + "'" }.joined(separator: " ")
        let quoted = line.replacingOccurrences(of: "\\", with: "\\\\").replacingOccurrences(of: "\"", with: "\\\"")
        return ("/usr/bin/osascript", ["-e", "tell application \"\(terminal)\" to do script \"\(quoted)\"", "-e", "tell application \"\(terminal)\" to activate"])
    }
}
