import SwiftUI

// invariant: SPEC A5: the drawing of a menu the system draws natively,
// for the Catalog and the goldens: the menu fill, 5 pt inside, 24 pt
// items with a 16 pt symbol slot and the key on the right, the header in
// small grey words, red what is destructive, a ✓ what is picked, a › what
// opens a submenu; the entry whose submenu is open is lit.
public struct MenuPreview: View {
    let model: MenuModel
    let open: String?
    @Environment(\.stationTokens) private var tokens

    public init(_ model: MenuModel, open: String? = nil) {
        self.model = model
        self.open = open
    }

    typealias M = Token.Metric.Menu
    typealias P = Token.MenuPreview

    // value: the height the menu takes, from its entries alone, so it is
    // placed before it is drawn.
    public static func height(_ model: MenuModel) -> CGFloat {
        let items = model.items.reduce(CGFloat(0)) { $0 + ($1.separator ? P.line : M.item) }
        return items + (model.header == nil ? 0 : P.header) + 2 * M.padding
    }

    // value: how far down the menu an entry's top sits.
    public static func top(of id: String, in model: MenuModel) -> CGFloat {
        var y = M.padding + (model.header == nil ? 0 : P.header)
        for item in model.items {
            if item.id == id { return y }
            y += item.separator ? P.line : M.item
        }
        return y
    }

    var checks: Bool { model.items.contains(where: \.checked) }
    var icons: Bool { model.items.contains { $0.icon != nil } }

    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if let header = model.header {
                Text(verbatim: header).font(Token.Role.findGroup.font).foregroundStyle(tokens.color.t3)
                    .lineLimit(1)
                    .padding(.horizontal, P.side)
                    .frame(height: P.header, alignment: .leading)
            }
            ForEach(Array(model.items.enumerated()), id: \.offset) { _, item in
                if item.separator {
                    Rectangle().fill(tokens.color.hair).frame(height: Token.Metric.Row.hairline)
                        .padding(.horizontal, P.side)
                        .frame(height: P.line)
                } else {
                    entry(item)
                }
            }
        }
        .padding(M.padding)
        .frame(minWidth: M.minWidth, alignment: .leading)
        .fixedSize()
        .background(tokens.color.menu, in: RoundedRectangle(cornerRadius: M.radius, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: M.radius, style: .continuous).strokeBorder(tokens.color.panelEdge, lineWidth: P.ring))
        .shadow(color: P.shadow.color, radius: P.shadow.radius, y: P.shadow.y)
        .accessibilityElement(children: .combine)
    }

    func entry(_ item: MenuItem) -> some View {
        let lit = item.id == open
        let ink: Color = lit ? P.litInk : item.destructive ? tokens.color.redText : item.enabled ? tokens.color.t1 : tokens.color.t3
        return HStack(spacing: P.gap) {
            if checks {
                Image(systemName: Token.Symbol.checkmark).font(.system(size: P.checkGlyph, weight: .semibold))
                    .opacity(item.checked ? 1 : 0)
                    .frame(width: P.check)
            }
            if icons {
                SwiftUI.Group {
                    if let icon = item.icon { Image(systemName: icon).font(.system(size: P.check, weight: .regular)) }
                }
                .foregroundStyle(lit ? P.litInk : item.destructive ? tokens.color.redText : tokens.color.t2)
                .frame(width: M.icon)
            }
            Text(verbatim: item.title).font(Token.Role.findResult.font).foregroundStyle(ink).lineLimit(1)
            Spacer(minLength: P.trailingRoom)
            if !item.submenu.isEmpty {
                Image(systemName: Token.Symbol.submenu).font(.system(size: P.chevron, weight: .semibold))
                    .foregroundStyle(lit ? P.litInk : tokens.color.t3)
            } else if let key = item.key {
                Text(verbatim: MenuModel.keys(key)).font(Token.Role.keyCap.font).foregroundStyle(lit ? P.litInk : tokens.color.t3)
            } else if let note = item.note {
                Text(verbatim: note).font(Token.Role.findResult.font).foregroundStyle(lit ? P.litInk : tokens.color.t3).lineLimit(1)
            }
        }
        .padding(.horizontal, P.side)
        .frame(height: M.item)
        .background(lit ? tokens.color.accent : .clear, in: RoundedRectangle(cornerRadius: M.itemRadius, style: .continuous))
    }
}

