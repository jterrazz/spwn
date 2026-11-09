import Foundation

// invariant: SPEC B1: a sheet's answers as the engine's form reads them.
// Every word the sheet says of them is the engine's, a template naming the
// answers (Go's page.Field.Summary); the kit only fills it in: {Key} is the
// answer's label, {Key:, |none} a multiselect's labels joined by what
// follows the colon, or what follows the bar when it holds none, and a
// [part] is dropped whole when an answer it names is empty or hidden.
public struct Answers: Equatable, Sendable {
    public let form: Form
    public var given: [String: String]

    public init(_ form: Form, _ given: [String: String] = [:]) {
        self.form = form
        self.given = given
    }

    // value: what a disclosure holds while it is open; its own answer.
    public static let open = "open"

    public var fields: [Field] { Self.walk(form.fields ?? []) }

    static func walk(_ fields: [Field]) -> [Field] {
        fields.flatMap { [$0] + walk($0.fields ?? []) }
    }

    public func field(_ key: String) -> Field? { fields.first { $0.key == key } }

    public func value(_ f: Field) -> String { given[f.key ?? ""] ?? f.default ?? "" }

    public func value(_ key: String) -> String { field(key).map(value) ?? given[key] ?? "" }

    public func isOpen(_ f: Field) -> Bool { value(f) == Self.open }

    public func picked(_ f: Field) -> [String] {
        value(f).split(separator: ",").map(String.init).filter { v in f.options?.contains { $0.value == v && $0.disabled == nil } ?? true }
    }

    // value: a field is shown while its When holds, and only inside a
    // disclosure that is open.
    public func shown(_ f: Field) -> Bool {
        guard let when = f.when, let key = when.key else { return true }
        let v = field(key).map { $0.kind == "multiselect" ? picked($0).joined(separator: ",") : value($0) } ?? value(key)
        return when.set == true ? !v.isEmpty : v == (when.value ?? "")
    }

    // value: a field applies while it and every disclosure holding it
    // are shown, open or not; it is drawn only once they are open too.
    public func applies(_ f: Field) -> Bool {
        shown(f) && Self.parents(of: f, in: form.fields ?? []).allSatisfy(shown)
    }

    static func parents(of f: Field, in fields: [Field]) -> [Field] {
        for g in fields {
            if (g.fields ?? []).contains(f) { return [g] }
            let deeper = parents(of: f, in: g.fields ?? [])
            if !deeper.isEmpty { return [g] + deeper }
        }
        return []
    }

    public func label(_ f: Field, joined: String = ", ") -> String {
        if f.kind == "multiselect" {
            return picked(f).map { v in f.options?.first { $0.value == v }?.label ?? v }.joined(separator: joined)
        }
        let v = value(f)
        return f.options?.first { $0.value == v }?.label ?? v
    }

    // value: a template filled with the answers it names.
    public func fill(_ template: String) -> String {
        var out = "", part = "", inPart = false, partEmpty = false
        var i = template.startIndex
        while i < template.endIndex {
            let c = template[i]
            if c == "[" {
                inPart = true
                part = ""
                partEmpty = false
            } else if c == "]", inPart {
                inPart = false
                if !partEmpty { out += part }
            } else if c == "{", let close = template[i...].firstIndex(of: "}") {
                let said = say(String(template[template.index(after: i)..<close]))
                if inPart {
                    part += said.text
                    partEmpty = partEmpty || said.empty
                } else {
                    out += said.text
                }
                i = template.index(after: close)
                continue
            } else if inPart {
                part.append(c)
            } else {
                out.append(c)
            }
            i = template.index(after: i)
        }
        return out
    }

    // value: the glyphs a template names rather than holds, the kit's own.
    static let glyphs = ["sep": Token.Glyph.sep, "ellipsis": Token.Glyph.ellipsis]

    // value: a value of the form's words filled: its text, or its items
    // filled one by one and joined as a list is drawn.
    public func fill(_ v: Value?) -> String {
        guard let v else { return "" }
        if let items = v.items { return items.map { fill($0.text ?? "") }.filter { !$0.isEmpty }.joined(separator: Token.Glyph.sep) }
        return fill(v.text ?? "")
    }

    func say(_ slot: String) -> (text: String, empty: Bool) {
        if let glyph = Self.glyphs[slot] { return (glyph, false) }
        let parts = slot.split(separator: ":", maxSplits: 1, omittingEmptySubsequences: false).map(String.init)
        guard let f = field(parts[0]) else {
            let v = given[parts[0]] ?? ""
            return (v, v.isEmpty)
        }
        var joined = ", ", none = ""
        if parts.count == 2 {
            let rest = parts[1].split(separator: "|", maxSplits: 1, omittingEmptySubsequences: false).map(String.init)
            joined = rest[0]
            if rest.count == 2 { none = rest[1] }
        }
        let text = label(f, joined: joined)
        guard applies(f) else { return ("", true) }
        if text.isEmpty { return (none, none.isEmpty) }
        return (text, false)
    }

    // value: the answers the call is sent: each field shown and holding
    // something, never a swap, a disclosure or a field hidden.
    public var sent: [String: String] {
        var out: [String: String] = [:]
        for f in fields where f.swaps != true && f.kind != "disclosure" && applies(f) {
            guard let key = f.key else { continue }
            let v = f.kind == "multiselect" ? (picked(f).isEmpty ? "" : Self.list(picked(f))) : value(f)
            if !v.isEmpty { out[key] = v }
        }
        return out
    }

    // value: a multiselect's answer as the call reads it, a JSON list.
    static func list(_ values: [String]) -> String {
        guard let data = try? JSONSerialization.data(withJSONObject: values), let text = String(data: data, encoding: .utf8) else { return "" }
        return text
    }

    // value: the HUD a create's sheet says before it runs anything: a name
    // already held refuses it, the field's own words.
    public var refused: String? {
        for f in fields where f.taken?.contains(value(f).trimmingCharacters(in: .whitespaces)) == true {
            return fill(f.taken_hud ?? "")
        }
        return nil
    }
}

// value: what a sheet's primary button does with its answers: runs the
// call, its HUD said once it is sent, or refuses it with the HUD alone.
public enum SheetOutcome: Equatable, Sendable {
    case runs(answers: [String: String], hud: String?)
    case refused(hud: String)

    public static func of(_ action: Action, _ answers: Answers) -> SheetOutcome {
        if let hud = answers.refused { return .refused(hud: hud) }
        return .runs(answers: answers.sent, hud: action.hud.map(answers.fill))
    }
}
