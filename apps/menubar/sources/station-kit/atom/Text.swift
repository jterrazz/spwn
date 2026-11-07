import SwiftUI

private struct SamplesKey: EnvironmentKey {
    static let defaultValue: [String: Double] = [:]
}

extension EnvironmentValues {
    public var stationSamples: [String: Double] {
        get { self[SamplesKey.self] }
        set { self[SamplesKey.self] = newValue }
    }
}

public struct KeyCap: View {
    let key: String
    let onTint: Bool
    @Environment(\.stationTokens) private var tokens

    public init(_ key: String, onTint: Bool = false) {
        self.key = key
        self.onTint = onTint
    }

    public var body: some View {
        let c = Token.Metric.Control.self
        Text(Token.Key.cap(key))
            .font(Token.Role.keyCap.font)
            .tracking(c.keyCapTracking)
            .foregroundStyle(onTint ? tokens.onTint : tokens.tone(Token.Role.keyCap.tone))
            .padding(c.keyCapPadding)
            .frame(minHeight: c.keyCap)
            .background(
                RoundedRectangle(cornerRadius: c.keyCapRadius, style: .continuous)
                    .fill(onTint ? tokens.onTint.opacity(c.keyOnTintShare) : tokens.color.key))
            .fixedSize()
    }
}

public struct Pill: View {
    public enum Reach: Sendable { case tailnet, open, unshared, off }

    let address: String
    let reach: Reach
    @Environment(\.stationTokens) private var tokens

    public init(_ address: String, reach: Reach) {
        self.address = address
        self.reach = reach
    }

    var dot: Color? {
        switch reach {
        case .tailnet: tokens.color.green
        case .open: tokens.color.accent
        case .unshared: nil
        case .off: tokens.color.t3
        }
    }

    public var body: some View {
        let c = Token.Metric.Control.self
        let shape = Capsule(style: .continuous)
        HStack(spacing: c.pillGap) {
            if let dot { Circle().fill(dot).frame(width: c.pillDot, height: c.pillDot) }
            Text(address)
                .font(Token.Role.address.font)
                .foregroundStyle(reach == .off ? tokens.color.t3 : reach == .unshared ? tokens.color.t2 : tokens.color.t1)
                .lineLimit(1)
                .truncationMode(.middle)
        }
        .padding(.horizontal, c.pillPadding)
        .frame(height: c.pill.height)
        .background {
            if reach == .unshared {
                shape.strokeBorder(tokens.color.t3, style: StrokeStyle(lineWidth: c.pillDash, dash: c.pillDashes))
            } else {
                shape.fill(tokens.color.q)
            }
        }
        .frame(maxWidth: c.pillMax, alignment: .leading)
        .fixedSize()
    }
}

public enum Live {
    public static func value(_ v: Value, samples: [String: Double]) -> Value {
        guard let key = v.live, let sample = samples[key] else { return v }
        var out = v
        switch v.kind {
        case "percent": out.n = Int64((sample * Token.Live.tenths).rounded())
        case "size": out.n = Int64((sample * Token.Live.giga).rounded())
        default: out.n = Int64(sample.rounded())
        }
        out.at = nil
        return out
    }
}

public struct LiveText: View {
    let value: Value
    let role: Token.Role
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationSamples) private var samples
    @Environment(\.stationNow) private var now

    public init(_ value: Value, role: Token.Role = .rowValue) {
        self.value = value
        self.role = role
    }

    public var body: some View {
        let shown = Live.value(value, samples: samples)
        Text(Format.show(shown, now: now))
            .font(role.font)
            .foregroundStyle(value.stale == true ? tokens.color.t3 : value.signal != nil ? tokens.signalText(value.signal) : tokens.tone(role.tone))
            .contentTransition(.numericText(value: Double(shown.n ?? 0)))
            .animation(tokens.motion(Token.Motion.sample), value: shown.n)
            .lineLimit(1)
    }
}

public struct FactsLine: View {
    let parts: [Value]
    let role: Token.Role
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationNow) private var now

    public init(_ value: Value?, role: Token.Role = .facts) {
        parts = value.map { $0.items ?? [$0] } ?? []
        self.role = role
    }

    public static func variants(_ parts: [Value]) -> [[Value]] {
        var shown = parts
        var out = [shown]
        let order = parts.enumerated().filter { ($0.element.pri ?? 0) > 0 }.sorted { ($0.element.pri ?? 0) > ($1.element.pri ?? 0) }.map(\.offset)
        var dropped = Set<Int>()
        for i in order {
            dropped.insert(i)
            shown = parts.enumerated().filter { !dropped.contains($0.offset) }.map(\.element)
            out.append(shown)
        }
        let short = shown.compactMap { part -> Value? in
            guard let s = part.short else { return part }
            if s.isEmpty { return nil }
            var v = part
            v.kind = "text"
            v.text = s
            return v
        }
        if short != shown { out.append(short) }
        return out
    }

    public static func words(_ parts: [Value], now: Date? = nil) -> String {
        parts.map { Format.show($0, now: now) }.filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep)
    }

    public static func fit(_ parts: [Value], width: CGFloat, now: Date? = nil, measure: (String) -> CGFloat) -> String {
        let all = variants(parts).map { words($0, now: now) }
        return all.first { measure($0) <= width } ?? all.last ?? ""
    }

    func line(_ parts: [Value]) -> Text {
        var text = Text(verbatim: "")
        var first = true
        for part in parts {
            let word = Format.show(part, now: now)
            if word.isEmpty { continue }
            if !first { text = text + Text(verbatim: Token.Glyph.sep) }
            first = false
            text = text + Text(verbatim: word).foregroundStyle(part.signal.map(tokens.signalText) ?? tokens.tone(role.tone))
        }
        return text
    }

    public var body: some View {
        ViewThatFits(in: .horizontal) {
            ForEach(Array(Self.variants(parts).enumerated()), id: \.offset) { _, variant in
                line(variant).font(role.font).foregroundStyle(tokens.tone(role.tone)).lineLimit(1).fixedSize()
            }
        }
    }
}

public struct TimeLeft: View {
    let seconds: Int64
    let total: Int64
    @Environment(\.stationTokens) private var tokens

    public init(_ value: Value) {
        seconds = value.n ?? 0
        total = max(value.of ?? value.n ?? 1, 1)
    }

    var low: Bool { seconds < Token.Hot.leftSeconds }

    public var body: some View {
        let c = Token.Metric.Control.self
        let share = min(max(Double(seconds) / Double(total), Token.Metric.Marks.progressLeast), 1)
        VStack(alignment: .trailing, spacing: c.timeLeftGap) {
            Text(Format.left(seconds))
                .font(Token.Role.rowSub.font)
                .foregroundStyle(low ? tokens.color.orangeText : tokens.color.t2)
                .lineLimit(1)
            Capsule().fill(tokens.color.track)
                .frame(height: c.timeLeftBar)
                .overlay(alignment: .leading) {
                    GeometryReader { g in
                        Capsule().fill(low ? tokens.color.orange : tokens.color.accent).frame(width: g.size.width * share)
                    }
                }
                .animation(tokens.motion(Token.Motion.progress), value: share)
        }
        .frame(width: c.timeLeft, alignment: .trailing)
    }
}
