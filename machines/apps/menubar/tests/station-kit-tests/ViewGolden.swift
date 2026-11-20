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

    // value: the view golden: the canvas, every token the drawing read with
    // its value once, then the tree, each line's frame relative to its
    // parent's.
    static func text(_ lines: [ViewLine], canvas: [String]) -> String {
        var reads: [TokenRead] = []
        func collect(_ line: ViewLine) {
            for read in line.mark.reads where !reads.contains(where: { $0.name == read.name }) { reads.append(read) }
            line.children.forEach(collect)
        }
        lines.forEach(collect)
        var out = ["canvas: { " + canvas.joined(separator: ", ") + " }"]
        if !reads.isEmpty { out += ["tokens:"] + reads.map { "  \($0.name): \($0.value)" } }
        out.append("tree:")
        func descendants(_ line: ViewLine) -> [ViewLine] { line.children + line.children.flatMap(descendants) }
        func write(_ line: ViewLine, parent: CGPoint, depth: Int, clip: (rect: CGRect, fade: CGFloat)?) {
            let n = ViewNumber.text
            var attrs = line.mark.kit.map { ["kit=" + $0] } ?? []
            attrs += ["at=\(n(line.frame.minX - parent.x)),\(n(line.frame.minY - parent.y))", "size=\(n(line.frame.width))x\(n(line.frame.height))"]
            attrs += line.mark.attrs
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
            let name = line.mark.name.map { " \"" + $0.replacingOccurrences(of: "\"", with: "\\\"") + "\"" } ?? ""
            let head = String(repeating: "  ", count: depth) + "- " + line.mark.role.rawValue + name + attrs.map { " [\($0)]" }.joined()
            out.append(head + (line.children.isEmpty ? "" : ":"))
            for child in line.children { write(child, parent: line.frame.origin, depth: depth + 1, clip: inner) }
        }
        for line in lines { write(line, parent: .zero, depth: 0, clip: nil) }
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

    // invariant: a view is proved by its view golden, compared whole as
    // text: nil when it is the same string, else the lines that differ,
    // its drawing written beside it as `<name>.actual.view.yaml`.
    @MainActor
    static func check(_ name: String, _ view: AnyView, scheme: ColorScheme, desk: Bool = true,
                      contrast: ColorSchemeContrast = .standard, reduceTransparency: Bool = false) throws -> String? {
        let file = "\(name).\(scheme == .dark ? "dark" : "light")"
        var canvas = ["scheme: \(scheme == .dark ? "dark" : "light")", "scale: 2", "unit: pt", "still: true", "ground: \(desk ? "desk" : "none")"]
        if contrast == .increased { canvas.append("contrast: increased") }
        if reduceTransparency { canvas.append("transparency: reduced") }
        let lines = record(view, scheme: scheme, desk: desk)
        if lines.isEmpty { return "\(file): the recorder read no view" }
        return try match(text(lines, canvas: canvas), file: file)
    }

    static func match(_ text: String, file: String) throws -> String? {
        let golden = Tree.goldens.appendingPathComponent(file + ".view.yaml")
        let actual = Tree.goldens.appendingPathComponent(file + ".actual.view.yaml")
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
