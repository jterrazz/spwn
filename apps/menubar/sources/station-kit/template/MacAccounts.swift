import SwiftUI

// invariant: a Mac's Accounts segment, its sections drawn plainly until its
// own lot draws them.
struct MacAccounts: View {
    let section: Section

    var body: some View { SegmentSection(section: section) }
}