// value: where a menu opens, as a story names it: a row's or a tile's
// right-click, or the More button of its page.
public struct MenuAnchors: PreferenceKey {
    public static let defaultValue: [String: Anchor<CGRect>] = [:]
    public static func reduce(value: inout [String: Anchor<CGRect>], nextValue: () -> [String: Anchor<CGRect>]) {
        value.merge(nextValue()) { $1 }
    }

    public static let page = "page"
    public static func row(_ id: String) -> String { "row:" + id }
}

extension View {
    // value: where a menu opens from this view, for its preview.
    public func menuAnchor(_ key: String) -> some View {
        anchorPreference(key: MenuAnchors.self, value: .bounds) { [key: $0] }
    }
}

// invariant: a menu's preview opens where the system would open it: a
// right-click's at the row's start, just under its middle; a More menu's
// under its button, its right edge on the button's; a submenu beside its
// entry, on the left when the right has no room; all kept inside the
// panel.
public struct MenuStage: View {
    let model: MenuModel
    let child: String?
    let anchor: CGRect
    let page: Bool
    let size: CGSize
    // value: a sheet's popup opens under its field, its left edge on the
    // field's.
    let popup: Bool

    public init(_ model: MenuModel, child: String? = nil, anchor: CGRect, page: Bool, popup: Bool = false, in size: CGSize) {
        self.model = model
        self.child = child
        self.anchor = anchor
        self.page = page
        self.popup = popup
        self.size = size
    }

    var sub: MenuItem? { model.items.first { $0.id == child && !$0.submenu.isEmpty } }

    public var body: some View {
        MenuPlacement(x: popup ? anchor.minX : page ? anchor.maxX : anchor.minX + Token.MenuPreview.inset,
                      y: page || popup ? anchor.maxY + Token.MenuPreview.inset : anchor.minY + Token.MenuPreview.pointer,
                      trailing: page && !popup, entry: sub.map { MenuPreview.top(of: $0.id, in: model) - Token.Metric.Menu.padding } ?? 0) {
            MenuPreview(model, open: child)
            if let sub { MenuPreview(MenuModel(header: nil, items: sub.submenu)) }
        }
        .frame(width: size.width, height: size.height)
        .allowsHitTesting(false)
    }
}

// value: the place of a menu and of its open submenu, from their own
// sizes: the menu at its point (its right edge there when trailing), the
// submenu beside the entry it opens from, both inside the bounds.
struct MenuPlacement: SwiftUI.Layout {
    let x: CGFloat
    let y: CGFloat
    let trailing: Bool
    let entry: CGFloat

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        proposal.replacingUnspecifiedDimensions()
    }

    static func clamp(_ v: CGFloat, _ length: CGFloat, within total: CGFloat) -> CGFloat {
        max(Token.MenuPreview.inset, min(v, total - length - Token.MenuPreview.inset))
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        guard let menu = subviews.first else { return }
        let m = menu.sizeThatFits(.unspecified)
        let mx = Self.clamp(trailing ? x - m.width : x, m.width, within: bounds.width)
        let my = Self.clamp(y, m.height, within: bounds.height)
        menu.place(at: CGPoint(x: bounds.minX + mx, y: bounds.minY + my), proposal: .unspecified)
        guard subviews.count > 1 else { return }
        let child = subviews[1]
        let c = child.sizeThatFits(.unspecified)
        let right = mx + m.width - Token.MenuPreview.overlap
        let cx = Self.clamp(right, c.width, within: bounds.width)
        let cy = Self.clamp(my + entry, c.height, within: bounds.height)
        child.place(at: CGPoint(x: bounds.minX + cx, y: bounds.minY + cy), proposal: .unspecified)
    }
}
