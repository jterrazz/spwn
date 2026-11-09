import SwiftUI

// value: a movement caught at one moment of its run, as a story freezes it
// (SPEC C4): the part of the page it moves, which reads it and draws itself
// at that moment through its own modifier. Nil is the page at rest.
public struct Caught: Equatable, Sendable {
    public var time: Double
    // value: the row leaving its list, by its id.
    public var row: String?
    public var hero = false
    // value: the panel's top layer arriving: its sheet, alert, Find field
    // or menu.
    public var layer = false
    // value: the segment the page leaves for the one it shows.
    public var segment: String?
    // value: the page drawn without its pane, inside a pane the movement
    // draws once around two pages.
    public var bare = false

    public init(time: Double, row: String? = nil, hero: Bool = false, layer: Bool = false, segment: String? = nil, bare: Bool = false) {
        self.time = time
        self.row = row
        self.hero = hero
        self.layer = layer
        self.segment = segment
        self.bare = bare
    }
}

extension EnvironmentValues {
    @Entry public var stationCaught: Caught? = nil
}

// invariant: a row leaves by its own id: the row the moment names
// collapses, slides and fades; every other row is drawn at rest.
struct CaughtRow: ViewModifier {
    let id: String?
    @Environment(\.stationCaught) private var caught
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        if let caught, let id, caught.row == id {
            content.modifier(RowLeave(progress: RowLeave.frozen(caught.time, reduce: tokens.reduceMotion)))
        } else {
            content
        }
    }
}

// invariant: the hero's parts morph only while a moment names the hero.
struct CaughtHero: ViewModifier {
    let part: HeroMorph.Part
    @Environment(\.stationCaught) private var caught
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        if let caught, caught.hero {
            content.modifier(HeroMorph(progress: HeroMorph.frozen(caught.time, part: part, reduce: tokens.reduceMotion), part: part))
        } else {
            content
        }
    }
}

// invariant: Find's field settles from above while Find is caught
// arriving.
struct CaughtField: ViewModifier {
    @Environment(\.stationCaught) private var caught
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        if let caught, caught.layer {
            content.modifier(FindAppear(progress: FindAppear.frozen(caught.time, field: true, reduce: tokens.reduceMotion), field: true))
        } else {
            content
        }
    }
}

// invariant: a menu's preview grows from where it opens while it is
// caught arriving.
struct CaughtMenu: ViewModifier {
    let anchor: UnitPoint
    @Environment(\.stationCaught) private var caught
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        if let caught, caught.layer {
            content.modifier(PopAppear(progress: PopAppear.frozen(caught.time, kind: .menu, reduce: tokens.reduceMotion), kind: .menu, anchor: anchor))
        } else {
            content
        }
    }
}

extension View {
    func caughtRow(_ id: String?) -> some View { modifier(CaughtRow(id: id)) }

    func caughtHero(_ part: HeroMorph.Part) -> some View { modifier(CaughtHero(part: part)) }
}

// value: M6 between two contents: as tall as its share of the way from
// one's height to the other's, each laid out at its own height
// from the top.
struct Morph: SwiftUI.Layout {
    var share: Double
    // value: backward, the way runs from the second content to the first.
    var backward = false

    var animatableData: Double {
        get { share }
        set { share = newValue }
    }

    func heights(_ width: CGFloat?, _ subviews: Subviews) -> [CGFloat] {
        subviews.map { $0.sizeThatFits(ProposedViewSize(width: width, height: nil)).height }
    }

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? subviews.map { $0.sizeThatFits(.unspecified).width }.max() ?? 0
        let h = heights(width, subviews)
        guard let first = h.first, let last = h.last else { return CGSize(width: width, height: 0) }
        return CGSize(width: width, height: HeightMorph.height(share, from: backward ? last : first, to: backward ? first : last))
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        for (view, height) in zip(subviews, heights(bounds.width, subviews)) {
            view.place(at: bounds.origin, anchor: .topLeading, proposal: ProposedViewSize(width: bounds.width, height: height))
        }
    }
}
