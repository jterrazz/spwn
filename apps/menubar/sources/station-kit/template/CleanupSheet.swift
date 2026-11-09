import SwiftUI

// invariant: SPEC A5 Clean up (A2): the finished agents, then the empty
// throwaways, each checked, an agent its throwaway carries checked and
// dimmed; the note naming what stays; Cancel, then the button worded
// from what is checked, red once a throwaway is.
public struct CleanupSheet: View {
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
                    case "multiselect": CheckRows(f, answers) { sheet.set(f, $0) }
                    case "note": SheetLine(f.note ?? "")
                    default: EmptyView()
                    }
                }
            }
        } footer: {
            HStack {
                PanelButton("Cancel", action: sheet.cancel).keyboardShortcut(.cancelAction)
                Spacer(minLength: 0)
                SheetPrimary(primary: answers.primary, fallback: sheet.action.label ?? "", run: sheet.run)
            }
        }
    }
}
