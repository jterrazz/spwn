import Foundation
import StationKit
import SwiftUI

struct ViewLine: Equatable {
    let mark: ViewMark
    let frame: CGRect
    let children: [ViewLine]

    init(mark: ViewMark, frame: CGRect, children: [ViewLine]) {
        self.mark = mark
        self.frame = frame
        self.children = children
    }

    init(_ recorded: RecordedView, _ proxy: GeometryProxy) {
        let rect = proxy[recorded.anchor]
        func grid(_ v: CGFloat) -> CGFloat { (v * 2).rounded() / 2 }
        self.init(mark: recorded.mark, frame: CGRect(x: grid(rect.minX), y: grid(rect.minY), width: grid(rect.width), height: grid(rect.height)),
                  children: recorded.children.map { ViewLine($0, proxy) })
    }
}

struct ViewNode: Equatable {
    let key: String
    let line: String
    let children: [ViewNode]
}

final class ViewSink {
    var lines: [ViewLine] = []

    func keep(_ lines: [ViewLine]) { self.lines = lines }
}

enum ViewGolden {

    // invariant: the recorder draws the view as its PNG golden does, over
    // the same canvas, and reads every recorded view's frame in the
    // canvas's points.
    @MainActor
    static func record(_ view: AnyView, scheme: ColorScheme, desk: Bool) -> [ViewLine] {
        let sink = ViewSink()
        let content = Snapshot.canvas(view, scheme: scheme, over: desk ? AnyView(Color.clear) : nil)
            .overlayPreferenceValue(RecordedViews.self) { recorded in
                GeometryReader { proxy in
                    let _ = sink.keep(recorded.map { ViewLine($0, proxy) })
                    Color.clear
                }
            }
        let renderer = ImageRenderer(content: content)
        renderer.scale = 2
        _ = renderer.cgImage
        return sink.lines
    }

    static func reads(_ lines: [ViewLine]) -> [TokenRead] {
        var reads: [TokenRead] = []
        func collect(_ line: ViewLine) {
            for read in line.mark.reads where !reads.contains(where: { $0.name == read.name }) { reads.append(read) }
            line.children.forEach(collect)
        }
        lines.forEach(collect)
        return reads
    }

    // value: the recorded views as the lines a view golden writes, each
    // with its frame relative to its parent's, its states, its lines of
    // words and what of it a scrolled parent hides.
    static func nodes(_ lines: [ViewLine]) -> [ViewNode] {
        func descendants(_ line: ViewLine) -> [ViewLine] { line.children + line.children.flatMap(descendants) }
        func node(_ line: ViewLine, parent: CGPoint, clip: (rect: CGRect, fade: CGFloat)?) -> ViewNode {
            let n = ViewNumber.text
            let name = line.mark.name.map { " \"" + $0.replacingOccurrences(of: "\"", with: "\\\"") + "\"" } ?? ""
            let key = line.mark.role.rawValue + name + (line.mark.kit.map { " [kit=\($0)]" } ?? "")
            var attrs = ["at=\(n(line.frame.minX - parent.x)),\(n(line.frame.minY - parent.y))", "size=\(n(line.frame.width))x\(n(line.frame.height))"]
            attrs += line.mark.attrs
            attrs += line.mark.states.map { "state=" + $0.rawValue }
            if let pitch = line.mark.pitch, pitch > 0 {
                let count = Int((line.frame.height / pitch).rounded())
                if count > 1 { attrs.append("lines=\(count)") }
            }
            var inner = clip
            if let own = line.mark.clip {
                let overflows = descendants(line).contains { $0.frame.maxY > line.frame.maxY + Token.Measure.grid }
                if overflows { attrs.append("fade=bottom " + own.token) }
                inner = (line.frame, overflows ? min(own.fade, line.frame.height) : 0)
            }
            if let clip {
                if line.frame.minY >= clip.rect.maxY {
                    attrs.append("hidden=scrolled")
                } else {
                    if line.frame.maxY > clip.rect.maxY { attrs.append("visible=" + n(clip.rect.maxY - line.frame.minY)) }
                    if clip.fade > 0 && line.frame.maxY > clip.rect.maxY - clip.fade { attrs.append("under=fade") }
                }
            }
            return ViewNode(key: key, line: key + attrs.map { " [\($0)]" }.joined(),
                            children: line.children.map { node($0, parent: line.frame.origin, clip: inner) })
        }
        return lines.map { node($0, parent: .zero, clip: nil) }
    }

