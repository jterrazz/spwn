import SwiftUI

private struct FocusLitKey: EnvironmentKey {
    static let defaultValue: PanelFocus? = nil
}

private struct FocusBindingKey: EnvironmentKey {
    static let defaultValue: FocusState<PanelFocus?>.Binding? = nil
}

private struct PointerHeldKey: EnvironmentKey {
    static let defaultValue = false
}

extension EnvironmentValues {
    public var stationFocus: PanelFocus? {
        get { self[FocusLitKey.self] }
        set { self[FocusLitKey.self] = newValue }
    }

    var stationFocusBinding: FocusState<PanelFocus?>.Binding? {
        get { self[FocusBindingKey.self] }
        set { self[FocusBindingKey.self] = newValue }
    }

    public var stationPointerHeld: Bool {
        get { self[PointerHeldKey.self] }
        set { self[PointerHeldKey.self] = newValue }
    }
}

public enum PointerRule {
    public static func releases(from: CGPoint, to: CGPoint) -> Bool {
        hypot(to.x - from.x, to.y - from.y) >= Token.Focus.pointer
    }
}

public enum FocusLook: Sendable { case row, control }

struct PanelFocusable: ViewModifier {
    let focus: PanelFocus
    let look: FocusLook
    let radius: CGFloat
    @Environment(\.stationFocus) private var lit
    @Environment(\.stationFocusBinding) private var binding
    @Environment(\.stationTokens) private var tokens

    func body(content: Content) -> some View {
        let on = lit == focus
        let ring = Token.Focus.ring
        bound(content)
            .overlay {
                if on, look == .row {
                    RoundedRectangle(cornerRadius: radius, style: .continuous)
                        .strokeBorder(tokens.color.accent.opacity(Token.Focus.rowShare), lineWidth: ring)
                        .allowsHitTesting(false)
                }
            }
            .overlay {
                if on, look == .control {
                    RoundedRectangle(cornerRadius: radius + ring, style: .continuous)
                        .strokeBorder(tokens.color.accent.opacity(Token.Focus.controlShare), lineWidth: ring)
                        .padding(-ring)
                        .allowsHitTesting(false)
                }
            }
    }

    @ViewBuilder
    func bound(_ content: Content) -> some View {
        if let binding {
            content.focusable(interactions: .edit).focusEffectDisabled().focused(binding, equals: focus)
        } else {
            content
        }
    }
}

extension View {
    public func panelFocus(_ focus: PanelFocus, look: FocusLook, radius: CGFloat) -> some View {
        modifier(PanelFocusable(focus: focus, look: look, radius: radius))
    }
}

struct PanelKeyboard: ViewModifier {
    let state: PanelState
    let handlers: PopoverHandlers
    @Environment(\.stationStill) private var still
    @FocusState private var focus: PanelFocus?
    @State private var held = false
    @State private var from: CGPoint?

    static let backTab = KeyEquivalent("\u{19}")

    var walk: [PanelFocus] { PanelOrder.walk(state) }
    var spot: String { "\(state.shown?.id ?? "")/\(state.shown?.context?.machine ?? "")/\(state.find != nil)/\(state.confirming ?? "")" }

    func body(content: Content) -> some View {
        if still {
            content
        } else {
            content
                .environment(\.stationFocus, focus?.lit)
                .environment(\.stationFocusBinding, $focus)
                .environment(\.stationPointerHeld, held)
                .background { anchor }
                .background { shortcuts }
                .onKeyPress(phases: .down) { press in
                    guard let key = Self.key(press) else { return .ignored }
                    return perform(key) ? .handled : .ignored
                }
                .simultaneousGesture(TapGesture().onEnded { if focus?.lit != nil { focus = .panel } })
                .onContinuousHover(coordinateSpace: .local) { phase in
                    guard held, case let .active(at) = phase else { return }
                    if let from {
                        if PointerRule.releases(from: from, to: at) { held = false }
                    } else {
                        from = at
                    }
                }
                .onChange(of: walk) { _, order in
                    if focus?.lit != nil, PanelOrder.follow(focus, in: order) == nil { focus = .panel }
                }
                .onChange(of: spot) {
                    held = true
                    from = nil
                }
        }
    }

