import AppKit
import SwiftUI

public enum PanelHeight {
    public static let sheetAnchor = "station.sheet"

    public static func cap(visible: CGFloat?) -> CGFloat {
        guard let visible else { return Token.Metric.Panel.cap }
        return min(Token.Metric.Panel.cap, visible - Token.Metric.Panel.screenRoom)
    }

    public static func room(cap: CGFloat, header: CGFloat, footer: CGFloat) -> CGFloat {
        max(cap - header - footer, 0)
    }

    public static func body(_ content: CGFloat, room: CGFloat) -> CGFloat { min(content, room) }

    public static func scrolls(_ content: CGFloat, room: CGFloat) -> Bool { content > room }

    @MainActor
    static var screen: CGFloat? { NSScreen.main?.visibleFrame.height }
}

public struct PageStack<Element: Equatable>: Equatable {
    public private(set) var pages: [Element]

    public init(_ root: Element) { pages = [root] }

    public var top: Element { pages.last ?? pages[0] }
    public var depth: Int { pages.count }
    public var root: Element { pages[0] }

    public mutating func push(_ page: Element) {
        if pages.count == Token.Pane.depth {
            pages[pages.count - 1] = page
        } else {
            pages.append(page)
        }
    }

    public mutating func pop() {
        if pages.count > 1 { pages.removeLast() }
    }

    public mutating func home() { pages = [pages[0]] }
}

public struct PanelLayers {
    public var find: AnyView?
    public var sheet: AnyView?
    public var alert: AnyView?

    public init(find: AnyView? = nil, sheet: AnyView? = nil, alert: AnyView? = nil) {
        self.find = find
        self.sheet = sheet
        self.alert = alert
    }
}

public enum PanelInsets: Sendable {
    case page
    case flush
}

public struct Panel<Header: View, Content: View, Footer: View>: View {
    let insets: PanelInsets
    let bounded: Bool
    let key: String
    let sheet: String?
    let layers: PanelLayers
    let header: Header
    let content: Content
    let footer: Footer

