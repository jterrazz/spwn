import SwiftUI

// invariant: SPEC A5 Share a port: Of the workspace (its popup swapping
// to that workspace's sheet), then what listens there, each with Share,
// or the line that nothing does; nothing is typed, so the footer is
// Cancel alone.
public struct SharePortSheet: View {
    let sheet: SheetForm

    public init(page: Page, action: Action, given: [String: String], handlers: PopoverHandlers) {
        sheet = SheetForm(page: page, action: action, given: given, handlers: handlers)
    }

    public var body: some View {
        SheetCard(sheet.form.title ?? "", subtitle: sheet.form.subtitle) {
            VStack(alignment: .leading, spacing: S.rowGap) {
                ForEach(Array(sheet.fields.enumerated()), id: \.offset) { _, f in
                    switch f.kind {
                    case "select": sheet.select(f)
                    case "rows": ListenerRows(f, verb: { sheet.page.action($0)?.label ?? "" }, run: share)
                    default: EmptyView()
                    }
                }
            }
        } footer: {
            HStack {
                Spacer(minLength: 0)
                PanelButton("Cancel", action: sheet.cancel).keyboardShortcut(.cancelAction)
            }
        }
    }

    func share(_ id: String) {
        guard let a = sheet.page.action(id) else { return }
        sheet.handlers.confirm(nil)
        perform(a, sheet.handlers)
    }
}
