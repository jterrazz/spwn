import SwiftUI

// invariant: a Mac's Activity segment, its sections drawn plainly until its
// own lot draws them.
struct MacActivity: View {
    let section: Section

    var body: some View { SegmentSection(section: section) }
}