    @Environment(\.stationStill) private var still
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationPanelWidth) private var width
    @Environment(\.stationPanelRoom) private var room
    @Environment(\.stationCaught) private var caught

    @State private var measured = Measured()
    @State private var shown: CGFloat = 0
    @State private var edges = Edges()
    @State private var offset: CGFloat = 0
    @State private var position = ScrollPosition(edge: .top)
    @State private var opening = false

    struct Measured: Equatable {
        var header: CGFloat = 0
        var content: CGFloat = 0
        var footer: CGFloat = 0
    }

    struct Edges: Equatable {
        var top = false
        var bottom = false
    }

    public init(insets: PanelInsets = .page, bounded: Bool = true, key: String = "", sheet: String? = nil, layers: PanelLayers = PanelLayers(),
                @ViewBuilder header: () -> Header, @ViewBuilder content: () -> Content, @ViewBuilder footer: () -> Footer) {
        self.insets = insets
        self.bounded = bounded
        self.key = key
        self.sheet = sheet
        self.layers = layers
        self.header = header()
        self.content = content()
        self.footer = footer()
    }

    var cap: CGFloat { room ?? PanelHeight.cap(visible: still ? nil : PanelHeight.screen) }

    var bodyRoom: CGFloat { PanelHeight.room(cap: cap, header: measured.header, footer: measured.footer) }

    var target: CGFloat { PanelHeight.body(measured.content, room: bodyRoom) }

    var spring: Animation { tokens.reduceMotion ? Token.Motion.reducedHeight : Token.Motion.smooth }

    public var body: some View {
        PanelColumn(cap: bounded ? cap : nil) {
            padded(header, insets == .page ? Token.Metric.Panel.header : nil)
                .onGeometryChange(for: CGFloat.self) { $0.size.height } action: { measured.header = $0 }
            pageBody
            padded(footer, insets == .page ? Token.Metric.Panel.footer : nil)
                .onGeometryChange(for: CGFloat.self) { $0.size.height } action: { measured.footer = $0 }
        }
        .frame(width: width, alignment: .topLeading)
        .overlayPreferenceValue(PopoverPins.self) { PopoverLayer(pins: $0) }
        .overlay(alignment: .top) { layered }
        .modifier(PanelPane(bare: caught?.bare == true))
    }

    func padded<V: View>(_ view: V, _ edges: EdgeInsets?) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            if let edges, V.self != EmptyView.self {
                view.padding(edges).frame(maxWidth: .infinity, alignment: .leading)
            } else {
                view
            }
        }
    }

    var page: some View {
        padded(content, insets == .page ? Token.Metric.Panel.body : nil)
    }

    var pageBody: some View {
        VStack(spacing: 0) { slot }
    }

    @ViewBuilder
    var slot: some View {
        if !bounded {
            page
        } else if still {
            StillCut { page }
                .clipped()
                .mask {
                    CutFade {
                        page.hidden()
                        Rectangle().fill(Token.Pane.opaque)
                        LinearGradient(colors: [Token.Pane.opaque, Color.clear], startPoint: .top, endPoint: .bottom)
                    }
                }
        } else {
            scrolled
        }
    }

    var scrolled: some View {
        ScrollViewReader { proxy in
            ScrollView(.vertical) {
                page.onGeometryChange(for: CGFloat.self) { $0.size.height } action: { measured.content = $0 }
                    .background(OverlayScroller())
            }
            .scrollPosition($position)
            .onScrollGeometryChange(for: Edges.self) { g in
                Edges(top: g.contentOffset.y > Token.Measure.grid,
                      bottom: g.contentOffset.y + g.containerSize.height < g.contentSize.height - Token.Measure.grid)
            } action: { _, now in edges = now }
            .onScrollGeometryChange(for: CGFloat.self) { $0.contentOffset.y } action: { _, y in offset = y }
            .onChange(of: sheet) { _, open in
                opening = open != nil
                guard open != nil else { return }
                DispatchQueue.main.async { withAnimation(spring) { proxy.scrollTo(PanelHeight.sheetAnchor, anchor: .bottom) } }
            }
            .onChange(of: measured.content) {
                guard opening else { return }
                opening = false
                DispatchQueue.main.async { withAnimation(spring) { proxy.scrollTo(PanelHeight.sheetAnchor, anchor: .bottom) } }
            }
        }
        .id(key)
        .onChange(of: key) {
            offset = 0
            position.scrollTo(edge: .top)
        }
        .onChange(of: measured.footer) { was, now in
            guard was > 0, now > was, PanelHeight.scrolls(measured.content, room: bodyRoom) else { return }
            position.scrollTo(y: offset + now - was)
        }
        .onChange(of: target, initial: true) { _, now in
            if shown == 0 {
                shown = now
            } else {
                withAnimation(spring) { shown = now }
            }
        }
        .scrollBounceBehavior(.basedOnSize)
        .scrollIndicators(.never)
        .mask { EdgeFade(top: edges.top, bottom: edges.bottom) }
        .frame(height: shown > 0 ? shown : nil)
    }

    @ViewBuilder
    var layered: some View {
        let shape = RoundedRectangle(cornerRadius: Token.Glass.window, style: .continuous)
        ZStack(alignment: .top) {
            if let find = layers.find {
                shape.fill(tokens.color.scrimFind)
                find
            }
            if let sheet = layers.sheet {
                arriving(shape.fill(tokens.color.scrim), sheet) { SheetReveal(progress: SheetReveal.frozen($0, reduce: $1)) }
            }
            if let alert = layers.alert {
                arriving(shape.fill(tokens.color.scrim), alert) { PopAppear(progress: PopAppear.frozen($0, kind: .alert, reduce: $1), kind: .alert) }
            }
        }
    }

    // invariant: a layer caught arriving draws its scrim and its card at
    // their moment; at rest both are drawn as they stand.
    @ViewBuilder
    func arriving<Scrim: View, M: ViewModifier>(_ scrim: Scrim, _ card: AnyView, _ move: (Double, Bool) -> M) -> some View {
        if let caught, caught.layer {
            scrim.modifier(ScrimFade(progress: SheetReveal.frozen(caught.time, reduce: tokens.reduceMotion)))
            card.modifier(move(caught.time, tokens.reduceMotion))
        } else {
            scrim
            card
        }
    }
}

