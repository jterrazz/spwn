import SwiftUI

// value: the movements of SPEC A8 the transition stories catch, each named
// as its story is: `transition.<movement>`, or `transition.reduce-motion.
// <movement>` for its Reduce Motion form.
public enum StoryMotion: String, CaseIterable, Sendable {
    case panelOpen = "panel-open"
    case panelClose = "panel-close"
    case push, pop, jump, segment, sheet, alert, find, menu
    case heroMorph = "hero-morph"
    case needCollapse = "need-collapse"
    case rowLeave = "row-leave"

    public static let reduced = "reduce-motion."

    public static func of(_ id: String) -> (motion: StoryMotion, reduce: Bool)? {
        let head = StoryGroup.transition.rawValue + "."
        guard id.hasPrefix(head) else { return nil }
        var name = String(id.dropFirst(head.count))
        let reduce = name.hasPrefix(reduced)
        if reduce { name.removeFirst(reduced.count) }
        return StoryMotion(rawValue: name).map { ($0, reduce) }
    }

    // value: a movement that slides a page in opaque (SPEC A8 M2, M3):
    // mid-flight its pane is not glass.
    public var slides: Bool { self == .push || self == .pop }

    // value: its run, as the live app animates it.
    public var duration: Double {
        switch self {
        case .panelClose, .menu: Token.Motion.snappyDuration
        case .jump: Token.Motion.jumpInDuration
        default: Token.Motion.smoothDuration
        }
    }
}

// invariant: SPEC C4: a transition story is drawn at one moment of its
// movement through the same modifiers the live app animates: the pages it
// moves between are its own and its `play.from` story's, each drawn as its
// story draws it at rest. Find takes the page's place at once: the page it
// leaves only gives the height its way starts from.
struct TransitionView: View {
    let story: Story
    let motion: StoryMotion
    let time: Double
    let handlers: PopoverHandlers
    @Environment(\.stationTokens) private var tokens

    var reduce: Bool { tokens.reduceMotion }
    var from: Story { story.from ?? story }

    func rest(_ s: Story, _ caught: Caught? = nil) -> some View {
        StoryView(s, handlers: handlers, rest: true).environment(\.stationCaught, caught)
    }

    var body: some View {
        switch motion {
        case .panelOpen:
            rest(story).modifier(PanelAppear(progress: PanelAppear.frozen(time, closing: false, reduce: reduce), anchor: .top))
        case .panelClose:
            rest(story).modifier(PanelAppear(progress: PanelAppear.frozen(time, closing: true, reduce: reduce), closing: true, anchor: .top))
        case .push:
            pair(under: from, top: story, backward: false,
                 PushTransition(progress: PushTransition.frozen(time, role: .under, way: .push, reduce: reduce), role: .under, way: .push),
                 PushTransition(progress: PushTransition.frozen(time, role: .top, way: .push, reduce: reduce), role: .top, way: .push))
        case .pop:
            pair(under: story, top: from, backward: true,
                 PushTransition(progress: PushTransition.frozen(time, role: .under, way: .pop, reduce: reduce), role: .under, way: .pop),
                 PushTransition(progress: PushTransition.frozen(time, role: .top, way: .pop, reduce: reduce), role: .top, way: .pop))
        case .jump:
            pair(under: from, top: story, backward: false,
                 JumpTransition(progress: JumpTransition.frozen(time, role: .leaving), role: .leaving),
                 JumpTransition(progress: JumpTransition.frozen(time, role: .arriving), role: .arriving))
        case .find:
            pair(under: from, top: story, backward: false, layer: true,
                 ScrimFade(progress: 0),
                 FindAppear(progress: FindAppear.frozen(time, field: false, reduce: reduce)))
        case .segment:
            rest(story, Caught(time: time, segment: from.ui.segment ?? from.pages.first.map(MacPage.picked)))
        case .sheet, .alert, .menu:
            rest(story, Caught(time: time, layer: true))
        case .heroMorph:
            rest(story, Caught(time: time, hero: true))
        case .needCollapse, .rowLeave:
            rest(story, Caught(time: time, row: story.ui.leaving))
        }
    }

    // invariant: two pages moving in one pane: each drawn bare, the one
    // under first, the pane's height on its way from the page left to the
    // page shown.
    func pair<Under: ViewModifier, Top: ViewModifier>(under: Story, top: Story, backward: Bool, layer: Bool = false, _ u: Under, _ t: Top) -> some View {
        let shape = RoundedRectangle(cornerRadius: Token.Glass.window, style: .continuous)
        return Morph(share: HeightMorph.frozen(time, reduce: reduce), backward: backward) {
            rest(under, Caught(time: time, bare: true)).modifier(u)
            rest(top, Caught(time: time, layer: layer, bare: true)).modifier(t)
        }
        .clipShape(shape)
        .modifier(PanelPane())
    }
}
