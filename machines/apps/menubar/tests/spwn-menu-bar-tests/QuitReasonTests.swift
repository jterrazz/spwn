@testable import SpwnMenuBar
import XCTest

@MainActor
final class QuitReasonTests: XCTestCase {
    func testEachReasonLoginwindowSendsIsReadAndAnyOtherIsTheOwnersQuit() {
        let codes = ["logo", "rlgo", "rrst", "rest", "rsdn", "shut", "quit"].map(QuitEvents.fourCC)
        XCTAssertEqual(codes.map(QuitEvents.reason), [.logout, .logout, .restart, .restart, .shutdown, .shutdown, .user])
        XCTAssertEqual(QuitEvents.reason(0), .user)
        XCTAssertEqual(QuitEvents.fourCC("rlgo"), 0x726C_676F)
    }

    func testTheReasonsAreNumberedAsTheCEntryPointsCarryThem() {
        XCTAssertEqual([QuitReason.user, .logout, .restart, .shutdown].map(\.rawValue), [0, 1, 2, 3])
    }
}
