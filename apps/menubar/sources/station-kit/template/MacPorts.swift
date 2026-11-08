import SwiftUI

// invariant: a Mac's Ports segment, its sections drawn plainly until its
// own lot draws them.
struct MacPorts: View {
    let section: Section

    var body: some View { SegmentSection(section: section) }
}
