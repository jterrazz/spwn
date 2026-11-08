import Foundation
import StationKit
import SwiftUI
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

    // invariant: a page that says signed in while the wait sleeps is not
    // asked for again.
    func testASignInSeenDuringTheWaitAsksNothingMore() async {
        let clock = Clock()
        let asked = await SignInWatch.run(since: clock.now(), now: { clock.now() },
                                          sleep: { clock.advance($0); clock.lock.withLock { clock.open = false } },
                                          open: { clock.lock.withLock { clock.open } }, refresh: { clock.lock.withLock { clock.asked += 1 } })
        XCTAssertEqual(asked, 0)
        XCTAssertEqual(clock.asked, 0)
    }

    // invariant: only a verb that opens Terminal and names its busy words
    // waits, its verb pending from the press, once however often it is
    // pressed: Extend calls the engine and is done when it returns.
    func testOnlySignInWaitsOnWhatItStarted() throws {
        let page = try story("machines.accounts.accounts")
        let signIn = try XCTUnwrap(page.action("sign-in:macbook-pro/npm"))
        XCTAssertEqual(signIn.busy, "Waiting")
        var pending: [String] = []
        XCTAssertEqual(SignInWatch.start(signIn, pending: &pending), "sign-in:macbook-pro/npm")
        XCTAssertEqual(pending, ["sign-in:macbook-pro/npm"])
        XCTAssertNil(SignInWatch.start(signIn, pending: &pending))
        XCTAssertEqual(pending, ["sign-in:macbook-pro/npm"])
        let extend = try XCTUnwrap(page.action("ending:macbook-pro/linux-1/claude"))
        XCTAssertNil(SignInWatch.start(extend, pending: &pending))
        XCTAssertEqual(pending, ["sign-in:macbook-pro/npm"])
    }

    // invariant: accounts.signing-in: while Terminal runs the login, the
    // row's verb reads its busy words, Waiting…, greyed, not Sign in. The
    // row alone is its golden, so a verb that kept its label fails it.
    @MainActor
    func testASignInRowReadsWaitingWhileTerminalRunsTheLogin() throws {
        let page = try story("machines.accounts.accounts")
        let row = try XCTUnwrap(page.sections?.flatMap { $0.items ?? [] }.compactMap(\.row).first { $0.verb == "sign-in:macbook-pro/npm" })
        let waiting = AnyView(NeedRow(row, actions: page.actions ?? [], busy: ["sign-in:macbook-pro/npm"]).frame(width: Token.Space.width))
        for scheme in [ColorScheme.dark, .light] {
            XCTAssertNil(try Golden.check("accounts.signing-in.row", waiting, scheme: scheme, desk: false))
        }
    }

    // invariant: accounts.signing-in drawn as its story draws it, from
    // ui.pending through the page to the row: the Accounts page and a
    // Mac's Accounts segment each draw something else once the press is
    // forgotten, so a page that drops the busy set fails here.
    @MainActor
    func testThePagesThatOfferSignInDrawItWaitingWhileItIsPending() throws {
        let id = "sign-in:macbook-pro/npm"
        for name in ["accounts.signing-in", "mac.this.accounts"] {
            let story = Stories[name]
            var pending = story.state
            pending.pending = [id]
            var idle = story.state
            idle.pending = []
            for scheme in [ColorScheme.dark, .light] {
                let waiting = try XCTUnwrap(Snapshot.render(AnyView(StoryView(story, state: pending)), scheme: scheme, desk: false))
                let signIn = try XCTUnwrap(Snapshot.render(AnyView(StoryView(story, state: idle)), scheme: scheme, desk: false))
                XCTAssertEqual([waiting.bitmap.width, waiting.bitmap.height], [signIn.bitmap.width, signIn.bitmap.height])
                guard waiting.bitmap.height == signIn.bitmap.height else { continue }
                XCTAssertGreaterThan(waiting.bitmap.differing(signIn.bitmap), 100, "\(name) \(scheme): Sign in draws the same pending or not")
            }
        }
    }
}