    static func tree(_ nodes: [ViewNode], depth: Int = 0) -> [String] {
        nodes.flatMap { node in
            [String(repeating: "  ", count: depth) + "- " + node.line + (node.children.isEmpty ? "" : ":")] + tree(node.children, depth: depth + 1)
        }
    }

    // value: the view golden: the canvas, every token the drawing read with
    // its value once, then the tree, each line's frame relative to its
    // parent's.
    static func text(_ lines: [ViewLine], canvas: [String]) -> String {
        let reads = Self.reads(lines)
        var out = ["canvas: { " + canvas.joined(separator: ", ") + " }"]
        if !reads.isEmpty { out += ["tokens:"] + reads.map { "  \($0.name): \($0.value)" } }
        out.append("tree:")
        out += tree(nodes(lines))
        return out.joined(separator: "\n") + "\n"
    }

    // value: what an overlay owns over its base, each entry under the path
    // of its parent: a line of the base drawn otherwise (changes), a view
    // the base does not draw (added), a view of the base the overlay hides
    // (removed). Siblings are paired by their role, words and kit, in order.
    static func delta(base: [ViewNode], over: [ViewNode]) -> (changes: [String], added: [String], removed: [String]) {
        var changes: [String] = [], added: [String] = [], removed: [String] = []
        func paths(_ nodes: [ViewNode]) -> [String] {
            nodes.enumerated().map { i, node in
                let same = nodes.filter { $0.key == node.key }.count
                return same > 1 ? node.key + " #\(nodes[..<i].filter { $0.key == node.key }.count + 1)" : node.key
            }
        }
        func under(_ entries: inout [String], _ parent: String, _ lines: [String]) {
            guard !lines.isEmpty else { return }
            entries.append("- in " + parent + ":")
            entries += lines.map { "  " + $0 }
        }
        func walk(_ a: [ViewNode], _ b: [ViewNode], parent: String) {
            var common = Array(repeating: Array(repeating: 0, count: b.count + 1), count: a.count + 1)
            for i in stride(from: a.count - 1, through: 0, by: -1) {
                for j in stride(from: b.count - 1, through: 0, by: -1) {
                    common[i][j] = a[i].key == b[j].key ? common[i + 1][j + 1] + 1 : max(common[i + 1][j], common[i][j + 1])
                }
            }
            let named = paths(b), was = paths(a)
            var changed: [String] = [], new: [String] = [], gone: [String] = [], pairs: [(ViewNode, ViewNode, String)] = []
            var i = 0, j = 0
            while i < a.count || j < b.count {
                if i < a.count, j < b.count, a[i].key == b[j].key {
                    if a[i].line != b[j].line { changed.append("- " + b[j].line) }
                    pairs.append((a[i], b[j], named[j]))
                    i += 1
                    j += 1
                } else if j < b.count, i == a.count || common[i][j + 1] >= common[i + 1][j] {
                    new += tree([b[j]])
                    j += 1
                } else {
                    gone.append("- " + was[i])
                    i += 1
                }
            }
            under(&changes, parent, changed)
            under(&added, parent, new)
            under(&removed, parent, gone)
            for (x, y, path) in pairs { walk(x.children, y.children, parent: parent == "root" ? path : parent + " › " + path) }
        }
        walk(base, over, parent: "root")
        return (changes, added, removed)
    }

    // value: an overlay's view golden: its canvas, the base it is drawn
    // over, the tokens it reads that the base does not, and its delta.
    static func text(_ lines: [ViewLine], canvas: [String], base: String, over baseLines: [ViewLine]) -> String {
        let held = Self.reads(baseLines)
        let reads = Self.reads(lines).filter { !held.contains($0) }
        var out = ["canvas: { " + canvas.joined(separator: ", ") + " }", "base: ./" + base + ".view.yaml"]
        if !reads.isEmpty { out += ["tokens:"] + reads.map { "  \($0.name): \($0.value)" } }
        let delta = Self.delta(base: nodes(baseLines), over: nodes(lines))
        for (title, entries) in [("changes", delta.changes), ("added", delta.added), ("removed", delta.removed)] where !entries.isEmpty {
            out += [title + ":"] + entries
        }
        return out.joined(separator: "\n") + "\n"
    }

