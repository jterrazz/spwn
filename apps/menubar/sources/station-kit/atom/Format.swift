import Foundation

public enum Format {
    static let suffixes = Array("KMGTPE")

    // value: a Mac's name as the engine says it: its name without
    // its domain (`jterrazz-macbook-pro`, never with `.local`).
    public static func bareName(_ raw: String) -> String {
        String(raw.split(separator: ".").first ?? Substring(raw))
    }

    public static func bytes(_ n: Int64) -> String {
        if n < 1000 { return "\(n) B" }
        var value = Double(n) / 1000
        var suffix = 0
        while value >= 999.95 && suffix < suffixes.count - 1 {
            value /= 1000
            suffix += 1
        }
        return String(format: "%.1f %@B", value, String(suffixes[suffix]))
    }

    // value: a column of sizes, each said in the unit of the largest.
    public static func column(_ sizes: [Int64]) -> [String] {
        guard let top = sizes.max(), top >= 1000 else { return sizes.map(bytes) }
        var div = 1000.0
        var suffix = 0
        while Double(top) / div >= 999.95 && suffix < suffixes.count - 1 {
            div *= 1000
            suffix += 1
        }
        return sizes.map { String(format: "%.1f %@B", Double($0) / div, String(suffixes[suffix])) }
    }

    // invariant: a total that is a whole number of GiB is memory, said as
    // macOS says it (16 GB for 16 GiB), its part counted the same way;
    // any other total in powers of 1000, as a disk's.
    static func ratio(_ used: Int64, _ total: Int64) -> String {
        let base: Double = total > 0 && total % (1 << 30) == 0 ? 1024 : 1000
        var div = 1.0
        var unit = "B"
        var step = base
        for s in suffixes where Double(total) >= step {
            div = step
            unit = "\(s)B"
            step *= base
        }
        return String(format: "%.1f/%.1f %@", Double(used) / div, Double(total) / div, unit)
    }

    public static func duration(_ seconds: Int64) -> String {
        let units: [(Int64, String)] = [(86400, "d"), (3600, "h"), (60, "min"), (1, "s")]
        var parts: [String] = []
        var rest = seconds
        for (n, name) in units {
            if rest >= n && parts.count < 2 {
                parts.append("\(rest / n) \(name)")
                rest %= n
            } else if parts.count == 1 {
                break
            }
        }
        return parts.isEmpty ? "0 s" : parts.joined(separator: " ")
    }

    static func percent(_ tenths: Int64) -> String {
        if tenths < 100 && tenths % 10 != 0 { return "\(tenths / 10).\(tenths % 10)\(glued)%" }
        return "\(tenths / 10)\(glued)%"
    }

    // value: the panel's one duration and percent format in the words the
    // engine sends a terminal (`3h 59m left`, `up 3d`, `18%`): `3 h 59 min
    // left`, `up 3 d`, `18 %`, each kept whole on its line.
    static let compact = try! NSRegularExpression(pattern: #"(?<![\w.])(\d+)(d|h|m|s)(?: (\d+)(h|m|s))?(?!\w)( left| ago)?|(?<![\w.])(\d+(?:\.\d+)?)%"#)
    static let units = ["d": "d", "h": "h", "m": "min", "s": "s"]
    static let glued = "\u{00A0}"

    public static func said(_ text: String) -> String {
        guard text.contains(where: \.isNumber) else { return text }
        var out = text
        for m in compact.matches(in: text, range: NSRange(text.startIndex..., in: text)).reversed() {
            guard let whole = Range(m.range, in: text) else { continue }
            func part(_ i: Int) -> String? { Range(m.range(at: i), in: text).map { String(text[$0]) } }
            var words: [String] = []
            if let percent = part(6) {
                words = [percent, "%"]
            } else if let n = part(1), let u = part(2) {
                words = [n, units[u] ?? u]
                if let n2 = part(3), let u2 = part(4) { words += [n2, units[u2] ?? u2] }
                if let tail = part(5) { words.append(tail.trimmingCharacters(in: .whitespaces)) }
            }
            out.replaceSubrange(Range(m.range, in: out) ?? whole, with: words.joined(separator: glued))
        }
        return out
    }

    public static func left(_ seconds: Int64) -> String {
        let minutes = Int64((Double(max(seconds, 0)) / 60).rounded())
        if minutes < 1 { return "<1\(glued)min left" }
        if minutes < 60 { return "\(minutes)\(glued)min left" }
        let hours = minutes / 60, rest = minutes % 60
        return rest == 0 ? "\(hours)\(glued)hr left" : "\(hours)\(glued)hr \(rest)\(glued)min left"
    }

    static func until(_ v: Value, now: Date?) -> String {
        var left = v.n ?? 0
        if let now, let at = v.at, let end = ISO8601DateFormatter().date(from: at) {
            left = Int64(end.timeIntervalSince(now))
        }
        if left <= 0 { return "expired" }
        if left < 60 { return "<1 min left" }
        return duration(left - left % 60) + " left"
    }

    static func age(_ v: Value) -> String {
        let n = v.n ?? 0
        if n > 30 * 86400, let at = v.at, at.count >= 10 { return String(at.prefix(10)) }
        let d = duration(n).split(separator: " ")
        let ago = "\(d.prefix(2).joined(separator: " ")) ago"
        guard let label = v.label, !label.isEmpty else { return ago }
        return "\(label) \(ago)"
    }

    public static func show(_ v: Value, now: Date? = nil) -> String {
        let n = v.n ?? 0
        switch v.kind ?? "" {
        case "", "none", "pending", "listing": return ""
        case "count":
            if let unit = v.unit, !unit.isEmpty { return "\(n) \(unit)" }
            return "\(n)"
        case "size":
            guard let unit = v.unit, !unit.isEmpty else { return bytes(n) }
            return unit.hasPrefix(Token.Glyph.slash) ? bytes(n) + unit : bytes(n) + " " + unit
        case "ratio": return ratio(n, v.of ?? 0)
        case "percent": return percent(n)
        case "duration": return duration(n)
        case "age": return age(v)
        case "until": return until(v, now: now)
        case "port": return "\(n)"
        case "portmap": return "\(n) \(Token.Glyph.arrow) \(v.of ?? 0)"
        case "range": return "\(n)\(Token.Glyph.dash)\(v.of ?? 0)"
        case "on": return "on"
        case "off": return "off"
        case "unknown": return Token.Glyph.unknown
        case "command": return "\(Token.Glyph.arrow) \(v.text ?? "")"
        case "version":
            guard let newest = v.label, !newest.isEmpty else { return v.text ?? "" }
            return "\(v.text ?? "") \(Token.Glyph.arrow) \(newest)"
        case "levels":
            return (v.items ?? []).map { "\($0.label ?? "") \(show($0, now: now))" }.joined(separator: Token.Glyph.sep)
        case "list":
            return (v.items ?? []).map { show($0, now: now) }.filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
        default: return said(v.text ?? "")
        }
    }

    public static func numeric(_ v: Value) -> Bool {
        ["count", "size", "ratio", "percent", "duration", "port", "portmap", "range"].contains(v.kind ?? "")
    }

    public static func url(_ v: Value) -> URL? {
        if v.kind == "url", let text = v.text { return URL(string: text) }
        for item in v.items ?? [] {
            if let u = url(item) { return u }
        }
        return nil
    }
}