extension Panel where Header == EmptyView, Footer == EmptyView {
    public init(insets: PanelInsets = .page, bounded: Bool = true, key: String = "", sheet: String? = nil, layers: PanelLayers = PanelLayers(),
                @ViewBuilder content: () -> Content) {
        self.init(insets: insets, bounded: bounded, key: key, sheet: sheet, layers: layers, header: { EmptyView() }, content: content, footer: { EmptyView() })
    }
}

extension Panel where Header == EmptyView {
    public init(insets: PanelInsets = .page, bounded: Bool = true, key: String = "", sheet: String? = nil, layers: PanelLayers = PanelLayers(),
                @ViewBuilder content: () -> Content, @ViewBuilder footer: () -> Footer) {
        self.init(insets: insets, bounded: bounded, key: key, sheet: sheet, layers: layers, header: { EmptyView() }, content: content, footer: footer)
    }
}

struct PanelColumn: SwiftUI.Layout {
    let cap: CGFloat?

    func heights(_ width: CGFloat?, _ subviews: Subviews) -> [CGFloat] {
        guard subviews.count == 3 else { return subviews.map { $0.sizeThatFits(ProposedViewSize(width: width, height: nil)).height } }
        let free = ProposedViewSize(width: width, height: nil)
        let header = subviews[0].sizeThatFits(free).height
        let footer = subviews[2].sizeThatFits(free).height
        let room = cap.map { PanelHeight.room(cap: $0, header: header, footer: footer) }
        let content = subviews[1].sizeThatFits(ProposedViewSize(width: width, height: room)).height
        return [header, room.map { PanelHeight.body(content, room: $0) } ?? content, footer]
    }

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? subviews.map { $0.sizeThatFits(.unspecified).width }.max() ?? 0
        return CGSize(width: width, height: heights(width, subviews).reduce(0, +))
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var y = bounds.minY
        for (view, height) in zip(subviews, heights(bounds.width, subviews)) {
            view.place(at: CGPoint(x: bounds.minX, y: y), anchor: .topLeading, proposal: ProposedViewSize(width: bounds.width, height: height))
            y += height
        }
    }
}

struct StillCut: SwiftUI.Layout {
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let ideal = subviews.first?.sizeThatFits(ProposedViewSize(width: proposal.width, height: nil)) ?? .zero
        return CGSize(width: proposal.width ?? ideal.width, height: min(ideal.height, proposal.height ?? ideal.height))
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        guard let content = subviews.first else { return }
        let ideal = content.sizeThatFits(ProposedViewSize(width: bounds.width, height: nil))
        content.place(at: bounds.origin, anchor: .topLeading, proposal: ProposedViewSize(width: bounds.width, height: ideal.height))
    }
}

struct CutFade: SwiftUI.Layout {
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        proposal.replacingUnspecifiedDimensions()
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        guard subviews.count == 3 else { return }
        let content = subviews[0].sizeThatFits(ProposedViewSize(width: bounds.width, height: nil)).height
        let fade = PanelHeight.scrolls(content, room: bounds.height + Token.Measure.grid) ? min(Token.Pane.fade, bounds.height) : 0
        subviews[0].place(at: bounds.origin, anchor: .topLeading, proposal: ProposedViewSize(width: bounds.width, height: content))
        subviews[1].place(at: bounds.origin, anchor: .topLeading, proposal: ProposedViewSize(width: bounds.width, height: bounds.height - fade))
        subviews[2].place(at: CGPoint(x: bounds.minX, y: bounds.maxY - fade), anchor: .topLeading, proposal: ProposedViewSize(width: bounds.width, height: fade))
    }
}

