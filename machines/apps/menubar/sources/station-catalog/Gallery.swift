import Foundation
import StationKit

enum Gallery {
    struct Figure: Equatable {
        let caption: String
        let file: String
    }

    struct Card: Equatable {
        let id: String
        let note: String
        let failing: [String]
        let renders: [Figure]
        let drift: [Figure]
        let prototype: [Figure]
    }

    struct Section: Equatable {
        let title: String
        let cards: [Card]

        var anchor: String {
            "g-" + title.lowercased().map { $0.isLetter || $0.isNumber ? String($0) : "-" }.joined()
        }
    }

    static let folders = ["golden", "actual", "prototype"]

    static func write(_ stories: [Story], to folder: URL, root: URL, goldens: URL, design: URL?) throws -> URL {
        let files = FileManager.default
        for name in folders { try? files.removeItem(at: folder.appendingPathComponent(name)) }
        func copy(_ source: URL?, as name: String) throws -> String? {
            guard let source else { return nil }
            let target = folder.appendingPathComponent(name)
            try files.createDirectory(at: target.deletingLastPathComponent(), withIntermediateDirectories: true)
            try files.copyItem(at: source, to: target)
            return name
        }
        let order = Paths.order(root: root)
        let rank = Dictionary(order.enumerated().map { ($1.id, $0) }) { first, _ in first }
        let titles = Dictionary(order.map { ($0.id, $0.section) }) { first, _ in first }
        var sections: [Section] = []
        for story in stories.sorted(by: { (rank[$0.id] ?? .max, $0.id) < (rank[$1.id] ?? .max, $1.id) }) {
            var failing: [String] = []
            var drift: [Figure] = []
            for scheme in Paths.schemes {
                guard let actual = Paths.actual(story.id, scheme: scheme, goldens: goldens) else { continue }
                failing.append(scheme)
                let golden = goldens.appendingPathComponent("\(story.id).\(scheme).png")
                if let file = try copy(files.fileExists(atPath: golden.path) ? golden : nil, as: "golden/\(story.id).\(scheme).png") {
                    drift.append(Figure(caption: "Golden, \(scheme)", file: file))
                }
                if let file = try copy(actual, as: "actual/\(story.id).\(scheme).png") {
                    drift.append(Figure(caption: "Actual, \(scheme)", file: file))
                }
            }
            var prototype: [Figure] = []
            for dark in [true, false] {
                if let file = try copy(Paths.capture(story.id, dark: dark, design: design),
                                       as: "prototype/\(story.id)\(dark ? "" : ".light").png") {
                    prototype.append(Figure(caption: "Prototype, \(dark ? "dark" : "light")", file: file))
                }
            }
            let card = Card(id: story.id, note: story.note, failing: failing,
                            renders: Paths.schemes.map { Figure(caption: $0.capitalized, file: "\(story.id).\($0).png") },
                            drift: drift, prototype: prototype)
            let title = titles[story.id] ?? story.group.rawValue.capitalized
            if sections.last?.title == title {
                sections[sections.count - 1] = Section(title: title, cards: sections[sections.count - 1].cards + [card])
            } else {
                sections.append(Section(title: title, cards: [card]))
            }
        }
        let page = folder.appendingPathComponent("index.html")
        try html(sections).write(to: page, atomically: true, encoding: .utf8)
        return page
    }

    static func escape(_ text: String) -> String {
        text.replacingOccurrences(of: "&", with: "&amp;").replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;").replacingOccurrences(of: "\"", with: "&quot;")
    }

    static func figures(_ figures: [Figure], id: String, kind: String) -> String {
        guard !figures.isEmpty else { return "" }
        let items = figures.map { figure in
            let file = escape(figure.file)
            return "<figure><a href=\"\(file)\"><img loading=\"lazy\" src=\"\(file)\" alt=\"\(escape(id)), \(escape(figure.caption))\"></a>"
                + "<figcaption>\(escape(figure.caption))</figcaption></figure>"
        }
        return "<div class=\"pair \(kind)\">\(items.joined())</div>"
    }

    static func html(_ sections: [Section]) -> String {
        let cards = sections.flatMap(\.cards)
        let failing = cards.filter { !$0.failing.isEmpty }.count
        var out = """
        <!DOCTYPE html>
        <html lang="en">
        <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <title>Station stories</title>
        <style>
        :root { color-scheme: light dark; --line: light-dark(#d9d9de, #3a3a40); --card: light-dark(#fff, #1e1e22); --dim: light-dark(#6e6e73, #98989f); --red: light-dark(#d70015, #ff6961); }
        * { box-sizing: border-box; }
        body { margin: 0; font: 15px/1.4 -apple-system, BlinkMacSystemFont, system-ui, sans-serif; background: light-dark(#f2f2f7, #111114); }
        header.top { padding: 16px 16px 8px; }
        header.top h1 { margin: 0; font-size: 22px; }
        header.top p { margin: 4px 0 0; color: var(--dim); }
        nav { position: sticky; top: 0; z-index: 1; display: flex; gap: 6px; overflow-x: auto; padding: 8px 16px; background: light-dark(rgba(242,242,247,.92), rgba(17,17,20,.92)); backdrop-filter: blur(12px); border-bottom: 1px solid var(--line); }
        nav a { flex: none; padding: 4px 10px; border-radius: 999px; background: var(--card); border: 1px solid var(--line); color: inherit; text-decoration: none; font-size: 13px; white-space: nowrap; }
        section { padding: 0 16px; scroll-margin-top: 48px; }
        section h2 { font-size: 17px; margin: 20px 0 10px; }
        .grid { display: grid; grid-template-columns: 1fr; gap: 12px; }
        @media (min-width: 600px) { .grid { grid-template-columns: 1fr 1fr; } }
        article { background: var(--card); border: 1px solid var(--line); border-radius: 12px; padding: 12px; min-width: 0; }
        article.failing { border: 2px solid var(--red); }
        article h3 { margin: 0; font: 600 14px ui-monospace, SFMono-Regular, Menlo, monospace; }
        article p { margin: 4px 0 10px; color: var(--dim); }
        .flag { color: var(--red); font-weight: 600; font-size: 13px; }
        .pair { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-top: 8px; }
        figure { margin: 0; min-width: 0; }
        figure img { display: block; width: 100%; height: auto; border-radius: 6px; }
        figcaption { font-size: 12px; color: var(--dim); margin-top: 2px; }
        .drift figcaption { color: var(--red); }
        </style>
        </head>
        <body>
        <header class="top"><h1>Station stories</h1><p>\(cards.count) stories, \(failing) failing</p></header>

        """
        out += "<nav>" + sections.map { "<a href=\"#\($0.anchor)\">\(escape($0.title))</a>" }.joined() + "</nav>\n"
        for section in sections {
            out += "<section id=\"\(section.anchor)\"><h2>\(escape(section.title))</h2><div class=\"grid\">\n"
            for card in section.cards {
                let flag = card.failing.isEmpty ? "" : "<div class=\"flag\">Failing in \(card.failing.joined(separator: " and "))</div>"
                out += "<article id=\"\(escape(card.id))\"\(card.failing.isEmpty ? "" : " class=\"failing\"")>"
                    + "<h3>\(escape(card.id))</h3>\(flag)<p>\(escape(card.note))</p>"
                    + figures(card.renders, id: card.id, kind: "render")
                    + figures(card.drift, id: card.id, kind: "drift")
                    + figures(card.prototype, id: card.id, kind: "prototype")
                    + "</article>\n"
            }
            out += "</div></section>\n"
        }
        return out + "</body>\n</html>\n"
    }
}