    var anchor: some View {
        Color.clear.frame(width: Token.Detail.anchor, height: Token.Detail.anchor)
            .focusable(interactions: .edit)
            .focusEffectDisabled()
            .focused($focus, equals: .panel)
            .onAppear { DispatchQueue.main.async { if focus == nil { focus = .panel } } }
            .accessibilityHidden(true)
    }

    @ViewBuilder
    var shortcuts: some View {
        let home = PanelKeys.home(state) != nil
        let keys = PanelKeys.shortcuts.filter { key in
            guard key != "cmd+t" || home, MenuModel.shortcut(key) != nil else { return false }
            switch PanelKeys.effect(key, state, focus: focus?.lit) {
            case .none, .segment, .pick: return false
            default: return PanelRegion.of(state) == .page
            }
        }
        ForEach(keys, id: \.self) { key in
            Button("") { _ = perform(key) }
                .keyboardShortcut(MenuModel.shortcut(key)!)
                .opacity(0)
                .allowsHitTesting(false)
                .accessibilityHidden(true)
        }
    }

    static func key(_ press: KeyPress) -> String? {
        let mods = press.modifiers
        guard !mods.contains(.option), !mods.contains(.control) else { return nil }
        let base: String
        switch press.key {
        case .escape: base = "esc"
        case .upArrow: base = "up"
        case .downArrow: base = "down"
        case .leftArrow: base = "left"
        case .rightArrow: base = "right"
        case .return: base = "enter"
        case .tab: base = mods.contains(.shift) ? "backtab" : "tab"
        case Self.backTab: base = "backtab"
        case .space: base = "space"
        default:
            guard !press.characters.isEmpty else { return nil }
            base = mods.contains(.command) ? press.key.character.lowercased() : press.characters
        }
        guard mods.contains(.command) else { return base }
        return (mods.contains(.shift) ? "shift+" : "") + "cmd+" + base
    }

    @MainActor
    func perform(_ key: String) -> Bool {
        let lit = focus?.lit
        switch PanelKeys.effect(key, state, focus: lit) {
        case .none, .segment, .pick:
            return false
        case .close:
            handlers.close()
        case .dismiss(.menu):
            handlers.toggleMenu()
        case .dismiss(.popover):
            if let qr = state.qr { handlers.toggleQR(qr) } else if let about = state.about { handlers.toggleAbout(about) }
        case .dismiss(.find):
            handlers.find(nil)
        case .dismiss(.sheet):
            handlers.confirm(nil)
        case .pop:
            handlers.back()
        case .home:
            handlers.home()
        case let .find(letter):
            handlers.find(FindQuery(letter))
        case let .move(step):
            if let find = state.find {
                handlers.find(find.moved(step, lines: PanelOrder.arrows(state).count))
            } else {
                focus = PanelOrder.moved(lit, by: step, in: PanelOrder.arrows(state)) ?? .panel
            }
        case let .walk(step):
            let region = PanelRegion.of(state)
            if region == .sheet || (region == .alert && walk.contains(.typed)) { return false }
            focus = PanelOrder.walked(lit, by: step, in: walk) ?? .panel
        case let .open(nav):
            go(nav, handlers)
        case let .run(id):
            guard let page = state.shown, let action = page.action(id) else { return false }
            verb(page, action, handlers)
        case let .answer(yes):
            guard PanelRegion.of(state) == .alert, let id = state.confirming, let action = PanelKeys.asking(state), let confirm = action.confirm else { return false }
            if yes, !AlertCard.allows(confirm, typed: state.answers[id]?[ActionRunner.typedKey] ?? "") { return false }
            handlers.confirm(nil)
            if yes { handlers.run(action, state.answers[id] ?? [:]) }
        }
        return true
    }
}
