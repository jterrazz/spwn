import SwiftUI

// invariant: SPEC A5's alert: a centred card over the scrim, the app's icon
// (or the orange warning), the question, the consequence, the losses, the
// name to type when the engine asks it typed, then Cancel and the action
// side by side. A gesture that loses something has Cancel as its default
// and its action in red; the action stays off until the name matches.
public struct AlertCard: View {
    let confirm: Confirm
    @Binding var typed: String
    let onAnswer: (Bool) -> Void
    @Environment(\.stationTokens) private var tokens
    @Environment(\.stationStill) private var still
    @FocusState private var focused: Bool

    public static let warn = "warn"

    public init(_ confirm: Confirm, typed: Binding<String>, onAnswer: @escaping (Bool) -> Void) {
        self.confirm = confirm
        _typed = typed
        self.onAnswer = onAnswer
    }

    public static func allows(_ confirm: Confirm, typed: String) -> Bool {
        guard let name = confirm.type else { return true }
        return typed.trimmingCharacters(in: .whitespaces) == name
    }

    // value: the button ↩ presses: the action of a gesture that loses
    // nothing, else Cancel, so the red button is never the default.
    public static func yesIsDefault(_ confirm: Confirm) -> Bool { confirm.danger != true && confirm.type == nil }

    var consequence: String? { confirm.lines?.first.map { Format.show($0) } }
    var losses: [String] { (confirm.lines ?? []).dropFirst().map { Format.show($0) } }

