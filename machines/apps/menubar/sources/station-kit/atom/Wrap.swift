import SwiftUI

// invariant: words wrap as the vision's browser wraps them: each line
// takes every word that fits, and none is pushed down to keep the last
// line company, as Text does; past its line limit the words end in an
// ellipsis.
public struct Wrap: View {
    let text: String
    let spacing: CGFloat
    let alignment: TextAlignment
    @Environment(\.lineLimit) private var limit

    public init(_ text: String, spacing: CGFloat = 0, alignment: TextAlignment = .leading) {
        self.text = text
        self.spacing = spacing
        self.alignment = alignment
    }

    var words: [String] { text.split(separator: " ", omittingEmptySubsequences: true).map(String.init) }

    public var body: some View {
        if limit == 1 {
            Text(text).lineLimit(1)
        } else {
            WordFlow(spacing: spacing, alignment: alignment, limit: limit) {
                Text("x x").hidden()
                Text("xx").hidden()
                Text(Token.Glyph.ellipsis)
                ForEach(Array(words.enumerated()), id: \.offset) { _, word in Text(word).lineLimit(1).fixedSize() }
            }
            .clipped()
            .accessibilityElement(children: .ignore)
            .accessibilityLabel(text)
        }
    }
}

struct WordFlow: SwiftUI.Layout {
    let spacing: CGFloat
    let alignment: TextAlignment
    let limit: Int?

    static let measures = 3
    // value: the hair a browser's line keeps free that CoreText's fills;
    // the flow's own width counts it, so a line placed in the width it
    // asked for breaks where it was measured.
    static let slack: CGFloat = 1

    struct Line {
        var words: [Int] = []
        var width: CGFloat = 0
        var height: CGFloat = 0
    }

    struct Flow {
        var lines: [Line]
        var cut: Bool
        var space: CGFloat
        var ellipsis: CGSize
    }

    func flow(_ width: CGFloat?, _ subviews: Subviews) -> Flow {
        let space = subviews[0].sizeThatFits(.unspecified).width - subviews[1].sizeThatFits(.unspecified).width
        let ellipsis = subviews[2].sizeThatFits(.unspecified)
        let room = width ?? .infinity
        var lines: [Line] = []
        var line = Line()
        for i in Self.measures..<subviews.count {
            let size = subviews[i].sizeThatFits(.unspecified)
            let needed = line.words.isEmpty ? size.width : line.width + space + size.width
            if !line.words.isEmpty && needed + Self.slack > room {
                lines.append(line)
                line = Line()
                line.width = size.width
            } else {
                line.width = needed
            }
            line.words.append(i)
            line.height = max(line.height, size.height)
        }
        if !line.words.isEmpty { lines.append(line) }
        var cut = false
        if let limit, limit > 0, lines.count > limit {
            lines = Array(lines.prefix(limit))
            cut = true
        }
        if lines.isEmpty { lines = [Line(words: [], width: 0, height: ellipsis.height)] }
        return Flow(lines: lines, cut: cut, space: space, ellipsis: ellipsis)
    }

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let f = flow(proposal.width, subviews)
        let widest = (f.lines.map(\.width).max() ?? 0) + Self.slack + Token.Measure.grid
        let height = f.lines.map(\.height).reduce(0, +) + spacing * CGFloat(f.lines.count - 1)
        let width = alignment == .leading ? widest : (proposal.width.map { $0.isFinite ? $0 : widest } ?? widest)
        return CGSize(width: width, height: height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let f = flow(bounds.width, subviews)
        let away = CGPoint(x: bounds.maxX + 10_000, y: bounds.minY)
        for i in 0..<subviews.count { subviews[i].place(at: away, proposal: .unspecified) }
        var y = bounds.minY
        for (n, line) in f.lines.enumerated() {
            let last = f.cut && n == f.lines.count - 1
            let width = last ? line.width + f.ellipsis.width : line.width
            var x: CGFloat
            switch alignment {
            case .center: x = bounds.minX + (bounds.width - width) / 2
            case .trailing: x = bounds.maxX - width
            default: x = bounds.minX
            }
            for i in line.words {
                let size = subviews[i].sizeThatFits(.unspecified)
                subviews[i].place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(size))
                x += size.width + f.space
            }
            if last { subviews[2].place(at: CGPoint(x: x - f.space, y: y), proposal: .unspecified) }
            y += line.height + spacing
        }
    }
}
