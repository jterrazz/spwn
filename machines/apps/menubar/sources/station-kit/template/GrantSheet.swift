import SwiftUI

// invariant: SPEC A5 Grant a password: To the machine (its popup swapping
// to that machine's sheet), the Mac's passwords as a pick list; Cancel,
// then the button that names the password. Every word is the page's.
public struct GrantSheet: View {
    let sheet: SheetForm

    public init(page: Page, action: Action, given: [String: String], handlers: PopoverHandlers) {
        sheet = SheetForm(page: page, action: action, given: given, handlers: handlers)
    }

    public var body: some View { drawing.recorded(viewMark) }

    var viewMark: ViewMark? { ViewMark(.dialog, kit: "GrantSheet") }

    @ViewBuilder var drawing: some View {
        let answers = sheet.answers
        SheetCard(sheet.form.title ?? "", subtitle: sheet.form.subtitle) {
            VStack(alignment: .leading, spacing: S.rowGap) {
                ForEach(Array(sheet.fields.enumerated()), id: \.offset) { _, f in
                    switch f.kind {
                    case "select": sheet.select(f)
                    case "pick": FormRow(f.label) { PickRows(f, answers) { sheet.set(f, $0) } }
                    default: EmptyView()
                    }
                }
            }
        } footer: {
            HStack {
                PanelButton("Cancel", action: sheet.cancel).keyboardShortcut(.cancelAction)
                Spacer(minLength: 0)
                SheetPrimary(primary: answers.primary.map { Primary(when: $0.when, label: answers.fill($0.label ?? ""), danger: $0.danger) }, fallback: sheet.action.label ?? "", run: sheet.run)
            }
        }
    }
}
