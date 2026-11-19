import SwiftUI

// invariant: SPEC A5 Hints: a title, two lines of why, each command the
// page gives in its box with Copy, then Done; the commands are the
// page's, never spelled here.
public struct HintSheet: View {
    let sheet: SheetForm

    public init(page: Page, action: Action, given: [String: String], handlers: PopoverHandlers) {
        sheet = SheetForm(page: page, action: action, given: given, handlers: handlers)
    }

    public var body: some View {
        SheetCard(sheet.form.title ?? "", subtitle: sheet.form.subtitle) {
            VStack(alignment: .leading, spacing: S.rowGap) {
                ForEach(Array(sheet.field("command").enumerated()), id: \.offset) { _, f in
                    CommandBox(f) { sheet.handlers.copy($0) }
                }
            }
        } footer: {
            HStack {
                Spacer(minLength: 0)
                SheetPrimary(primary: sheet.answers.primary, fallback: "", run: sheet.cancel)
            }
        }
    }
}
