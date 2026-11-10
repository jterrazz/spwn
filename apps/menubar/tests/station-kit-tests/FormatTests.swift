import AppKit
@testable import StationKit
import SwiftUI
import XCTest

@MainActor
final class FormatTests: XCTestCase {
    func decoded<T: Decodable>(_ json: String) throws -> T { try JSONDecoder().decode(T.self, from: Data(json.utf8)) }

    // invariant: one duration format and one percent format across the
    // panel: `3 h 58 min`, `12 d`, `18 %`, whatever words the engine sent.
    func testOneDurationAndOnePercentFormat() throws {
        func shown(_ v: Value) -> String { Format.show(v).replacingOccurrences(of: Format.glued, with: " ") }
        XCTAssertEqual(shown(Value(kind: "text", text: "Awake until 14:11 · 3h 59m left")), "Awake until 14:11 · 3 h 59 min left")
        XCTAssertEqual(shown(Value(kind: "text", text: "up 3d 4h")), "up 3 d 4 h")
        XCTAssertEqual(shown(Value(kind: "text", text: "1h left")), "1 h left")
        XCTAssertEqual(shown(Value(kind: "text", text: "rates over 10s")), "rates over 10 s")
        XCTAssertEqual(shown(Value(kind: "text", text: "battery at 18%")), "battery at 18 %")
        XCTAssertEqual(shown(Value(kind: "text", text: "linux-1 · tail1234 · 2026.11.01 · 5173")), "linux-1 · tail1234 · 2026.11.01 · 5173")
        XCTAssertEqual(shown(try decoded(#"{"kind":"percent","n":180}"#) as Value), "18 %")
        XCTAssertEqual(shown(try decoded(#"{"kind":"percent","n":75}"#) as Value), "7.5 %")
        XCTAssertEqual(Format.show(try decoded(#"{"kind":"duration","n":14280}"#) as Value), "3 h 58 min")
        XCTAssertEqual(Format.show(Value(kind: "text", text: "3h 59m left")), ["3", "h", "59", "min", "left"].joined(separator: Format.glued), "a duration is kept whole on its line")
    }

    // invariant: a column of sizes says every figure in the unit of its
    // largest, as the CLI's: 4.4 GB, 2.0 GB, 0.6 GB, never 644.2 MB under
    // them.
    func testAColumnOfSizesSaysOneUnit() {
        XCTAssertEqual(Format.column([4_402_341_478, 2_040_109_465, 644_245_094]), ["4.4 GB", "2.0 GB", "0.6 GB"])
        XCTAssertEqual(Format.column([900_000, 12_000]), ["900.0 KB", "12.0 KB"])
        XCTAssertEqual(Format.column([]), [])
    }

    // invariant: memory is said as macOS says it, 16 GB for 16 GiB, the
    // same on Home as on Activity; a disk stays in powers of 1000.
    func ratio(_ n: Int64, _ of: Int64) -> Value {
        var v = Value(kind: "ratio")
        v.n = n
        v.of = of
        return v
    }

    func testMemoryIsSaidInGiBAsMacOSSaysIt() {
        XCTAssertEqual(Format.show(ratio(13_314_398_618, 17_179_869_184)), "12.4/16.0 GB")
        XCTAssertEqual(Format.show(ratio(16_106_127_360, 25_769_803_776)), "15.0/24.0 GB")
        XCTAssertEqual(Format.show(ratio(606_000_000_000, 994_000_000_000)), "606.0/994.0 GB")
    }

    // invariant: a duration is spaced as the vision's: `3 h 57 min`.
    func testDurationsAreSpacedAsTheVisions() {
        XCTAssertEqual(Format.duration(3 * 3600 + 57 * 60), "3 h 57 min")
        XCTAssertEqual(Format.duration(12 * 86400 + 3600), "12 d 1 h")
        XCTAssertEqual(Format.duration(40), "40 s")
    }

    // invariant: the engine-stopped page names the Mac as the engine does.
    func testHostNameDropsItsDomain() {
        XCTAssertEqual(Format.bareName("jterrazz-macbook-pro.local"), "jterrazz-macbook-pro")
        XCTAssertEqual(Format.bareName("mini"), "mini")
    }
}
