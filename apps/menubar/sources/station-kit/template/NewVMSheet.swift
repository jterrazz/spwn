import SwiftUI

// invariant: SPEC A5 New VM: the engine's form drawn on the sheet's grid:
// On, then Options folded into one line and its caption, or open on Kind,
// Lifetime, System, Memory, Share and Name; a server's Name first and the
// line saying what its Mac's sleep does to it; Cancel, then Create and
// its arrow. Every word is the page's; On and Kind open the sibling form
// the page carries for that Mac and kind.
public struct NewVMSheet: View {
    let page: Page
    let action: Action
    let given: [String: String]
    let handlers: PopoverHandlers
    @Environment(\.stationTokens) private var tokens

    public init(page: Page, action: Action, given: [String: String], handlers: PopoverHandlers) {
        self.page = page
        self.action = action
        self.given = given
        self.handlers = handlers
    }

    var form: Form { action.form ?? Form() }
    var answers: Answers { Answers(form, given) }
    var id: String { action.id ?? "" }

    public var body: some View {
        SheetCard(form.title ?? "", subtitle: form.subtitle) {
            VStack(alignment: .leading, spacing: S.rowGap) {
                rows(form.fields ?? [], depth: 0)
                if let warn = form.warn {
                    Text(warn).font(Token.Role.sheetBody.font).foregroundStyle(tokens.color.orangeText)
                        .fixedSize(horizontal: false, vertical: true).padding(.top, S.warnTop)
                }
            }
        } footer: {
            HStack {
                PanelButton("Cancel") { handlers.confirm(nil) }.keyboardShortcut(.cancelAction)
                Spacer(minLength: 0)
                SplitButton(form.primary?.first?.label ?? action.label ?? "", items: (form.split ?? []).compactMap(page.action), run: { create(action) }, other: create)
            }
        }
    }

    func rows(_ fields: [Field], depth: Int) -> AnyView {
        AnyView(ForEach(Array(fields.enumerated()), id: \.offset) { _, f in
            if answers.shown(f) { row(f, depth: depth) }
        })
    }

    @ViewBuilder
    func row(_ f: Field, depth: Int) -> some View {
        switch f.kind {
        case "disclosure":
            let open = answers.isOpen(f)
            FormRow(f.label, caption: open ? nil : f.note.map(answers.fill)) {
                DisclosureRow(answers.fill(f.summary), open: open) { set(f, open ? "" : Answers.open) }
            }
            if open { rows(f.fields ?? [], depth: depth + 1) }
        case "select":
            FormRow(f.label) { PopupField(f, answers) { pick(f, $0) } }
        case "slider":
            FormRow(f.label) { MemorySlider(f, answers) { set(f, $0) }.padding(.top, (S.field - S.knob) / 2) }
        case "multiselect":
            FormRow(nil) { ChipPicker(f, answers) { set(f, $0) } }
        case "segmented":
            if depth > 1 {
                FormRow(nil) {
                    HStack(spacing: S.formGap) {
                        RoleText(f.label ?? "", .formLabel).frame(width: S.shareLabel, alignment: .leading)
                        segmented(f)
                    }
                }
            } else {
                FormRow(f.label) { segmented(f) }
            }
        default:
            FormRow(f.label, caption: answers.refused.map { _ in answers.fill(f.taken_hud ?? "") } ?? f.note) {
                FormText(text: Binding(get: { answers.value(f) }, set: { set(f, $0) }), submit: { create(action) })
            }
        }
    }

    func segmented(_ f: Field) -> some View {
        Segmented(options: (f.options ?? []).map { o in (o.label ?? "", o.value == answers.value(f), { pick(f, o.value ?? "") }) })
    }

    func set(_ f: Field, _ value: String) { handlers.answer(id, f.key ?? "", value) }

    // value: a swap opens the sibling form, the answers it shares kept.
    func pick(_ f: Field, _ value: String) {
        guard f.swaps == true else { return set(f, value) }
        guard value != id, let next = page.action(value) else { return }
        handlers.confirm(next)
        for (key, v) in given where answers.field(key)?.swaps != true { handlers.answer(value, key, v) }
    }

    func create(_ run: Action) {
        switch SheetOutcome.of(run, Answers(form, given)) {
        case .refused:
            return
        case .runs:
            handlers.confirm(nil)
            handlers.run(run, given)
        }
    }
}
