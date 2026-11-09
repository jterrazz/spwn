import SwiftUI

// value: what every engine sheet does with its form: its answers as they
// stand, an answer set, a swap that opens the sibling form with the
// answers it shares, the To or Of row (its popup, or its one value), and
// the call run once the primary is pressed.
struct SheetForm {
    let page: Page
    let action: Action
    let given: [String: String]
    let handlers: PopoverHandlers

    var form: Form { action.form ?? Form() }
    var answers: Answers { Answers(form, given) }
    var id: String { action.id ?? "" }
    var fields: [Field] { form.fields ?? [] }

    func field(_ kind: String) -> [Field] { fields.filter { $0.kind == kind && answers.shown($0) } }

    func set(_ f: Field, _ value: String) { handlers.answer(id, f.key ?? "", value) }

    func pick(_ f: Field, _ value: String) {
        guard f.swaps == true else { return set(f, value) }
        guard value != id, let next = page.action(value) else { return }
        handlers.confirm(next)
        for (key, v) in given where answers.field(key)?.swaps != true { handlers.answer(value, key, v) }
    }

    // value: a select of one choice says it; of more, its popup.
    @ViewBuilder
    func select(_ f: Field) -> some View {
        let options = f.options ?? []
        if options.count == 1 {
            FormRow(f.label) { FixedValue(option: options[0], answers: answers) }
        } else {
            FormRow(f.label) { PopupField(f, answers) { pick(f, $0) } }
        }
    }

    func cancel() { handlers.confirm(nil) }

    func run() {
        handlers.confirm(nil)
        handlers.run(action, given)
    }
}
