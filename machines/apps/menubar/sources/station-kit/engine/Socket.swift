import Foundation

public enum SocketError: Error, Equatable {
    case refused(Int32)
    case closed
}

public final class UnixSocket: @unchecked Sendable {
    let fd: Int32
    private var buffer = Data()
    private let lock = NSLock()

    init(fd: Int32) { self.fd = fd }

    public static func connect(_ path: String) throws -> UnixSocket {
        let fd = socket(AF_UNIX, SOCK_STREAM, 0)
        guard fd >= 0 else { throw SocketError.refused(errno) }
        var one: Int32 = 1
        setsockopt(fd, SOL_SOCKET, SO_NOSIGPIPE, &one, socklen_t(MemoryLayout<Int32>.size))
        var addr = sockaddr_un()
        addr.sun_family = sa_family_t(AF_UNIX)
        let bytes = Array(path.utf8)
        guard bytes.count < MemoryLayout.size(ofValue: addr.sun_path) else {
            close(fd)
            throw SocketError.refused(ENAMETOOLONG)
        }
        withUnsafeMutableBytes(of: &addr.sun_path) { raw in
            raw.copyBytes(from: bytes)
            raw[bytes.count] = 0
        }
        let size = socklen_t(MemoryLayout<sockaddr_un>.size)
        let result = withUnsafePointer(to: &addr) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) { Darwin.connect(fd, $0, size) }
        }
        guard result == 0 else {
            let code = errno
            close(fd)
            throw SocketError.refused(code)
        }
        return UnixSocket(fd: fd)
    }

    public func write(line: Data) throws {
        var data = line
        data.append(0x0A)
        try data.withUnsafeBytes { raw in
            var sent = 0
            while sent < raw.count {
                let n = Darwin.write(fd, raw.baseAddress! + sent, raw.count - sent)
                if n <= 0 { throw SocketError.closed }
                sent += n
            }
        }
    }

    public func readLine() throws -> Data {
        while true {
            if let i = buffer.firstIndex(of: 0x0A) {
                let line = buffer[buffer.startIndex..<i]
                buffer.removeSubrange(buffer.startIndex...i)
                return Data(line)
            }
            var chunk = [UInt8](repeating: 0, count: 65536)
            let n = Darwin.read(fd, &chunk, chunk.count)
            if n <= 0 { throw SocketError.closed }
            buffer.append(contentsOf: chunk[0..<n])
        }
    }

    public func shutdownBoth() { Darwin.shutdown(fd, SHUT_RDWR) }

    deinit { close(fd) }
}

public enum EnginePath {
    public static func socket(environment: [String: String] = ProcessInfo.processInfo.environment) -> String {
        if let state = environment["XDG_STATE_HOME"], !state.isEmpty { return state + "/station/engine.sock" }
        return (environment["HOME"] ?? NSHomeDirectory()) + "/.local/state/station/engine.sock"
    }

    public static func agentPlist(environment: [String: String] = ProcessInfo.processInfo.environment) -> String {
        (environment["HOME"] ?? NSHomeDirectory()) + "/Library/LaunchAgents/com.jterrazz.station.engine.plist"
    }
}