struct EdgeFade: View {
    let top: Bool
    let bottom: Bool

    var body: some View {
        VStack(spacing: 0) {
            if top { LinearGradient(colors: [Color.clear, Token.Pane.opaque], startPoint: .top, endPoint: .bottom).frame(height: Token.Pane.fade) }
            Rectangle().fill(Token.Pane.opaque)
            if bottom { LinearGradient(colors: [Token.Pane.opaque, Color.clear], startPoint: .top, endPoint: .bottom).frame(height: Token.Pane.fade) }
        }
    }
}

// invariant: live the pane paints one fill, the plate under the content,
// clipped to the window's shape and edged by nothing: MenuBarExtra's window
// is already the system glass and its rim, so a rim or a second tint over it
// doubles the background. Drawn still, the shadows, outline and rim stand in
// for that glass under the same plate.
struct PanelPane: ViewModifier {
    var bare = false
    @Environment(\.stationStill) private var still
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        let shape = RoundedRectangle(cornerRadius: Token.Glass.window, style: .continuous)
        if bare {
            content.containerShape(shape)
        } else if !still {
            content.containerShape(shape).background(shape.fill(tokens.plate))
        } else {
            pane(content)
        }
    }

    func pane(_ content: Content) -> some View {
        let shape = RoundedRectangle(cornerRadius: Token.Glass.window, style: .continuous)
        let scheme = tokens.scheme
        return content
            .containerShape(shape)
            .background {
                shape.fill(tokens.plate)
                .background {
                    ZStack {
                        ForEach(Array(Token.Pane.shades(scheme).enumerated()), id: \.offset) { _, shade in
                            shape.fill(Token.Pane.opaque).shadow(color: shade.color, radius: shade.radius, y: shade.y)
                        }
                    }
                    .mask { Rectangle().padding(-Token.Metric.Panel.cap).overlay { shape.blendMode(.destinationOut) }.compositingGroup() }
                    RoundedRectangle(cornerRadius: Token.Pane.outlineRadius, style: .continuous)
                        .strokeBorder(Token.Pane.ring(scheme), lineWidth: Token.Pane.outline)
                        .padding(-Token.Pane.outline)
                }
            }
            .overlay {
                if tokens.solid {
                    shape.strokeBorder(tokens.color.hair, lineWidth: Token.Pane.solidEdge)
                } else {
                    shape.strokeBorder(LinearGradient(stops: Token.Pane.rim(scheme), startPoint: .top, endPoint: .bottom), lineWidth: Token.Pane.rimWidth)
                }
            }
    }
}

// invariant: the panel's scroll view keeps no scroller room: with a mouse
// plugged in macOS lays its scroller beside the content and narrows the
// visible width under a body laid out at the panel's full width, so its
// trailing edge is cut; its scrollers float over the content, and stay
// hidden as the panel draws none.
struct OverlayScroller: NSViewRepresentable {
    final class Floater: NSView {
        var watching: NSObjectProtocol?

        override func viewDidMoveToWindow() {
            super.viewDidMoveToWindow()
            float()
            guard watching == nil else { return }
            watching = NotificationCenter.default.addObserver(forName: NSScroller.preferredScrollerStyleDidChangeNotification, object: nil, queue: .main) { [weak self] _ in
                MainActor.assumeIsolated { self?.float() }
            }
        }

        func float() {
            guard let scroll = enclosingScrollView else { return }
            scroll.scrollerStyle = .overlay
            scroll.hasVerticalScroller = false
            scroll.hasHorizontalScroller = false
        }

        deinit {
            if let watching { NotificationCenter.default.removeObserver(watching) }
        }
    }

    func makeNSView(context: NSViewRepresentableContext<OverlayScroller>) -> Floater { Floater() }

    func updateNSView(_ view: Floater, context: NSViewRepresentableContext<OverlayScroller>) { view.float() }
}
