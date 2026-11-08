import Foundation
import StationKit
import XCTest

final class AccountsTests: XCTestCase {
    func story(_ name: String) throws -> Page {
        try Page.decode(Data(contentsOf: Tree.stories.appendingPathComponent("pages/\(name).json")))
    }

    final class Clock: @unchecked Sendable {
        let lock = NSLock()
        var at: TimeInterval = 0
        var asked = 0
        var open = true
        func advance(_ by: TimeInterval) { lock.withLock { at += by } }
        func now() -> Date { lock.withLock { Date(timeIntervalSinceReferenceDate: at) } }
    }

    // invariant: SPEC B5.19: Sign in waits on the login Terminal runs: it
    // asks for the page again every 2 s and stops at 5 min, signed in or not.
    func testASignInIsAskedForEvery2sAndNoLongerThan5Min() async throws {
        let clock = Clock()
        let asked = await SignInWatch.run(since: clock.now(), now: { clock.now() }, sleep: { clock.advance($0) },
                                          open: { true }, refresh: { clock.lock.withLock { clock.asked += 1 } })
        XCTAssertEqual(asked, 149)
        XCTAssertEqual(clock.now().timeIntervalSinceReferenceDate, SignInWatch.most)
    }

    // invariant: the wait ends as soon as the page read again no longer
    // offers Sign in: the account is signed in.
    func testASignInStopsWaitingOnceThePageSaysSignedIn() async throws {
        let waiting = try story("machines.accounts.accounts")
        let signedIn = try story("machines.accounts.signed-in.accounts")
        let id = "sign-in:macbook-pro/npm"
        XCTAssertTrue(SignInWatch.open(id, on: waiting))
        XCTAssertFalse(SignInWatch.open(id, on: signedIn))
        let clock = Clock()
        let asked = await SignInWatch.run(since: clock.now(), now: { clock.now() }, sleep: { clock.advance($0) },
                                          open: { clock.lock.withLock { clock.asked < 3 } }, refresh: { clock.lock.withLock { clock.asked += 1 } })
        XCTAssertEqual(asked, 3)
        XCTAssertEqual(clock.now().timeIntervalSinceReferenceDate, 3 * SignInWatch.every)
    }

    // invariant: only a verb that opens Terminal and names its busy words
    // waits: Extend calls the engine and is done when it returns.
    func testOnlySignInWaitsOnWhatItStarted() throws {
        let page = try story("machines.accounts.accounts")
        let signIn = try XCTUnwrap(page.action("sign-in:macbook-pro/npm"))
        XCTAssertEqual(signIn.busy, "Waiting")
        XCTAssertTrue(SignInWatch.waits(signIn))
        let extend = try XCTUnwrap(page.action("ending:macbook-pro/linux-1/claude"))
        XCTAssertFalse(SignInWatch.waits(extend))
    }
}
