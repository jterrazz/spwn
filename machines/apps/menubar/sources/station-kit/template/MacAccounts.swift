import SwiftUI

// invariant: SPEC A4 A Mac's Accounts segment: each of its sections as
// its accounts card.
struct MacAccounts: View {
    let section: Section
    let page: Page
    let busy: Set<String>
    let run: (Action, Row?) -> Void

    var body: some View { AccountsSection(section, actions: page.actions ?? [], busy: busy, run: run) }
}
