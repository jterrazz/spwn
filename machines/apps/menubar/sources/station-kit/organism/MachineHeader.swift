import SwiftUI

// invariant: SPEC A4: a machine page's header: the way back, the
// machine's glyph and its title (the H1 the engine sent), its buttons (New VM, its menu), its facts, a
// fact that links pushing its page, and its segments.
public struct MachineHeader: View {
    let page: Page
    let segment: String?
    let pick: (String) -> Void
    let back: () -> Void
    let run: (Action) -> Void
    let open: (String) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still
    @Environment(\.stationMenuPick) private var menuPick
    @Environment(\.stationCaught) private var caught

    public init(_ page: Page, segment: String?, pick: @escaping (String) -> Void = { _ in }, back: @escaping () -> Void = {}, run: @escaping (Action) -> Void = { _ in }, open: @escaping (String) -> Void = { _ in }) {
        self.page = page
        self.segment = segment
        self.pick = pick
        self.back = back
        self.run = run
        self.open = open
    }

    var summary: Section? { page.section("summary") }

    // value: the facts of the page's summary.
    public static func facts(_ page: Page) -> Value {
        var line = Value(kind: "list")
        line.items = page.section("summary")?.caption ?? []
        return line
    }

    // value: the H1 of a subpage: the title the engine sent in its header.
    public static func title(_ page: Page) -> String { page.header?.title ?? "" }

    // value: the page's mark: a symbol.
    static func image(_ glyph: Glyph) -> Image? {
        guard case let .symbol(name) = GlyphView.drawn(glyph) else { return nil }
        return Image(systemName: name)
    }

    public var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.group, kit: "MachineHeader") }

    @ViewBuilder var drawing: some View {
        let m = Token.Metric.PageHeader.self
        VStack(alignment: .leading, spacing: 0) {
            if let words = page.header?.back {
                Button(action: back) {
                    HStack(spacing: m.backGap) {
                        Image(systemName: Token.Symbol.back).font(.system(size: m.backChevron, weight: .semibold))
                        Text(words).recorded(words, Token.Role.back)
                    }
                    .foregroundStyle(tokens.tone(Token.Role.back.tone))
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .padding(.bottom, m.backBottom)
            }
            HStack(spacing: m.titleGap) {
                if let glyph = page.header?.glyph, let image = Self.image(glyph) {
                    image.font(.system(size: m.glyph, weight: .regular))
                        .foregroundStyle(glyph.signal.map { $0 == "active" ? tokens.color.t2 : tokens.signal($0) } ?? tokens.color.t2)
                        .frame(width: m.glyphSlot)
                }
                Text(Self.title(page)).font(Token.Role.pageTitle.font).tracking(Token.Role.pageTitle.tracking).foregroundStyle(tokens.tone(Token.Role.pageTitle.tone)).lineLimit(1)
                Spacer(minLength: 0)
                HStack(spacing: m.buttonsGap) {
                    ForEach(Array((page.header?.buttons ?? []).compactMap { page.action($0) }.enumerated()), id: \.offset) { _, a in
                        if a.icon == Token.Symbol.menu {
                            menu(a)
                        } else {
                            PanelButton(a.label ?? "", symbol: a.icon) { run(a) }
                        }
                    }
                }
            }
            .frame(height: m.title)
            FactsLine(Self.facts(page), open: open).padding(.top, m.factsTop)
            if let segments = page.segments, !segments.isEmpty {
                Segmented(options: segmented(segments), keys: segments.map(\.key), leaving: caught?.segment.flatMap { from in segments.firstIndex { $0.id == from } })
                .padding(.top, m.segmentsTop)
            }
        }
    }

    // value: the page's menu: its submenu's verbs on a click, each run as
    // its own; drawn still as the round button it opens from.
    @ViewBuilder
    func menu(_ a: Action) -> some View {
        let model = MenuModel.page(page)
        if still || model.isEmpty {
            IconButton(Token.Symbol.menu) { run(a) }.accessibilityLabel(a.label ?? "").menuAnchor(MenuAnchors.page)
        } else {
            Image(systemName: Token.Symbol.menu).font(.system(size: Token.Metric.Control.iconGlyph, weight: .regular))
                .foregroundStyle(tokens.color.t2)
                .frame(width: Token.Metric.Control.icon, height: Token.Metric.Control.icon)
                .contentShape(Circle())
                .systemMenu(model) { menuPick?($0, nav: nil) }
                .accessibilityElement()
                .accessibilityAddTraits(.isButton)
                .accessibilityLabel(a.label ?? "")
            .menuAnchor(MenuAnchors.page)
        }
    }

    func segmented(_ segments: [Segment]) -> [(label: String, on: Bool, tap: () -> Void)] {
        segments.map { s in (label: s.label ?? "", on: s.id == segment, tap: { pick(s.id ?? "") }) }
    }
}
