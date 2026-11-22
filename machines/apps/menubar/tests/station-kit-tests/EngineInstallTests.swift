import XCTest
@testable import StationKit

struct AnsweringLauncher: Launcher {
    let calls: Recorded
    let launched: Launched
    func run(_ path: String, _ args: [String], environment: [String: String]) async -> Launched {
        calls.add([path] + args)
        return launched
    }
}

final class EngineInstallTests: XCTestCase {
    func read(_ launched: Launched) async -> (installed: Bool, calls: [[String]]) {
        let calls = Recorded()
        let installed = await EngineInstall.read(executable: "/u/bin/station", launcher: AnsweringLauncher(calls: calls, launched: launched))
        return (installed, calls.all)
    }

    func reading(_ state: String) -> Launched {
        Launched(code: 0, stderr: "", stdout: #"{"schema":"station.engine.read/v1","engine":{"state":"\#(state)","vms":0}}"#)
    }

    func testARunningEngineReadsAsInstalledWhoeverHoldsItsJob() async {
        let answer = await read(reading("running"))
        XCTAssertTrue(answer.installed)
        XCTAssertEqual(answer.calls, [["/u/bin/station", "engine", "--json"]])
    }

    func testAStoppedEngineReadsAsInstalled() async {
        let answer = await read(reading("stopped"))
        XCTAssertTrue(answer.installed)
    }

    func testAnEngineTheJobManagerDoesNotHoldReadsAsNotInstalled() async {
        let answer = await read(reading("not-installed"))
        XCTAssertFalse(answer.installed)
    }

    func testACommandThatCannotRunOrAnswersNothingReadsAsNotInstalled() async {
        let missing = await read(Launched(code: 127, stderr: "Error: /u/bin/station could not run", stdout: ""))
        XCTAssertFalse(missing.installed)
        let garbled = await read(Launched(code: 0, stderr: "", stdout: "not json"))
        XCTAssertFalse(garbled.installed)
    }
}