    // value: the lines of want and got that differ, `-` for the golden's
    // and `+` for the drawing's, in the order of a longest common run.
    static func diff(_ want: String, _ got: String) -> [String] {
        let a = want.components(separatedBy: "\n"), b = got.components(separatedBy: "\n")
        var common = Array(repeating: Array(repeating: 0, count: b.count + 1), count: a.count + 1)
        for i in stride(from: a.count - 1, through: 0, by: -1) {
            for j in stride(from: b.count - 1, through: 0, by: -1) {
                common[i][j] = a[i] == b[j] ? common[i + 1][j + 1] + 1 : max(common[i + 1][j], common[i][j + 1])
            }
        }
        var out: [String] = []
        var i = 0, j = 0
        while i < a.count || j < b.count {
            if i < a.count, j < b.count, a[i] == b[j] {
                i += 1
                j += 1
            } else if j < b.count, i == a.count || common[i][j + 1] >= common[i + 1][j] {
                out.append("+" + b[j])
                j += 1
            } else {
                out.append("-" + a[i])
                i += 1
            }
        }
        return out
    }

    static func canvas(_ scheme: ColorScheme, ground: String, contrast: ColorSchemeContrast, reduceTransparency: Bool) -> [String] {
        var canvas = ["scheme: \(scheme == .dark ? "dark" : "light")", "scale: 2", "unit: pt", "still: true", "ground: \(ground)"]
        if contrast == .increased { canvas.append("contrast: increased") }
        if reduceTransparency { canvas.append("transparency: reduced") }
        return canvas
    }

    // invariant: a view is proved by its view golden, compared whole as
    // text: nil when it is the same string, else the lines that differ,
    // its drawing written beside it as `<name>.actual.view.yaml`. A view
    // drawn over a base is written as its delta over the base's drawing:
    // with the base's own golden proved beside it, the pair holds the
    // whole drawing, each fact once.
    @MainActor
    static func check(_ name: String, _ view: AnyView, scheme: ColorScheme, desk: Bool = true, ground: String? = nil,
                      contrast: ColorSchemeContrast = .standard, reduceTransparency: Bool = false,
                      in folder: URL = Tree.goldens, over base: (name: String, view: AnyView)? = nil) throws -> String? {
        let suffix = scheme == .dark ? "dark" : "light"
        let file = "\(name).\(suffix)"
        let canvas = Self.canvas(scheme, ground: ground ?? (desk ? "desk" : "none"), contrast: contrast, reduceTransparency: reduceTransparency)
        let lines = record(view, scheme: scheme, desk: desk)
        if lines.isEmpty { return "\(file): the recorder read no view" }
        guard let base else { return try match(text(lines, canvas: canvas), file: file, in: folder) }
        let under = record(base.view, scheme: scheme, desk: desk)
        if under.isEmpty { return "\(file): the recorder read no view of its base \(base.name)" }
        return try match(text(lines, canvas: canvas, base: "\(base.name).\(suffix)", over: under), file: file, in: folder)
    }

    static func match(_ text: String, file: String, in folder: URL = Tree.goldens) throws -> String? {
        let golden = folder.appendingPathComponent(file + ".view.yaml")
        let actual = folder.appendingPathComponent(file + ".actual.view.yaml")
        let want = try? String(contentsOf: golden, encoding: .utf8)
        let listed = Expectation.listed(golden, in: Golden.expected)
        if want == text {
            try? FileManager.default.removeItem(at: actual)
            return listed ? "\(file).view.yaml: listed in STATION_EXPECT_GOLDENS, but it did not change" : nil
        }
        try FileManager.default.createDirectory(at: golden.deletingLastPathComponent(), withIntermediateDirectories: true)
        if Golden.update || listed {
            try Data(text.utf8).write(to: golden)
            try? FileManager.default.removeItem(at: actual)
            return nil
        }
        try Data(text.utf8).write(to: actual)
        guard let want else { return "\(file).view.yaml: no golden (STATION_UPDATE_GOLDENS=1 swift test writes it)" }
        return "\(file).view.yaml differs:\n" + diff(want, text).joined(separator: "\n")
    }
}
