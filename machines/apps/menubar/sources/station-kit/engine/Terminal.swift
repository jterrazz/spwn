import Foundation

public struct Launched: Sendable, Equatable {
    public var code: Int32
    public var stderr: String
    public init(code: Int32, stderr: String) {
        self.code = code
        self.stderr = stderr
    }
}

public protocol Launcher: Sendable {
    func run(_ path: String, _ args: [String], environment: [String: String]) async -> Launched
}

// invariant: the one place the menu bar starts a program, and it starts only
// the opening of a terminal (Open in Terminal): every other action is a call
// over the engine's socket.
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
                p.standardOutput = FileHandle.nullDevice
                let err = Pipe()
                p.standardError = err
                do { try p.run() } catch {
                    c.resume(returning: Launched(code: 127, stderr: "Error: \(path) could not run"))
                    return
                }
                let timer = DispatchWorkItem { if p.isRunning { p.terminate() } }
                DispatchQueue.global().asyncAfter(deadline: .now() + Self.budget, execute: timer)
                let data = err.fileHandleForReading.readDataToEndOfFile()
                p.waitUntilExit()
                timer.cancel()
                c.resume(returning: Launched(code: p.terminationStatus, stderr: String(decoding: data, as: UTF8.self)))
            }
        }
    }
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
