import SwiftUI

// invariant: SPEC A5 Lend an account: From its Mac, To the workspace (its
// popup swapping to that workspace's sheet), the Mac's accounts as a pick
// list, For with the day it ends under it; Cancel, then the button that
// names the account and the time. Every word is the page's.
public struct LendSheet: View {
    let sheet: SheetForm

    public init(page: Page, action: Action, given: [String: String], handlers: PopoverHandlers) {
        sheet = SheetForm(page: page, action: action, given: given, handlers: handlers)
    }

    public var body: some View {
        let answers = sheet.answers
        SheetCard(sheet.form.title ?? "", subtitle: sheet.form.subtitle) {
            VStack(alignment: .leading, spacing: S.rowGap) {
                ForEach(Array(sheet.fields.enumerated()), id: \.offset) { _, f in
                    switch f.kind {
                    case "select": sheet.select(f)
                    case "pick": FormRow(f.label) { PickRows(f, answers) { sheet.set(f, $0) } }
                    case "segmented":
                        let picked = (f.options ?? []).first { $0.value == answers.value(f) }
                        FormRow(f.label, caption: picked?.note) {
                            Segmented(options: (f.options ?? []).map { o in (o.label ?? "", o.value == answers.value(f), { sheet.set(f, o.value ?? "") }) })
                        }
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