    public var body: some View {
        let a = Token.Metric.Alert.self
        VStack(spacing: 0) {
            icon.padding(.bottom, a.iconBelow)
            Text(confirm.question ?? "")
                .font(Token.Role.alertTitle.font).foregroundStyle(tokens.color.of(Token.Role.alertTitle.tone))
                .multilineTextAlignment(.center).fixedSize(horizontal: false, vertical: true)
            if let consequence {
                Text(consequence)
                    .font(Token.Role.alertBody.font).foregroundStyle(tokens.color.of(Token.Role.alertBody.tone))
                    .lineSpacing(a.lineSpacing)
                    .multilineTextAlignment(.center).fixedSize(horizontal: false, vertical: true)
                    .padding(.top, a.bodyAbove)
            }
            if !losses.isEmpty {
                VStack(alignment: .leading, spacing: a.lossGap) {
                    ForEach(Array(losses.enumerated()), id: \.offset) { _, loss in
                        Text("\(Token.Glyph.bullet) \(loss)").font(Token.Role.alertBody.font).foregroundStyle(tokens.color.of(Token.Role.alertBody.tone))
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, a.lossesInset).padding(.top, a.lossesAbove)
            }
            if let name = confirm.type { field(name).padding(.top, a.fieldAbove) }
            buttons.padding(.top, a.buttonsAbove)
        }
        .padding(a.padding)
        .frame(width: a.width)
        .background(RoundedRectangle(cornerRadius: a.radius, style: .continuous).fill(tokens.color.sheetSolid))
        .overlay(RoundedRectangle(cornerRadius: a.radius, style: .continuous).strokeBorder(tokens.color.hair, lineWidth: a.edge))
        .shadow(color: Token.alertShadow.opacity(a.shadowShare), radius: a.shadowRadius, y: a.shadowY)
        .accessibilityElement(children: .contain)
        .accessibilityAddTraits(.isModal)
        .accessibilityLabel(confirm.question ?? "")
    }

    @ViewBuilder
    var icon: some View {
        let a = Token.Metric.Alert.self
        if confirm.icon == Self.warn {
            Image(systemName: Token.Symbol.warn).font(.system(size: a.icon)).foregroundStyle(tokens.color.orange)
                .frame(width: a.icon, height: a.icon)
        } else {
            RoundedRectangle(cornerRadius: a.iconRadius, style: .continuous)
                .fill(LinearGradient(colors: [Token.AppIcon.top, Token.AppIcon.bottom], startPoint: .top, endPoint: .bottom))
                .overlay { Image(systemName: Token.Symbol.server).font(.system(size: a.iconMark, weight: .semibold)).foregroundStyle(Token.AppIcon.mark) }
                .frame(width: a.icon, height: a.icon)
                .shadow(color: Token.AppIcon.shadow, radius: Token.AppIcon.shadowRadius, y: Token.AppIcon.shadowY)
                .accessibilityHidden(true)
        }
    }

    func field(_ name: String) -> some View {
        let a = Token.Metric.Alert.self
        let shape = RoundedRectangle(cornerRadius: a.fieldRadius, style: .continuous)
        return VStack(alignment: .leading, spacing: a.labelBelow) {
            HStack(spacing: a.labelGap) {
                Text("Type").font(Token.Role.alertLabel.font).foregroundStyle(tokens.color.of(Token.Role.alertLabel.tone))
                Spacer(minLength: 0)
                Text(name).font(Token.Role.alertName.font).foregroundStyle(tokens.color.of(Token.Role.alertName.tone))
                Spacer(minLength: 0)
                Text("to confirm").font(Token.Role.alertLabel.font).foregroundStyle(tokens.color.of(Token.Role.alertLabel.tone))
            }
            SwiftUI.Group {
                if still {
                    HStack(spacing: 0) {
                        if !typed.isEmpty { Text(typed).foregroundStyle(tokens.color.t1) }
                        Rectangle().fill(tokens.color.accent).frame(width: a.caret, height: a.caretHeight)
                        if typed.isEmpty { Text(name).foregroundStyle(tokens.color.t3) }
                        Spacer(minLength: 0)
                    }
                } else {
                    TextField(name, text: $typed).textFieldStyle(.plain)
                        .focused($focused)
                        .onSubmit { if Self.allows(confirm, typed: typed) { onAnswer(true) } }
                        .onAppear { DispatchQueue.main.async { focused = true } }
                }
            }
            .font(.system(size: a.fieldSize))
            .lineLimit(1)
            .padding(.horizontal, a.fieldPadding)
            .frame(height: a.field)
            .background(shape.fill(tokens.color.q))
            .overlay(shape.strokeBorder(tokens.color.accent, lineWidth: Token.Metric.Alert.edge * 2))
            .overlay(shape.inset(by: -a.focusRing / 2).stroke(tokens.color.accent.opacity(a.focusShare), lineWidth: a.focusRing))
            .accessibilityLabel("Type \(name) to confirm")
        }
    }

    var buttons: some View {
        let a = Token.Metric.Alert.self
        let danger = confirm.danger == true || confirm.type != nil
        let allowed = Self.allows(confirm, typed: typed)
        let cancel = PanelButton(confirm.no ?? "Cancel", look: danger ? .tint : .plain, big: true, grow: true) { onAnswer(false) }
        let yes = PanelButton(confirm.yes ?? "OK", look: danger ? .destructive : .tint, big: true, grow: true) { onAnswer(true) }.disabled(!allowed)
        return HStack(spacing: a.buttonGap) {
            if Self.yesIsDefault(confirm) {
                cancel.keyboardShortcut(.cancelAction)
                yes.keyboardShortcut(.defaultAction)
            } else {
                cancel.keyboardShortcut(.defaultAction)
                yes
            }
        }
    }
}

// value: the alert of the gesture being confirmed, on the panel's alert
// layer, when the gesture asks a question and not a form: the scrim under
// it closes it.
public enum AlertLayer {
    @MainActor
    public static func layers(_ page: Page?, _ state: PanelState, _ handlers: PopoverHandlers) -> PanelLayers {
        guard let page, let id = state.confirming, let action = page.action(id), action.form == nil, let confirm = action.confirm else { return PanelLayers() }
        let typed = Binding(get: { state.answers[id]?[ActionRunner.typedKey] ?? "" }, set: { handlers.answer(id, ActionRunner.typedKey, $0) })
        let card = AlertCard(confirm, typed: typed) { yes in
            handlers.confirm(nil)
            if yes { handlers.run(action, state.answers[id] ?? [:]) }
        }
        return PanelLayers(alert: AnyView(ZStack {
            Color.clear.contentShape(Rectangle()).onTapGesture { handlers.confirm(nil) }
            card.transition(.movement { PopAppear(progress: $0, kind: .alert) })
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)))
    }
}
